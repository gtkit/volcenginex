package volcenginex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"
)

type providerTestTransport func(*http.Request) (*http.Response, error)

func (f providerTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func providerTestClient(t *testing.T, transport providerTestTransport, opts ...Option) *Client {
	t.Helper()
	options := append([]Option{WithHTTPClient(&http.Client{Transport: transport})}, opts...)
	client, err := New(Config{AK: "example-ak", SK: "example-sk", SessionToken: "example-token"}, options...)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func providerTestResponse(req *http.Request, status int, body string) (*http.Response, error) {
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestTranslateSDKProtocol(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		source     string
		sourceJSON string
		texts      []string
		response   string
		items      []TextItem
		characters int
	}{
		{
			name:       "single text with auto detection",
			texts:      []string{"Hello world."},
			response:   `{"ResponseMetadata":{"RequestId":"sdk-text"},"Result":{"TranslationList":[{"Translation":"你好，世界。","DetectedSourceLanguage":"en"}]}}`,
			items:      []TextItem{{Index: 0, Source: "Hello world.", Translation: "你好，世界。", DetectedSourceLanguage: "en"}},
			characters: 12,
		},
		{
			name:       "multiple texts with explicit source",
			source:     "en",
			sourceJSON: `"en"`,
			texts:      []string{"Hello", "Thanks"},
			response:   `{"ResponseMetadata":{"RequestId":"sdk-text"},"Result":{"TranslationList":[{"Translation":"你好"},{"Translation":"谢谢"}]}}`,
			items: []TextItem{
				{Index: 0, Source: "Hello", Translation: "你好"},
				{Index: 1, Source: "Thanks", Translation: "谢谢"},
			},
			characters: 11,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				query := req.URL.Query()
				if req.Method != http.MethodPost || req.URL.Host != "open.volcengineapi.com" ||
					query.Get("Action") != "TranslateText" || query.Get("Version") != "2025-03-01" {
					t.Errorf("unexpected SDK request: %s %s", req.Method, req.URL)
				}
				var body struct {
					TargetLanguage string
					SourceLanguage json.RawMessage
					TextList       []string
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					return nil, errors.Join(err, req.Body.Close())
				}
				if body.TargetLanguage != "zh" || string(body.SourceLanguage) != tc.sourceJSON || !slices.Equal(body.TextList, tc.texts) {
					t.Errorf("unexpected SDK request body: target=%q source=%s texts=%q", body.TargetLanguage, body.SourceLanguage, body.TextList)
				}
				return providerTestResponse(req, http.StatusOK, tc.response)
			}, WithMaxRetries(0))
			got, err := client.Text().Translate(t.Context(), TextTranslateRequest{
				SourceLanguage: tc.source,
				TargetLanguage: "zh",
				TextList:       tc.texts,
			})
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 1 || got.RequestID != "sdk-text" || got.CharacterCount != tc.characters || !slices.Equal(got.Items, tc.items) {
				t.Fatalf("attempts = %d, result = %#v, want one attempt and translations %v", calls.Load(), got, tc.items)
			}
		})
	}
}

func TestTranslateDocumentSegmentsWithinRequestLimits(t *testing.T) {
	t.Parallel()
	const protected = "{{name}}"
	segments := []string{
		strings.Repeat("段", 3000),
		strings.Repeat("🙂", 2000-len(protected)) + protected,
		" \n ",
	}
	for i := range 17 {
		segments = append(segments, string(rune('a'+i)))
	}

	var calls atomic.Int32
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		var input struct{ TextList []string }
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		characters := 0
		translations := make([]struct{ Translation string }, len(input.TextList))
		for i, segment := range input.TextList {
			if strings.TrimSpace(segment) == "" {
				t.Error("blank segment sent to provider")
			}
			characters += CountVolcMTChar(segment)
			translations[i].Translation = "译:" + segment
		}
		if len(input.TextList) > 16 || characters > 5000 {
			t.Errorf("request contains %d segments and %d characters", len(input.TextList), characters)
		}
		body, err := json.Marshal(map[string]any{"Result": map[string]any{"TranslationList": translations}})
		if err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		return providerTestResponse(req, http.StatusOK, string(body))
	}, WithMaxRetries(0))

	result, err := client.Text().Translate(t.Context(), TextTranslateRequest{
		TargetLanguage: "zh",
		TextList:       segments,
		BatchSize:      128,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || len(result.Items) != len(segments) || result.CharacterCount != 5017 {
		t.Fatalf("requests = %d, items = %d, characters = %d", calls.Load(), len(result.Items), result.CharacterCount)
	}
	for i, item := range result.Items {
		want := ""
		if strings.TrimSpace(segments[i]) != "" {
			want = "译:" + segments[i]
		}
		if item.Index != i || item.Source != segments[i] || item.Translation != want {
			t.Errorf("translation did not match input segment %d", i)
		}
	}
}

func TestTranslateRejectsOversizeSegmentBeforeHTTP(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return providerTestResponse(req, http.StatusOK, "{}")
	}, WithMaxRetries(0))

	result, err := client.Text().Translate(t.Context(), TextTranslateRequest{
		TargetLanguage: "zh",
		TextList:       []string{"valid", strings.Repeat("🙂", 5001)},
		BatchSize:      1,
	})
	if !errors.Is(err, ErrInvalidArgument) || result != nil || calls.Load() != 0 {
		t.Fatalf("requests = %d, result present = %t, error = %v", calls.Load(), result != nil, err)
	}
}

func TestTranslateExplicitRetryCount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		opts  []Option
		calls int32
	}{
		{name: "default", calls: 4},
		{name: "explicit zero", opts: []Option{WithMaxRetries(0)}, calls: 1},
		{name: "later zero", opts: []Option{WithMaxRetries(2), WithMaxRetries(0)}, calls: 1},
		{name: "later positive", opts: []Option{WithMaxRetries(0), WithMaxRetries(1)}, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return providerTestResponse(req, http.StatusServiceUnavailable, `{"ResponseMetadata":{"Error":{"Code":"ServiceUnavailable","Message":"unavailable"}}}`)
			}, tc.opts...)
			_, err := client.Text().Translate(t.Context(), TextTranslateRequest{TargetLanguage: "zh", TextList: []string{"hello"}})
			got, ok := errors.AsType[*Error](err)
			if !ok || got.HTTPStatus != http.StatusServiceUnavailable || !got.Retryable || calls.Load() != tc.calls {
				t.Fatalf("attempts = %d, want %d; error = %v", calls.Load(), tc.calls, err)
			}
		})
	}
}

func TestTranslatePreservesNumericProviderError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload, code string
		status, codeN       int
	}{
		{"named error", `{"Code":"Rejected","CodeN":-415,"Message":"invalid direction"}`, "Rejected", http.StatusOK, -415},
		{"HTTP 200 numeric only", `{"CodeN":100018,"Message":"limited"}`, "100018", http.StatusOK, 100018},
		{"HTTP 403 numeric only", `{"CodeN":100013,"Message":"denied"}`, "100013", http.StatusForbidden, 100013},
		{"text code only", `{"Code":"-403","Message":"unsynchronized"}`, "-403", http.StatusOK, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				body := `{"ResponseMetadata":{"RequestId":"translate-error","Error":` + tc.payload + `},"Result":{"TranslationList":[{"Translation":"discard"}]}}`
				return providerTestResponse(req, tc.status, body)
			}, WithMaxRetries(0))
			result, err := client.Text().Translate(t.Context(), TextTranslateRequest{TargetLanguage: "zh", TextList: []string{"hello"}})
			got, ok := errors.AsType[*Error](err)
			if !ok || result != nil || got.Code != tc.code || got.HTTPStatus != tc.status || got.RequestID != "translate-error" {
				t.Fatalf("result = %#v, error = %#v", result, err)
			}
			if codeN, present := got.CodeN(); codeN != tc.codeN || present != (tc.codeN != 0) {
				t.Fatalf("CodeN() = (%d, %t), want (%d, %t)", codeN, present, tc.codeN, tc.codeN != 0)
			}
			if failure, ok := errors.AsType[volcengineerr.RequestFailure](err); !ok || failure.RequestID() != "translate-error" {
				t.Fatal("provider cause was not preserved")
			}
		})
	}
}

func TestTranslateRejectsInvalidResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body string }{
		{"missing", `{"Result":{}}`},
		{"legacy top level", `{"TranslationList":[{"Translation":"你好"}]}`},
		{"extra item", `{"Result":{"TranslationList":[{"Translation":"你好"},{"Translation":"额外"}]}}`},
		{"null item", `{"Result":{"TranslationList":[null]}}`},
		{"blank", `{"Result":{"TranslationList":[{"Translation":" \t "}]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				return providerTestResponse(req, http.StatusOK, tc.body)
			}, WithMaxRetries(0))
			result, err := client.Text().Translate(t.Context(), TextTranslateRequest{TargetLanguage: "zh", TextList: []string{"hello"}})
			if got, ok := errors.AsType[*Error](err); !ok || got.Code != "InvalidResponse" || result != nil {
				t.Fatalf("result = %#v, error = %#v", result, err)
			}
		})
	}
}

func TestTranslatePartialFailureExcludesInvalidBatch(t *testing.T) {
	t.Parallel()
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		response := `{"Result":{"TranslationList":[]}}`
		if strings.Contains(string(body), `"hello"`) {
			response = `{"ResponseMetadata":{"RequestId":"valid-batch"},"Result":{"TranslationList":[{"Translation":"你好","DetectedSourceLanguage":"en"}]}}`
		}
		return providerTestResponse(req, http.StatusOK, response)
	}, WithMaxRetries(0), WithPartialFailure())
	result, err := client.Text().Translate(t.Context(), TextTranslateRequest{TargetLanguage: "zh", TextList: []string{"hello", "invalid", " "}, BatchSize: 1})
	if err != nil || result == nil || len(result.Errors) != 1 || result.CharacterCount != 5 || result.RequestID != "valid-batch" {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	if len(result.Items) != 3 || result.Items[0].Translation != "你好" || result.Items[1].Translation != "" || result.Items[2].Translation != "" {
		t.Fatalf("items = %#v", result.Items)
	}
}

func TestTranslateRetainsCallerCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		cancel()
		return nil, errors.Join(req.Context().Err(), req.Body.Close())
	}, WithMaxRetries(0))
	_, err := client.Text().Translate(ctx, TextTranslateRequest{TargetLanguage: "zh", TextList: []string{"hello"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
