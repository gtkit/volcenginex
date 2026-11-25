package volcenginex

import (
	"errors"
	"net"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"
)

func TestProviderErrorCarriesProviderFields(t *testing.T) {
	provider := volcengineerr.New("InvalidParameter", "bad input", nil)
	err := providerError("TranslateText", volcengineerr.NewRequestFailure(provider, 400, "rid-1"))
	var normalized *Error
	if !errors.As(err, &normalized) {
		t.Fatalf("error type = %T", err)
	}
	if normalized.Code != "InvalidParameter" || normalized.Message != "bad input" || normalized.RequestID != "rid-1" || normalized.Retryable {
		t.Fatalf("normalized = %#v", normalized)
	}
}

func TestErrorSentinelAndRetryable(t *testing.T) {
	wrapped := wrapSentinel("op", ErrInvalidArgument, "invalid argument")
	if !errors.Is(wrapped, ErrInvalidArgument) || wrapped.Error() == "" {
		t.Fatalf("sentinel error = %v", wrapped)
	}
	if !errors.Is((&Error{Err: ErrInvalidArgument}), ErrInvalidArgument) {
		t.Fatal("Error.Is did not preserve sentinel")
	}
	if !retryableError(&net.DNSError{IsTimeout: true}) {
		t.Fatal("timeout was not retryable")
	}
	if requestIDFromMetadata(&response.ResponseMetadata{RequestId: "rid"}) != "rid" || requestIDFromMetadata(nil) != "" {
		t.Fatal("request ID extraction failed")
	}
}
