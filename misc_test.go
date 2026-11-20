package volcenginex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/i18nopenapi"
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
	fake := &fakeDocumentAPI{details: []*i18nopenapi.DocumentTaskDetailOutput{{TaskInfo: &i18nopenapi.TaskInfoForDocumentTaskDetailOutput{State: new(int32(3)), TaskId: new(int32(9))}}}}
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
}
