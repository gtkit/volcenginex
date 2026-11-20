package volcenginex

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// TextService wraps the official synchronous text translation APIs.
type TextService struct {
	client         *Client
	api            translate20250301.TRANSLATE20250301API
	batchSize      int
	partialFailure bool
}

// Translate translates text items while preserving their original order.
func (s *TextService) Translate(ctx context.Context, req TextTranslateRequest) (*TextTranslateResult, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if strings.TrimSpace(req.TargetLanguage) == "" {
		return nil, wrapSentinel("TranslateText", ErrInvalidArgument, "target language is required")
	}
	batchSize := req.BatchSize
	if batchSize == 0 {
		batchSize = s.batchSize
	}
	if batchSize <= 0 {
		return nil, wrapSentinel("TranslateText", ErrInvalidArgument, "batch size must be positive")
	}

	result := &TextTranslateResult{Items: make([]TextItem, len(req.TextList))}
	indexes := make([]int, 0, len(req.TextList))
	for i, text := range req.TextList {
		result.Items[i] = TextItem{Index: i, Source: text}
		if strings.TrimSpace(text) != "" {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) == 0 {
		return result, nil
	}

	requestID := trace.NewRequestID()
	ctx = trace.WithRequestID(ctx, requestID)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, workerCtx := errgroup.WithContext(workerCtx)
	limit := semaphore.NewWeighted(4)
	var mu sync.Mutex
	var firstErr error
	for start := 0; start < len(indexes); start += batchSize {
		end := min(start+batchSize, len(indexes))
		batchIndexes := append([]int(nil), indexes[start:end]...)
		batchTexts := make([]string, len(batchIndexes))
		for i, index := range batchIndexes {
			batchTexts[i] = req.TextList[index]
		}
		g.Go(func() error {
			if err := limit.Acquire(workerCtx, 1); err != nil {
				if s.partialFailure {
					mu.Lock()
					result.Errors = append(result.Errors, err)
					mu.Unlock()
					return nil
				}
				return err
			}
			defer limit.Release(1)
			out, err := s.api.TranslateTextWithContext(workerCtx, textInput(req, batchTexts))
			if err != nil {
				mapped := providerError("TranslateText", err)
				if s.partialFailure {
					mu.Lock()
					result.Errors = append(result.Errors, mapped)
					mu.Unlock()
					return nil
				}
				mu.Lock()
				if firstErr == nil {
					firstErr = mapped
				}
				mu.Unlock()
				return mapped
			}
			mu.Lock()
			if out == nil {
				mu.Unlock()
				return nil
			}
			if id := requestIDFromMetadata(out.Metadata); id != "" && result.RequestID == "" {
				result.RequestID = id
			}
			for i, item := range out.TranslationList {
				if i >= len(batchIndexes) || item == nil {
					continue
				}
				index := batchIndexes[i]
				result.Items[index].Translation = valueOrEmpty(item.Translation)
				result.Items[index].DetectedSourceLanguage = valueOrEmpty(item.DetectedSourceLanguage)
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil && !s.partialFailure {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, err
	}
	if result.RequestID == "" {
		result.RequestID = requestID
	}
	return result, nil
}

// Detect detects the language of each non-blank text item.
func (s *TextService) Detect(ctx context.Context, texts []string) ([]DetectItem, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	indexes := make([]int, 0, len(texts))
	values := make([]string, 0, len(texts))
	for i, text := range texts {
		if strings.TrimSpace(text) != "" {
			indexes = append(indexes, i)
			values = append(values, text)
		}
	}
	if len(values) == 0 {
		return []DetectItem{}, nil
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.LangDetectWithContext(ctx, langDetectInput(values))
	if err != nil {
		return nil, providerError("LangDetect", err)
	}
	return detectOutput(out, indexes), nil
}

// Usage returns the last thirty days of machine translation usage points.
func (s *TextService) Usage(ctx context.Context) (*Usage, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	now := time.Now().UTC()
	from := int32(now.Add(-30 * 24 * time.Hour).Unix())
	to := int32(now.Unix())
	service := "translate"
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.GetUsageWithContext(ctx, &translate20250301.GetUsageInput{
		From:    new(from),
		To:      new(to),
		Service: new(service),
	})
	if err != nil {
		return nil, providerError("GetUsage", err)
	}
	return usageOutput(out), nil
}
