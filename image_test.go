package volcenginex

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
)

type fakeImageAPI struct {
	translate20250301.TRANSLATE20250301API
	input *translate20250301.TranslateImageInput
}

func (f *fakeImageAPI) TranslateImageWithContext(_ context.Context, input *translate20250301.TranslateImageInput, _ ...request.Option) (*translate20250301.TranslateImageOutput, error) {
	f.input = input
	return &translate20250301.TranslateImageOutput{Image: new("translated-image")}, nil
}

func TestImageTranslateEncodesBytes(t *testing.T) {
	fake := &fakeImageAPI{}
	service := &ImageService{api: fake, maxBytes: 5}
	data := []byte("hello")
	result, err := service.Translate(t.Context(), ImageTranslateRequest{Image: data, TargetLanguage: "en"})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if result.ImageURL != "translated-image" {
		t.Fatalf("image = %q", result.ImageURL)
	}
	if result.ImageCount != 1 {
		t.Fatalf("image count = %d, want 1", result.ImageCount)
	}
	if got := *fake.input.Image; got != base64.StdEncoding.EncodeToString(data) {
		t.Fatalf("encoded image = %q", got)
	}
}

func TestImageTranslateRejectsOversize(t *testing.T) {
	service := &ImageService{api: &fakeImageAPI{}, maxBytes: 2}
	_, err := service.Translate(t.Context(), ImageTranslateRequest{Image: []byte("123"), TargetLanguage: "en"})
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("error = %v, want ErrImageTooLarge", err)
	}
}

func TestImageTranslateURLAndFile(t *testing.T) {
	fake := &fakeImageAPI{}
	service := &ImageService{api: fake, maxBytes: 1024}
	result, err := service.TranslateURL(t.Context(), "https://example/image.png", "zh", "en")
	if err != nil || result.ImageURL != "translated-image" || result.ImageCount != 1 || *fake.input.Image != "https://example/image.png" {
		t.Fatalf("URL result=%#v input=%#v err=%v", result, fake.input, err)
	}
	path := t.TempDir() + "/image.bin"
	if err := os.WriteFile(path, []byte("bytes"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	fileResult, err := service.TranslateFile(t.Context(), path, "zh", "en")
	if err != nil {
		t.Fatalf("TranslateFile error: %v", err)
	}
	if fileResult.ImageCount != 1 {
		t.Fatalf("file image count = %d, want 1", fileResult.ImageCount)
	}
}

func TestImageTranslateRejectsInvalidInputs(t *testing.T) {
	service := &ImageService{api: &fakeImageAPI{}, maxBytes: 1024}
	if _, err := service.Translate(t.Context(), ImageTranslateRequest{Image: []byte("x")}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing target error = %v", err)
	}
	if _, err := service.Translate(t.Context(), ImageTranslateRequest{TargetLanguage: "en"}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing image error = %v", err)
	}
	if _, err := service.TranslateFile(t.Context(), "/missing/image", "zh", "en"); err == nil {
		t.Fatal("missing file returned nil error")
	}
	if _, err := service.TranslateURL(t.Context(), "", "zh", "en"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing URL error = %v", err)
	}
}
