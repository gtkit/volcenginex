package volcenginex

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
)

const downloadImageOp = "DownloadImage"

var (
	globalIPv6         = netip.MustParsePrefix("2000::/3")
	blockedImageRanges = [...]netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"),
		netip.MustParsePrefix("3fff::/20"),
	}
)

func newImageURLClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		// Use a separate transport so source requests do not inherit SDK headers,
		// cookies or proxies that can bypass public-address checks.
		Transport: &http.Transport{
			DialContext:            dialImageURL,
			ForceAttemptHTTP2:      true,
			TLSHandshakeTimeout:    5 * time.Second,
			MaxIdleConns:           16,
			IdleConnTimeout:        30 * time.Second,
			MaxResponseHeaderBytes: 1 << 20,
		},
		CheckRedirect: checkImageRedirect,
	}
}

func publicImageIP(addr netip.Addr) bool {
	ip := addr.Unmap()
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && ip.Zone() == "" &&
		(ip.Is4() || globalIPv6.Contains(ip)) &&
		!slices.ContainsFunc(blockedImageRanges[:], func(prefix netip.Prefix) bool { return prefix.Contains(ip) })
}

func dialImageURL(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, ErrInvalidArgument
	}
	for _, ip := range addresses {
		if !publicImageIP(ip) {
			return nil, ErrInvalidArgument
		}
	}
	dialer := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	var dialErr error
	// Dial the resolved IP directly to avoid a second DNS lookup.
	for _, ip := range addresses {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = errors.Join(dialErr, err)
	}
	return nil, dialErr
}

func validImageURL(u *url.URL) bool {
	return (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

func checkImageRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > 5 || !validImageURL(req.URL) ||
		(len(via) > 0 && via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme != "https") {
		return ErrInvalidArgument
	}
	req.Header.Del("Referer")
	return nil
}

func (s *ImageService) fetchImage(ctx context.Context, raw string) (data []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(raw), nil)
	if err != nil || !validImageURL(req.URL) {
		return nil, wrapSentinel(downloadImageOp, ErrInvalidArgument, "image URL must use HTTP or HTTPS without user information")
	}
	resp, err := s.urlClient.Do(req)
	if err != nil {
		return nil, imageDownloadError(err, "could not fetch image")
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil && err == nil {
			err = imageDownloadError(closeErr, "could not close image response")
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &Error{
			Op: downloadImageOp, Code: "ImageDownloadFailed", Message: "image server returned an unsuccessful response",
			HTTPStatus: resp.StatusCode, Retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
		}
	}
	if resp.ContentLength > s.maxBytes {
		return nil, ErrImageTooLarge
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, min(s.maxBytes, math.MaxInt64-1)+1))
	if err != nil {
		return nil, imageDownloadError(err, "could not read image response")
	}
	if int64(len(data)) > s.maxBytes {
		return nil, ErrImageTooLarge
	}
	return data, nil
}

func imageDownloadError(err error, message string) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ErrInvalidArgument):
		return wrapSentinel(downloadImageOp, ErrInvalidArgument, "image URL destination or redirect is not allowed")
	default:
		// Transport errors can contain signed URL parameters.
		return &Error{Op: downloadImageOp, Code: "ImageDownloadFailed", Message: message, Retryable: retryableError(err)}
	}
}
