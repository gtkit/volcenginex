package volcenginex

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const maxTextBatchCharacters = 5000

// TextService wraps the official synchronous text translation APIs.
type TextService struct {
	client         *Client
	api            translate20250301.TRANSLATE20250301API
	batchSize      int
	partialFailure bool
}

// Translate translates text segments while preserving their original order.
// It rejects any nonblank segment exceeding 5000 Unicode characters before sending requests.
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
	batches, err := prepareTextBatches(req.TextList, result.Items, min(batchSize, defaultBatchSize))
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
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
	for _, batch := range batches {
		batchTexts := make([]string, len(batch.indexes))
		for i, index := range batch.indexes {
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
			out, err := s.api.TranslateTextWithContext(workerCtx, textInput(req, batchTexts), retainNumericProviderError)
			if mapped := validateTranslationResponse(workerCtx, out, len(batchTexts), err); mapped != nil {
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
			result.CharacterCount += batch.characters
			if id := requestIDFromMetadata(out.Metadata); id != "" && result.RequestID == "" {
				result.RequestID = id
			}
			for i, item := range out.TranslationList {
				index := batch.indexes[i]
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

type textBatch struct {
	indexes    []int
	characters int
}

func prepareTextBatches(texts []string, items []TextItem, batchSize int) ([]textBatch, error) {
	var batches []textBatch
	var batch textBatch
	for i, segment := range texts {
		items[i] = TextItem{Index: i, Source: segment}
		if strings.TrimSpace(segment) == "" {
			continue
		}
		characters := CountVolcMTChar(segment)
		if characters > maxTextBatchCharacters {
			message := fmt.Sprintf("text segment at index %d exceeds %d characters", i, maxTextBatchCharacters)
			return nil, wrapSentinel("TranslateText", ErrInvalidArgument, message)
		}
		if len(batch.indexes) == batchSize || batch.characters+characters > maxTextBatchCharacters {
			batches = append(batches, batch)
			batch = textBatch{}
		}
		batch.indexes = append(batch.indexes, i)
		batch.characters += characters
	}
	if len(batch.indexes) != 0 {
		batches = append(batches, batch)
	}
	return batches, nil
}

func validateTranslationResponse(ctx context.Context, out *translate20250301.TranslateTextOutput, expected int, err error) error {
	const op = "TranslateText"
	var metadata *response.ResponseMetadata
	if out != nil {
		metadata = out.Metadata
	}
	if err := providerResponseError(ctx, op, metadata, err); err != nil {
		return err
	}
	if out == nil || len(out.TranslationList) != expected {
		return invalidProviderResponse(op, metadata, "translation count does not match input")
	}
	for _, item := range out.TranslationList {
		if item == nil || strings.TrimSpace(valueOrEmpty(item.Translation)) == "" {
			return invalidProviderResponse(op, metadata, "translation is empty")
		}
	}
	return nil
}

// Detect detects nonblank texts in batches of up to 16 items.
// Results preserve input order and the original indexes. A failed batch returns an error and no results.
func (s *TextService) Detect(ctx context.Context, texts []string) ([]DetectItem, error) {
	const op = "LangDetect"
	const maxBatchSize = 16
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
	items := make([]DetectItem, 0, len(values))
	for batch := range slices.Chunk(values, maxBatchSize) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out, err := s.api.LangDetectWithContext(ctx, langDetectInput(batch), retainNumericProviderError)
		var metadata *response.ResponseMetadata
		if out != nil {
			metadata = out.Metadata
		}
		if err := providerResponseError(ctx, op, metadata, err); err != nil {
			return nil, err
		}
		if out == nil || len(out.DetectedLanguageList) != len(batch) {
			return nil, invalidProviderResponse(op, metadata, "detection count does not match input")
		}
		items = append(items, detectOutput(out, indexes[:len(batch)])...)
		indexes = indexes[len(batch):]
	}
	return items, nil
}
