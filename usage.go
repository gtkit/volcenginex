package volcenginex

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

// UsageService identifies the machine translation service whose usage is queried.
type UsageService string

const (
	// UsageServiceText selects text translation usage.
	UsageServiceText UsageService = "text"
	// UsageServiceImage selects image translation usage.
	UsageServiceImage UsageService = "image"
	// UsageServiceAudio selects audio and video file translation usage.
	UsageServiceAudio UsageService = "audio"
)

var usageLocation = time.FixedZone("UTC+8", 8*60*60)

// CountVolcMTChar counts text according to Volcengine machine translation
// billing semantics. Each Unicode rune counts as one character, including
// letters, digits, symbols, whitespace, newlines, tags, and placeholders.
func CountVolcMTChar(text string) int {
	return utf8.RuneCountInString(text)
}

// GetUsage returns provider usage points for the selected translation service.
// The range includes both Beijing calendar dates covered by from and to.
// Invalid arguments return ErrInvalidArgument before sending a request.
func (c *Client) GetUsage(ctx context.Context, service UsageService, from, to time.Time) (*Usage, error) {
	const op = "GetUsage"
	if ctx == nil {
		return nil, wrapSentinel(op, ErrInvalidArgument, "context is required")
	}
	switch service {
	case UsageServiceText, UsageServiceImage, UsageServiceAudio:
	default:
		return nil, wrapSentinel(op, ErrInvalidArgument, "unsupported usage service")
	}

	from, to = from.In(usageLocation), to.In(usageLocation)
	if from.Year() < 1000 || from.Year() > 9999 || to.Year() < 1000 || to.Year() > 9999 {
		return nil, wrapSentinel(op, ErrInvalidArgument, "usage dates require four-digit years")
	}
	firstDate, lastDate := usageDate(from), usageDate(to)
	if firstDate > lastDate {
		return nil, wrapSentinel(op, ErrInvalidArgument, "usage start date is after end date")
	}
	return getUsage(ctx, c.usageAPI, op, service, firstDate, lastDate)
}

// Usage returns text translation usage for the last thirty calendar days
// including today, in UTC+8.
func (s *TextService) Usage(ctx context.Context) (*Usage, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	now := time.Now().In(usageLocation)
	return getUsage(ctx, s.api, "GetUsage", UsageServiceText, usageDate(now.AddDate(0, 0, -29)), usageDate(now))
}

// GetUsageCommon returns daily text translation usage for the UTC+8 dates
// spanned by the inclusive Unix-second range [from, to]. It is suitable for
// reconciliation and is not used to populate one-off translation results.
func (s *TextService) GetUsageCommon(ctx context.Context, from, to int32) (*Usage, error) {
	if ctx == nil || from > to {
		return nil, wrapSentinel("GetUsageCommon", ErrInvalidArgument, "context and usage range are invalid")
	}
	firstDate := usageDate(time.Unix(int64(from), 0))
	lastDate := usageDate(time.Unix(int64(to), 0))
	return getUsage(ctx, s.api, "GetUsageCommon", UsageServiceText, firstDate, lastDate)
}

func usageDate(t time.Time) int32 {
	year, month, day := t.In(usageLocation).Date()
	// #nosec G115 -- callers use validated four-digit years, int32 Unix seconds, or recent local dates. YYYYMMDD fits in int32.
	return int32(year*10000 + int(month)*100 + day)
}

func getUsage(ctx context.Context, api translate20250301.TRANSLATE20250301API, op string, service UsageService, from, to int32) (*Usage, error) {
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := api.GetUsageWithContext(ctx, &translate20250301.GetUsageInput{
		From:    new(from),
		To:      new(to),
		Service: new(string(service)),
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
