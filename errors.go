package volcenginex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"
)

var (
	// ErrMissingCredentials means access key or secret key is absent.
	ErrMissingCredentials = errors.New("volcenginex: missing credentials")
	// ErrInvalidArgument means a client-side argument is invalid.
	ErrInvalidArgument = errors.New("volcenginex: invalid argument")
	// ErrImageTooLarge means an image exceeds the configured byte limit.
	ErrImageTooLarge = errors.New("volcenginex: image too large")
	// ErrTooManyDocs means a task contains more than ten documents.
	ErrTooManyDocs = errors.New("volcenginex: too many documents")
	// ErrFileTooLarge means a document exceeds 200 MB.
	ErrFileTooLarge = errors.New("volcenginex: file too large")
	// ErrTaskTooLarge means a task exceeds 2000 MB total.
	ErrTaskTooLarge = errors.New("volcenginex: task too large")
	// ErrDocTypeUnsupported means the source type cannot be parsed online.
	ErrDocTypeUnsupported = errors.New("volcenginex: document type unsupported")
	// ErrTaskAborted means a document task reached a stopped or deleted state.
	ErrTaskAborted = errors.New("volcenginex: document task aborted")
	// ErrPollExhausted means repeated polling failures exceeded the limit.
	ErrPollExhausted = errors.New("volcenginex: polling exhausted")
	// ErrTOSNotConfigured means TOS upload configuration is absent.
	ErrTOSNotConfigured = errors.New("volcenginex: TOS is not configured")
	// ErrIncompleteDownload means response bytes differ from Content-Length.
	ErrIncompleteDownload = errors.New("volcenginex: incomplete download")
)

// Error is the normalized provider or transport error returned by the SDK.
type Error struct {
	Op         string
	Code       string
	Message    string
	RequestID  string
	HTTPStatus int
	Retryable  bool
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{e.Op}
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying provider or transport error.
func (e *Error) Unwrap() error { return e.Err }

// Is preserves sentinel matching through a normalized error.
func (e *Error) Is(target error) bool {
	if target == nil {
		return e == nil
	}
	return errors.Is(e.Err, target)
}

func clientError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Code: "ClientError", Message: err.Error(), Err: err}
}

func providerError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	e := &Error{Op: op, Err: err, Retryable: retryableError(err)}
	if requestFailure, ok := errors.AsType[volcengineerr.RequestFailure](err); ok {
		e.HTTPStatus = requestFailure.StatusCode()
		e.RequestID = requestFailure.RequestID()
		e.Retryable = e.Retryable || e.HTTPStatus >= 500 || e.HTTPStatus == 429
	}
	if provider, ok := errors.AsType[volcengineerr.Error](err); ok {
		e.Code = provider.Code()
		e.Message = provider.Message()
		if e.Message == "" {
			e.Message = provider.Error()
		}
	}
	if e.Message == "" {
		e.Message = "provider request failed"
	}
	return e
}

func retryableError(err error) bool {
	if err == nil {
		return false
	}
	if requestFailure, ok := errors.AsType[volcengineerr.RequestFailure](err); ok {
		status := requestFailure.StatusCode()
		return status == 429 || status >= 500
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func requestIDFromMetadata(metadata *response.ResponseMetadata) string {
	if metadata == nil {
		return ""
	}
	return metadata.RequestId
}

func wrapSentinel(op string, sentinel error, message string) error {
	return &Error{Op: op, Code: "InvalidArgument", Message: message, Err: sentinel}
}

func formatLimitError(sentinel error, format string, args ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, args...))
}
