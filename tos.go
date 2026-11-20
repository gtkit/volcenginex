package volcenginex

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"github.com/gtkit/httpc"
	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

// TOSService uploads source documents through TOS pre-signed URLs.
type TOSService struct {
	httpClient *httpc.Client
	bucket     string
	prefix     string
	signer     func(*tos.PreSignedURLInput) (*tos.PreSignedURLOutput, error)
}

func newTOSService(hc *httpc.Client, cfg TOSConfig, defaultAK, defaultSK string) (*TOSService, error) {
	ak, sk := cfg.AccessKey, cfg.SecretKey
	if ak == "" {
		ak = defaultAK
	}
	if sk == "" {
		sk = defaultSK
	}
	if ak == "" || sk == "" {
		return nil, ErrMissingCredentials
	}
	region := cfg.Region
	if region == "" {
		region = defaultRegion
	}
	client, err := tos.NewClientV2(cfg.Endpoint,
		tos.WithCredentials(tos.NewStaticCredentials(ak, sk)),
		tos.WithRegion(region),
	)
	if err != nil {
		return nil, clientError("TOSClient", err)
	}
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "volcenginex"
	}
	if prefix != "" {
		prefix += "/"
	}
	return &TOSService{
		httpClient: hc,
		bucket:     cfg.Bucket,
		prefix:     prefix,
		signer:     client.PreSignedURL,
	}, nil
}

// Upload uploads a local file without loading it fully into memory.
func (s *TOSService) Upload(ctx context.Context, localPath string) (*UploadedObject, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	// #nosec G304 -- the caller explicitly supplies the local source path.
	file, err := os.Open(localPath)
	if err != nil {
		return nil, clientError("TOSUpload", err)
	}
	info, err := file.Stat()
	if err != nil {
		closeErr := file.Close()
		return nil, errors.Join(clientError("TOSUpload", err), closeErr)
	}
	result, uploadErr := s.UploadReader(ctx, filepath.Base(localPath), readerOnly{Reader: file}, info.Size())
	closeErr := file.Close()
	return result, errors.Join(uploadErr, closeErr)
}

type readerOnly struct{ io.Reader }

// UploadBytes uploads bytes through a TOS pre-signed PUT URL.
func (s *TOSService) UploadBytes(ctx context.Context, name string, data []byte) (*UploadedObject, error) {
	if ctx == nil || strings.TrimSpace(name) == "" {
		return nil, ErrInvalidArgument
	}
	key, signed, err := s.presign(name)
	if err != nil {
		return nil, err
	}
	headers, _, status, err := s.httpClient.RequestRawWithHeader(
		trace.WithRequestID(ctx, trace.NewRequestID()),
		http.MethodPut,
		signed.SignedUrl,
		signed.SignedHeader,
		httpc.RawBody(data),
	)
	if err != nil {
		return nil, providerError("TOSUpload", err)
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, &Error{Op: "TOSUpload", Code: http.StatusText(status), HTTPStatus: status, Retryable: status == 429 || status >= 500}
	}
	return &UploadedObject{URL: signed.SignedUrl, ObjectKey: key, ETag: headers.Get("ETag"), Size: int64(len(data))}, nil
}

// UploadReader streams a reader through a TOS pre-signed PUT URL.
func (s *TOSService) UploadReader(ctx context.Context, name string, reader io.Reader, size int64) (*UploadedObject, error) {
	if ctx == nil || reader == nil || size < 0 || strings.TrimSpace(name) == "" {
		return nil, ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, signed, err := s.presign(name)
	if err != nil {
		return nil, clientError("TOSUpload", err)
	}
	if signed == nil || signed.SignedUrl == "" {
		return nil, clientError("TOSUpload", errors.New("empty pre-signed URL"))
	}
	req, err := http.NewRequestWithContext(trace.WithRequestID(ctx, trace.NewRequestID()), http.MethodPut, signed.SignedUrl, reader)
	if err != nil {
		return nil, clientError("TOSUpload", err)
	}
	req.ContentLength = size
	for header, value := range signed.SignedHeader {
		req.Header.Set(header, value)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, providerError("TOSUpload", err)
	}
	_, copyErr := io.Copy(io.Discard, resp.Body)
	closeErr := resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &Error{Op: "TOSUpload", Code: resp.Status, HTTPStatus: resp.StatusCode, Retryable: resp.StatusCode == 429 || resp.StatusCode >= 500}
	}
	if err := errors.Join(copyErr, closeErr); err != nil {
		return nil, providerError("TOSUpload", err)
	}
	return &UploadedObject{URL: signed.SignedUrl, ObjectKey: key, ETag: resp.Header.Get("ETag"), Size: size}, nil
}

func (s *TOSService) presign(name string) (string, *tos.PreSignedURLOutput, error) {
	if s.signer == nil {
		return "", nil, ErrTOSNotConfigured
	}
	key := s.prefix + time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.New().String() + filepath.Ext(name)
	signed, err := s.signer(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodType(http.MethodPut),
		Bucket:     s.bucket,
		Key:        key,
		Expires:    int64((30 * time.Minute) / time.Second),
	})
	if err != nil {
		return "", nil, clientError("TOSUpload", err)
	}
	if signed == nil || signed.SignedUrl == "" {
		return "", nil, clientError("TOSUpload", errors.New("empty pre-signed URL"))
	}
	return key, signed, nil
}
