package volcenginex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/volcengine/custom"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
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

// CodeN returns the numeric provider error code when supplied by a machine translation endpoint.
func (e *Error) CodeN() (int, bool) {
	if e != nil {
		if numeric, ok := errors.AsType[*numericProviderError](e.Err); ok {
			return numeric.code, true
		}
	}
	return 0, false
}

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

type numericProviderError struct {
	volcengineerr.RequestFailure
	code int
}

func (e *numericProviderError) Unwrap() error { return e.RequestFailure }

func newNumericProviderError(metadata *response.ResponseMetadata, status int, simple *bool) error {
	code := metadata.Error.Code
	if code == "" {
		code = strconv.Itoa(metadata.Error.CodeN)
	}
	cause := volcengineerr.New(code, metadata.Error.Message, nil)
	return &numericProviderError{
		RequestFailure: volcengineerr.NewRequestFailure(cause, status, metadata.RequestId, simple),
		code:           metadata.Error.CodeN,
	}
}

func retainNumericProviderError(req *request.Request) {
	req.Config.CustomerUnmarshalError = func(_ context.Context, _ custom.RequestMetadata, raw response.VolcengineResponse) error {
		metadata := raw.ResponseMetadata
		if metadata == nil || metadata.Error == nil || metadata.Error.CodeN == 0 {
			return nil
		}
		return newNumericProviderError(metadata, req.HTTPResponse.StatusCode, req.Config.SimpleError)
	}
}

func providerResponseError(ctx context.Context, op string, metadata *response.ResponseMetadata, err error) error {
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	// The SDK leaves a CodeN-only error in the metadata on HTTP 200.
	if err == nil && metadata != nil && metadata.Error != nil && metadata.Error.CodeN != 0 {
		err = newNumericProviderError(metadata, metadata.HTTPCode, nil)
	}
	return providerError(op, err)
}

func invalidProviderResponse(op string, metadata *response.ResponseMetadata, message string) error {
	err := &Error{Op: op, Code: "InvalidResponse", Message: message, Retryable: true}
	if metadata != nil {
		err.RequestID = metadata.RequestId
		err.HTTPStatus = metadata.HTTPCode
	}
	return err
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
