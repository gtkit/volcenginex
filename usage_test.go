package volcenginex

import (
	"context"
	"errors"
	"testing"
)

func TestCountVolcMTChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "billing characters", text: "中A 1!?", want: 6},
		{name: "multibyte rune", text: "🙂", want: 1},
		{name: "newlines and tags", text: "<b>\n你好</b>", want: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CountVolcMTChar(tt.text); got != tt.want {
				t.Fatalf("CountVolcMTChar(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestGetUsageCommon(t *testing.T) {
	var gotInput map[string]any
	fake := &fakeTextAPI{usageRaw: func(_ context.Context, input *map[string]any) (*map[string]any, error) {
		gotInput = *input
		output := map[string]any{
			"Points": []any{
				map[string]any{"Timestamp": int32(10), "Value": int32(20)},
			},
		}
		return &output, nil
	}}
	service := &TextService{api: fake}
	usage, err := service.GetUsageCommon(t.Context(), 10, 20)
	if err != nil {
		t.Fatalf("GetUsageCommon returned error: %v", err)
	}
	if gotInput["From"] != int32(10) || gotInput["To"] != int32(20) || gotInput["Service"] != machineTranslationService {
		t.Fatalf("input = %#v", gotInput)
	}
	if len(usage.Points) != 1 || usage.Points[0].Timestamp != 10 || usage.Points[0].Value != 20 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestGetUsageCommonValidationAndProviderError(t *testing.T) {
	called := false
	providerErr := errors.New("provider failure")
	fake := &fakeTextAPI{usageRaw: func(context.Context, *map[string]any) (*map[string]any, error) {
		called = true
		return nil, providerErr
	}}
	service := &TextService{api: fake}
	if _, err := service.GetUsageCommon(t.Context(), 2, 1); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("range error = %v", err)
	}
	if called {
		t.Fatal("invalid range invoked provider")
	}
	if _, err := service.GetUsageCommon(nil, 1, 2); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil context error = %v", err)
	}
	if called {
		t.Fatal("nil context invoked provider")
	}
	if _, err := service.GetUsageCommon(t.Context(), 1, 2); err == nil || !errors.Is(err, providerErr) {
		t.Fatalf("provider error = %v", err)
	}
}
