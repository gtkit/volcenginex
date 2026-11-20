package volcenginex

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMiscBranches(t *testing.T) {
	if err := waitDuration(t.Context(), 0); err != nil {
		t.Fatalf("zero wait error: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if !errors.Is(waitDuration(ctx, time.Second), context.Canceled) {
		t.Fatal("canceled wait did not return context error")
	}
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

func TestDocumentTranslateFileWithExistingDocs(t *testing.T) {
	fake := &fakeDocumentAPI{details: []*documentTaskDetailOutput{{TaskInfo: &documentTaskDetailTaskInfo{State: new(int32(3)), ID: new(int32(9))}}}}
	service := &DocumentService{
		client: &Client{tos: &TOSService{}},
		api:    fake,
		sleep:  func(context.Context, time.Duration) error { return nil },
	}
	result, err := service.TranslateFile(t.Context(), FileTranslateRequest{
		ProjectID: 1,
		TaskName:  "task",
		Docs:      []DocInput{{Name: "source.md", Type: DocTypeMd, URL: "https://example/source.md"}},
	})
	if err != nil || result == nil || result.Detail.State != TaskDone {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if result.EstimatedCharacterCount != 0 {
		t.Fatalf("unexpected uploaded document estimate = %d", result.EstimatedCharacterCount)
	}
}

func TestDocumentEstimateUsesInlineContent(t *testing.T) {
	fake := &fakeDocumentAPI{details: []*documentTaskDetailOutput{{TaskInfo: &documentTaskDetailTaskInfo{State: new(int32(3)), ID: new(int32(9))}}}}
	service := &DocumentService{
		client: &Client{tos: &TOSService{}},
		api:    fake,
		sleep:  func(context.Context, time.Duration) error { return nil },
	}
	result, err := service.TranslateFile(t.Context(), FileTranslateRequest{
		ProjectID: 1,
		TaskName:  "task",
		Docs:      []DocInput{{Name: "inline.md", Type: DocTypeMd, Content: "你好 🙂"}},
	})
	if err != nil {
		t.Fatalf("TranslateFile returned error: %v", err)
	}
	if result.EstimatedCharacterCount != CountVolcMTChar("你好 🙂") {
		t.Fatalf("estimate = %d, want %d", result.EstimatedCharacterCount, CountVolcMTChar("你好 🙂"))
	}
}
