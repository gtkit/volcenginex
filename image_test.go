package volcenginex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"os"
	"strings"
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

func TestImageTranslateFile(t *testing.T) {
	fake := &fakeImageAPI{}
	service := &ImageService{api: fake, maxBytes: 1024}
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

func TestImageTranslateURLFetchesBytes(t *testing.T) {
	t.Parallel()
	const sourceURL = "https://images.example.test/source.png?signature=fixture"
	const imageBytes = "synthetic image bytes"
	fetches, translations := 0, 0
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		translations++
		query := req.URL.Query()
		if query.Get("Action") != "TranslateImage" || query.Get("Version") != "2025-03-01" {
			t.Errorf("unexpected provider request: %s", req.URL)
		}
		var body struct {
			Image, SourceLanguage, TargetLanguage string
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		if body.Image != base64.StdEncoding.EncodeToString([]byte(imageBytes)) || body.SourceLanguage != "en" || body.TargetLanguage != "zh" {
			t.Error("provider did not receive the encoded image and requested languages")
		}
		return providerTestResponse(req, http.StatusOK, `{"Result":{"Image":"translated","TextBlocks":[{"Translation":"你好"}]}}`)
	}, WithMaxRetries(0))
	client.image.urlClient.Transport = providerTestTransport(func(req *http.Request) (*http.Response, error) {
		fetches++
		if req.Method != http.MethodGet || req.URL.String() != sourceURL || req.Header.Get("Authorization") != "" || req.Header.Get("X-Security-Token") != "" {
			t.Error("unexpected source image request")
		}
		return &http.Response{
			StatusCode: http.StatusOK, ContentLength: int64(len(imageBytes)),
			Body: io.NopCloser(strings.NewReader(imageBytes)),
		}, nil
	})
	got, err := client.Image().TranslateURL(t.Context(), sourceURL, "en", "zh")
	if err != nil || got == nil || got.ImageCount != 1 || got.ImageURL != "translated" ||
		len(got.TextBlocks) != 1 || got.TextBlocks[0].Translation != "你好" || fetches != 1 || translations != 1 {
		t.Fatalf("result=%#v err=%v fetches=%d translations=%d", got, err, fetches, translations)
	}
}

func TestImageTranslateURLDownloadFailures(t *testing.T) {
	t.Parallel()
	const secret = "synthetic-signature"
	for _, tc := range []struct {
		name           string
		status         int
		contentLength  int64
		transportError error
		wantError      error
	}{
		{name: "advertised size limit", status: 200, contentLength: 4, wantError: ErrImageTooLarge},
		{name: "stream size limit", status: 200, contentLength: -1, wantError: ErrImageTooLarge},
		{name: "HTTP failure", status: 403, contentLength: 1},
		{name: "canceled", transportError: context.Canceled, wantError: context.Canceled},
		{name: "redacted transport failure", transportError: errors.New(secret)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeImageAPI{}
			service := &ImageService{api: fake, maxBytes: 3, urlClient: &http.Client{Transport: providerTestTransport(func(*http.Request) (*http.Response, error) {
				if tc.transportError != nil {
					return nil, tc.transportError
				}
				return &http.Response{
					StatusCode: tc.status, ContentLength: tc.contentLength,
					Body: io.NopCloser(strings.NewReader("abcd")),
				}, nil
			})}}
			result, err := service.TranslateURL(t.Context(), "https://images.example.test/source.png?signature="+secret, "en", "zh")
			if err == nil || result != nil || fake.input != nil || strings.Contains(err.Error(), secret) {
				t.Fatalf("unexpected result or download error: %v", err)
			}
			if tc.wantError != nil {
				if !errors.Is(err, tc.wantError) {
					t.Fatalf("error = %v, want %v", err, tc.wantError)
				}
				return
			}
			got, ok := errors.AsType[*Error](err)
			if !ok || got.Code != "ImageDownloadFailed" || got.HTTPStatus != tc.status {
				t.Fatalf("download error = %#v", err)
			}
		})
	}
}

func TestImageTranslateURLRejectsNonPublicDestinations(t *testing.T) {
	t.Parallel()
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		t.Error("private image destination reached the translation provider")
		return providerTestResponse(req, http.StatusOK, "{}")
	}, WithMaxRetries(0))
	for _, source := range []string{
		"file:///tmp/image.png",
		"https://account:password@images.example.test/image.png",
		"http://127.0.0.1/image.png",
		"http://[::ffff:169.254.169.254]/image.png",
		"http://100.100.100.200/image.png",
		"http://[fd00::1]/image.png",
	} {
		if _, err := client.Image().TranslateURL(t.Context(), source, "", "zh"); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("destination was not rejected as invalid: %v", err)
		}
	}
	for _, source := range []string{"1.1.1.1", "2606:4700::1111"} {
		if !publicImageIP(netip.MustParseAddr(source)) {
			t.Errorf("public IP rejected: %s", source)
		}
	}
}

func TestImageURLRedirectPolicy(t *testing.T) {
	t.Parallel()
	start, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://images.example.test/img?signature=secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	next := start.Clone(t.Context())
	next.Header.Set("Referer", start.URL.String())
	if err := checkImageRedirect(next, []*http.Request{start}); err != nil || next.Header.Get("Referer") != "" {
		t.Fatal("HTTPS redirect should succeed with no referer")
	}
	next.URL.Scheme = "http"
	if err := checkImageRedirect(next, []*http.Request{start}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("downgrade redirect error = %v", err)
	}
	next.URL.Scheme = "https"
	if err := checkImageRedirect(next, make([]*http.Request, 6)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("redirect limit error = %v", err)
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
