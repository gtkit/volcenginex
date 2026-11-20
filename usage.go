package volcenginex

import (
	"context"
	"unicode/utf8"

	gtjson "github.com/gtkit/json/v2"
	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

const machineTranslationService = "translate"

// CountVolcMTChar counts text according to Volcengine machine translation
// billing semantics. Each Unicode rune counts as one character, including
// letters, digits, symbols, whitespace, newlines, tags, and placeholders.
func CountVolcMTChar(text string) int {
	return utf8.RuneCountInString(text)
}

func estimateDocumentCharacters(docs []DocInput) int {
	count := 0
	for _, doc := range docs {
		count += CountVolcMTChar(doc.Content)
	}
	return count
}

func usageCommonInput(from, to int32) *map[string]any {
	input := map[string]any{
		"From":    from,
		"To":      to,
		"Service": machineTranslationService,
	}
	return &input
}

func usageCommonOutput(raw *map[string]any) (*Usage, error) {
	if raw == nil {
		return usageOutput(nil), nil
	}
	data, err := gtjson.Marshal(*raw)
	if err != nil {
		return nil, clientError("GetUsageCommon", err)
	}
	output := new(translate20250301.GetUsageOutput)
	if err := gtjson.Unmarshal(data, output); err != nil {
		return nil, clientError("GetUsageCommon", err)
	}
	return usageOutput(output), nil
}

// GetUsageCommon returns machine translation usage for the inclusive Unix
// second range [from, to]. It is intended for scheduled or manual billing
// reconciliation and is not used to populate one-off translation results.
func (s *TextService) GetUsageCommon(ctx context.Context, from, to int32) (*Usage, error) {
	if ctx == nil || from > to {
		return nil, wrapSentinel("GetUsageCommon", ErrInvalidArgument, "context and usage range are invalid")
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	raw, err := s.api.GetUsageCommonWithContext(ctx, usageCommonInput(from, to))
	if err != nil {
		return nil, providerError("GetUsageCommon", err)
	}
	return usageCommonOutput(raw)
}
