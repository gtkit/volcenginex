package volcenginex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

type fakeTextAPI struct {
	translate20250301.TRANSLATE20250301API
	translate func(context.Context, *translate20250301.TranslateTextInput) (*translate20250301.TranslateTextOutput, error)
	detect    func(context.Context, *translate20250301.LangDetectInput) (*translate20250301.LangDetectOutput, error)
	usage     func(context.Context, *translate20250301.GetUsageInput) (*translate20250301.GetUsageOutput, error)
}

func (f *fakeTextAPI) TranslateTextWithContext(ctx context.Context, input *translate20250301.TranslateTextInput, _ ...request.Option) (*translate20250301.TranslateTextOutput, error) {
	return f.translate(ctx, input)
}

func (f *fakeTextAPI) LangDetectWithContext(ctx context.Context, input *translate20250301.LangDetectInput, _ ...request.Option) (*translate20250301.LangDetectOutput, error) {
	return f.detect(ctx, input)
}

func (f *fakeTextAPI) GetUsageWithContext(ctx context.Context, input *translate20250301.GetUsageInput, _ ...request.Option) (*translate20250301.GetUsageOutput, error) {
	return f.usage(ctx, input)
}

func TestTextTranslateBatchesAndPreservesOrder(t *testing.T) {
	var mu sync.Mutex
	var calls [][]string
	fake := &fakeTextAPI{}
	fake.translate = func(_ context.Context, input *translate20250301.TranslateTextInput) (*translate20250301.TranslateTextOutput, error) {
		batch := make([]string, len(input.TextList))
		for i, value := range input.TextList {
			batch[i] = *value
		}
		mu.Lock()
		calls = append(calls, batch)
		mu.Unlock()
		items := make([]*translate20250301.TranslationListForTranslateTextOutput, len(batch))
		for i, value := range batch {
			items[i] = &translate20250301.TranslationListForTranslateTextOutput{Translation: new("translated:" + value)}
		}
		return &translate20250301.TranslateTextOutput{
			Metadata:        &response.ResponseMetadata{RequestId: "request-1"},
			TranslationList: items,
		}, nil
	}
	service := &TextService{api: fake, batchSize: 16}
	texts := make([]string, 17)
	for i := range texts {
		texts[i] = string(rune('a' + i))
	}
	result, err := service.Translate(t.Context(), TextTranslateRequest{TargetLanguage: "en", TextList: texts})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}
	if result.RequestID != "request-1" {
		t.Fatalf("request ID = %q", result.RequestID)
	}
	for i, item := range result.Items {
		want := "translated:" + texts[i]
		if item.Index != i || item.Translation != want {
			t.Errorf("item %d = %#v, want translation %q", i, item, want)
		}
	}
}

func TestTextTranslateSkipsBlankItems(t *testing.T) {
	called := false
	fake := &fakeTextAPI{translate: func(context.Context, *translate20250301.TranslateTextInput) (*translate20250301.TranslateTextOutput, error) {
		called = true
		return nil, nil
	}}
	service := &TextService{api: fake, batchSize: 16}
	result, err := service.Translate(t.Context(), TextTranslateRequest{TargetLanguage: "en", TextList: []string{" ", "\n"}})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if called {
		t.Fatal("blank items triggered a remote call")
	}
	if len(result.Items) != 2 || result.Items[0].Translation != "" || result.Items[1].Translation != "" {
		t.Fatalf("unexpected blank result: %#v", result.Items)
	}
}

func TestTextTranslatePartialFailure(t *testing.T) {
	var calls atomic.Int32
	fake := &fakeTextAPI{translate: func(_ context.Context, input *translate20250301.TranslateTextInput) (*translate20250301.TranslateTextOutput, error) {
		call := calls.Add(1)
		if call == 1 {
			return nil, errors.New("temporary failure")
		}
		return &translate20250301.TranslateTextOutput{
			TranslationList: []*translate20250301.TranslationListForTranslateTextOutput{{Translation: new("ok")}},
		}, nil
	}}
	service := &TextService{api: fake, batchSize: 1, partialFailure: true}
	result, err := service.Translate(t.Context(), TextTranslateRequest{TargetLanguage: "en", TextList: []string{"one", "two"}})
	if err != nil {
		t.Fatalf("partial Translate returned error: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("got %d partial errors, want 1", len(result.Errors))
	}
}

func TestTextTranslateRejectsMissingTarget(t *testing.T) {
	service := &TextService{api: &fakeTextAPI{}}
	_, err := service.Translate(t.Context(), TextTranslateRequest{TextList: []string{"hello"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestTextDetectAndUsage(t *testing.T) {
	fake := &fakeTextAPI{}
	fake.detect = func(_ context.Context, input *translate20250301.LangDetectInput) (*translate20250301.LangDetectOutput, error) {
		if len(input.TextList) != 1 || *input.TextList[0] != "hello" {
			t.Fatalf("unexpected detect input: %#v", input)
		}
		return &translate20250301.LangDetectOutput{DetectedLanguageList: []*translate20250301.DetectedLanguageListForLangDetectOutput{{Language: new("en"), Confidence: new(0.9)}}}, nil
	}
	fake.usage = func(_ context.Context, input *translate20250301.GetUsageInput) (*translate20250301.GetUsageOutput, error) {
		if input.Service == nil || *input.Service != "translate" {
			t.Fatalf("unexpected usage service: %#v", input.Service)
		}
		return &translate20250301.GetUsageOutput{Points: []*translate20250301.PointForGetUsageOutput{{Timestamp: new(int32(1)), Value: new(int32(2))}}}, nil
	}
	service := &TextService{api: fake, batchSize: 16}
	detected, err := service.Detect(t.Context(), []string{" ", "hello"})
	if err != nil || len(detected) != 1 || detected[0].Index != 1 || detected[0].Language != "en" {
		t.Fatalf("detected=%#v err=%v", detected, err)
	}
	usage, err := service.Usage(t.Context())
	if err != nil || len(usage.Points) != 1 || usage.Points[0].Value != 2 {
		t.Fatalf("usage=%#v err=%v", usage, err)
	}
}
