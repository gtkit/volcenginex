package volcenginex

import (
	"errors"
	"testing"
)

func TestMiscBranches(t *testing.T) {
	base := errors.New("base")
	normalized := &Error{Err: base}
	if !errors.Is(normalized, base) || normalized.Unwrap() != base {
		t.Fatal("error unwrap failed")
	}
	discard := loggerOrDiscard(nil)
	discard.Info(t.Context(), "info")
	discard.Error(t.Context(), "error")
	if _, err := (&Client{}).Download(t.Context(), "", ""); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid client download error = %v", err)
	}
}
