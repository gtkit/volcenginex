package volcenginex

import (
	"context"
	"encoding/base64"
	"os"
	"strings"

	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

// ImageService wraps the official image translation API.
type ImageService struct {
	client   *Client
	api      translate20250301.TRANSLATE20250301API
	maxBytes int64
}

// Translate translates image bytes after base64 encoding them.
func (s *ImageService) Translate(ctx context.Context, req ImageTranslateRequest) (*ImageTranslateResult, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if strings.TrimSpace(req.TargetLanguage) == "" {
		return nil, wrapSentinel("TranslateImage", ErrInvalidArgument, "target language is required")
	}
	if len(req.Image) == 0 {
		return nil, wrapSentinel("TranslateImage", ErrInvalidArgument, "image is required")
	}
	if int64(len(req.Image)) > s.maxBytes {
		return nil, formatLimitError(ErrImageTooLarge, "image size %d exceeds %d bytes", len(req.Image), s.maxBytes)
	}
	encoded := base64.StdEncoding.EncodeToString(req.Image)
	return s.translateEncoded(ctx, req, encoded)
}

// TranslateURL translates an image URL through the same official image API.
func (s *ImageService) TranslateURL(ctx context.Context, url, source, target string) (*ImageTranslateResult, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if strings.TrimSpace(url) == "" || strings.TrimSpace(target) == "" {
		return nil, wrapSentinel("TranslateImage", ErrInvalidArgument, "image URL and target language are required")
	}
	return s.translateEncoded(ctx, ImageTranslateRequest{SourceLanguage: source, TargetLanguage: target}, url)
}

// TranslateFile reads and translates a local image file.
func (s *ImageService) TranslateFile(ctx context.Context, path, source, target string) (*ImageTranslateResult, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, clientError("TranslateImage", err)
	}
	if info.Size() > s.maxBytes {
		return nil, formatLimitError(ErrImageTooLarge, "image size %d exceeds %d bytes", info.Size(), s.maxBytes)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// #nosec G304 -- the caller explicitly supplies the local image path.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, clientError("TranslateImage", err)
	}
	return s.Translate(ctx, ImageTranslateRequest{Image: data, SourceLanguage: source, TargetLanguage: target})
}

func (s *ImageService) translateEncoded(ctx context.Context, req ImageTranslateRequest, encoded string) (*ImageTranslateResult, error) {
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.TranslateImageWithContext(ctx, imageInput(req, encoded))
	if err != nil {
		return nil, providerError("TranslateImage", err)
	}
	return imageOutput(out), nil
}
