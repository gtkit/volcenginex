package volcenginex

import (
	"context"
	"strings"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	sdkclient "github.com/volcengine/volcengine-go-sdk/volcengine/client"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

// AudioService submits and queries audio and video file translations.
// The service returned by Client.Audio is safe for concurrent use.
type AudioService struct {
	api translate20250301.TRANSLATE20250301API
}

// SubmitAudio submits a media URI for translation and returns a task ID.
// It returns ErrInvalidArgument for a nil context or a blank required field.
// The SDK makes no automatic retries because submission creates a task.
func (s *AudioService) SubmitAudio(ctx context.Context, req SubmitAudioRequest) (*SubmitAudioResult, error) {
	const op = "SubmitAudio"
	if ctx == nil || strings.TrimSpace(req.URI) == "" ||
		strings.TrimSpace(req.SourceLanguage) == "" || strings.TrimSpace(req.TargetLanguage) == "" {
		return nil, wrapSentinel(op, ErrInvalidArgument, "context, media URI, source language, and target language are required")
	}

	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.SubmitAudioWithContext(ctx, &translate20250301.SubmitAudioInput{
		Uri:            new(req.URI),
		SourceLanguage: new(req.SourceLanguage),
		TargetLanguage: new(req.TargetLanguage),
	}, retainNumericProviderError, func(r *request.Request) {
		r.Retryer = sdkclient.NoOpRetryer{}
	})
	var metadata *response.ResponseMetadata
	if out != nil {
		metadata = out.Metadata
	}
	if err := providerResponseError(ctx, op, metadata, err); err != nil {
		return nil, err
	}
	if out == nil || strings.TrimSpace(valueOrEmpty(out.TaskId)) == "" {
		return nil, invalidProviderResponse(op, metadata, "audio task ID is missing")
	}
	return &SubmitAudioResult{
		TaskID:    *out.TaskId,
		RequestID: requestIDFromMetadata(metadata),
	}, nil
}

// QueryAudio returns the current task status and available subtitles.
// It returns ErrInvalidArgument for a nil context or a blank task ID.
// The caller controls when to query again.
func (s *AudioService) QueryAudio(ctx context.Context, taskID string) (*QueryAudioResult, error) {
	const op = "QueryAudio"
	if ctx == nil || strings.TrimSpace(taskID) == "" {
		return nil, wrapSentinel(op, ErrInvalidArgument, "context and audio task ID are required")
	}

	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.QueryAudioWithContext(ctx, &translate20250301.QueryAudioInput{
		TaskId: new(taskID),
	}, retainNumericProviderError)
	var metadata *response.ResponseMetadata
	if out != nil {
		metadata = out.Metadata
	}
	if err := providerResponseError(ctx, op, metadata, err); err != nil {
		return nil, err
	}
	if out == nil || strings.TrimSpace(valueOrEmpty(out.Status)) == "" {
		return nil, invalidProviderResponse(op, metadata, "audio task status is missing")
	}

	result := &QueryAudioResult{
		Status:     *out.Status,
		Confidence: out.Confidence,
		Subtitles:  make([]AudioSubtitle, 0, len(out.Subtitles)),
		RequestID:  requestIDFromMetadata(metadata),
	}
	for _, subtitle := range out.Subtitles {
		if subtitle == nil {
			continue
		}
		result.Subtitles = append(result.Subtitles, AudioSubtitle{
			StartTime:   valueOrZero(subtitle.StartTime),
			EndTime:     valueOrZero(subtitle.EndTime),
			Text:        valueOrEmpty(subtitle.Text),
			Translation: valueOrEmpty(subtitle.Translation),
		})
	}
	return result, nil
}
