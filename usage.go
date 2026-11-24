package volcenginex

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

const machineTranslationService = "text"

var usageLocation = time.FixedZone("UTC+8", 8*60*60)

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

// Usage returns text translation usage for the last thirty calendar days
// including today, in UTC+8.
func (s *TextService) Usage(ctx context.Context) (*Usage, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	now := time.Now().In(usageLocation)
	return s.getTextUsage(ctx, "GetUsage", now.AddDate(0, 0, -29), now)
}

// GetUsageCommon returns daily text translation usage for the UTC+8 dates
// spanned by the inclusive Unix-second range [from, to]. It is suitable for
// reconciliation and is not used to populate one-off translation results.
func (s *TextService) GetUsageCommon(ctx context.Context, from, to int32) (*Usage, error) {
	if ctx == nil || from > to {
		return nil, wrapSentinel("GetUsageCommon", ErrInvalidArgument, "context and usage range are invalid")
	}
	return s.getTextUsage(ctx, "GetUsageCommon", time.Unix(int64(from), 0), time.Unix(int64(to), 0))
}

func usageDate(t time.Time) int32 {
	year, month, day := t.In(usageLocation).Date()
	return int32(year*10000 + int(month)*100 + day)
}

func (s *TextService) getTextUsage(ctx context.Context, op string, from, to time.Time) (*Usage, error) {
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.GetUsageWithContext(ctx, &translate20250301.GetUsageInput{
		From:    new(usageDate(from)),
		To:      new(usageDate(to)),
		Service: new(machineTranslationService),
	}, retainNumericProviderError)
	var metadata *response.ResponseMetadata
	if out != nil {
		metadata = out.Metadata
	}
	if err := providerResponseError(ctx, op, metadata, err); err != nil {
		return nil, err
	}
	return usageOutput(out), nil
}
