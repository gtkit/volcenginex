package volcenginex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gtkit/httpc"
)

func download(ctx context.Context, client *httpc.Client, url, destPath string) (written int64, err error) {
	if ctx == nil || client == nil || url == "" || destPath == "" {
		return 0, ErrInvalidArgument
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, clientError("Download", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, providerError("Download", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		err = errors.Join(err, closeErr)
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, &Error{Op: "Download", Code: resp.Status, HTTPStatus: resp.StatusCode, Retryable: resp.StatusCode == 429 || resp.StatusCode >= 500}
	}
	dir := filepath.Dir(destPath)
	part, err := os.CreateTemp(dir, "."+filepath.Base(destPath)+".*.part")
	if err != nil {
		return 0, clientError("Download", err)
	}
	partPath := part.Name()
	removePart := true
	partClosed := false
	defer func() {
		if !partClosed {
			err = errors.Join(err, part.Close())
		}
		if removePart {
			if removeErr := os.Remove(partPath); err == nil {
				err = removeErr
			}
		}
	}()
	written, err = io.Copy(part, resp.Body)
	if err != nil {
		if resp.ContentLength >= 0 {
			return written, fmt.Errorf("%w: %v", ErrIncompleteDownload, err)
		}
		return written, providerError("Download", err)
	}
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return written, fmt.Errorf("%w: expected %d bytes, wrote %d", ErrIncompleteDownload, resp.ContentLength, written)
	}
	if err := part.Close(); err != nil {
		return written, clientError("Download", err)
	}
	partClosed = true
	if err := os.Rename(partPath, destPath); err != nil {
		return written, clientError("Download", err)
	}
	removePart = false
	return written, nil
}
