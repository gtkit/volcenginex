package volcenginex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
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

func TestDetectBatchesPreserveOrder(t *testing.T) {
	t.Parallel()
	var texts, wantTexts []string
	var want []DetectItem
	for i := range 33 {
		if i%16 == 0 {
			texts = append(texts, "", " \n\u3000")
		}
		text := "text-" + strconv.Itoa(i)
		wantTexts = append(wantTexts, text)
		want = append(want, DetectItem{Index: len(texts), Language: "en", Confidence: float64(i+1) / 64})
		texts = append(texts, text)
	}

	var calls atomic.Int32
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		query := req.URL.Query()
		if req.Method != http.MethodPost || query.Get("Action") != "LangDetect" || query.Get("Version") != "2025-03-01" {
			t.Errorf("unexpected SDK request: %s %s", req.Method, req.URL)
		}
		var input struct{ TextList []string }
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		start := (int(calls.Add(1)) - 1) * 16
		end := min(start+16, len(wantTexts))
		if start >= end || !slices.Equal(input.TextList, wantTexts[start:end]) {
			t.Errorf("unexpected batch starting at %d: %v", start, input.TextList)
			return providerTestResponse(req, http.StatusBadRequest, `{"ResponseMetadata":{"Error":{"Code":"BadBatch"}}}`)
		}
		detected := make([]struct {
			Language   string
			Confidence float64
		}, end-start)
		for i, item := range want[start:end] {
			detected[i].Language = item.Language
			detected[i].Confidence = item.Confidence
		}
		body, err := json.Marshal(map[string]any{"Result": map[string]any{"DetectedLanguageList": detected}})
		if err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		return providerTestResponse(req, http.StatusOK, string(body))
	}, WithMaxRetries(0))

	got, err := client.Text().Detect(t.Context(), texts)
	if err != nil || !slices.Equal(got, want) || calls.Load() != 3 {
		t.Fatalf("Detect got %v, %v in %d requests, want %v in 3 requests", got, err, calls.Load(), want)
	}
	got, err = client.Text().Detect(t.Context(), []string{"", "\t\u3000\n"})
	if err != nil || got == nil || len(got) != 0 || calls.Load() != 3 {
		t.Fatalf("Detect for blank input got %#v, %v with %d total requests", got, err, calls.Load())
	}
}

func TestDetectStopsAfterFailedBatch(t *testing.T) {
	t.Parallel()
	const item = `{"Language":"en","Confidence":1}`
	success := `{"Result":{"DetectedLanguageList":[` + strings.Repeat(item+",", 15) + item + `]}}`
	for _, tc := range []struct {
		name, body, code string
	}{
		{
			name: "gateway error",
			body: `{"ResponseMetadata":{"RequestId":"batch-2","Error":{"CodeN":100018,"Message":"limit reached"}}}`,
			code: "100018",
		},
		{
			name: "mismatched count",
			body: `{"ResponseMetadata":{"RequestId":"batch-2"},"Result":{"DetectedLanguageList":[` + item + `]}}`,
			code: "InvalidResponse",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return providerTestResponse(req, http.StatusOK, success)
				}
				return providerTestResponse(req, http.StatusOK, tc.body)
			}, WithMaxRetries(0))
			got, err := client.Text().Detect(t.Context(), slices.Repeat([]string{"hello"}, 33))
			provider, ok := errors.AsType[*Error](err)
			if !ok || provider.Code != tc.code || provider.RequestID != "batch-2" || got != nil || calls.Load() != 2 {
				t.Fatalf("Detect got %#v, %v in %d requests, want error %s from batch 2", got, err, calls.Load(), tc.code)
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

func TestMachineTranslationGatewayErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload, code, message string
		status, codeN                int
		call                         func(context.Context, *Client) error
	}{
		{
			name:    "language detection preserves numeric-only gateway error",
			payload: `{"CodeN":100018,"Message":"limit for this request"}`,
			code:    "100018",
			message: "limit for this request",
			status:  http.StatusOK,
			codeN:   100018,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.Text().Detect(ctx, []string{"hello"})
				return err
			},
		},
		{
			name:    "image translation preserves unlisted error",
			payload: `{"Code":"FutureError","CodeN":100099,"Message":"new provider error"}`,
			code:    "FutureError",
			message: "new provider error",
			status:  http.StatusForbidden,
			codeN:   100099,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.Image().Translate(ctx, ImageTranslateRequest{Image: []byte("image"), TargetLanguage: "zh"})
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				body := `{"ResponseMetadata":{"RequestId":"gateway-request","Error":` + tc.payload + `}}`
				return providerTestResponse(req, tc.status, body)
			}, WithMaxRetries(0))
			err := tc.call(t.Context(), client)
			got, ok := errors.AsType[*Error](err)
			if !ok || got.Code != tc.code || got.Message != tc.message ||
				got.RequestID != "gateway-request" || got.HTTPStatus != tc.status {
				t.Fatalf("error = %#v, want code %q, message %q, request ID gateway-request and status %d", err, tc.code, tc.message, tc.status)
			}
			if code, present := got.CodeN(); !present || code != tc.codeN {
				t.Fatalf("CodeN() = (%d, %t), want (%d, true)", code, present, tc.codeN)
			}
		})
	}
}

func TestLookupGatewayError(t *testing.T) {
	t.Parallel()
	want := GatewayErrorInfo{
		Code:            CodeMissingParameter,
		CodeN:           100002,
		MessageTemplate: "The request is missing %s parameter.",
	}
	for _, code := range []string{CodeMissingParameter, "100002"} {
		if got, ok := LookupGatewayError(code); !ok || got != want {
			t.Errorf("LookupGatewayError(%q) = (%+v, %t), want (%+v, true)", code, got, ok, want)
		}
	}
	for _, code := range []string{"", "FutureError", "100099", "-415"} {
		if got, ok := LookupGatewayError(code); ok || got != (GatewayErrorInfo{}) {
			t.Errorf("LookupGatewayError(%q) = (%+v, %t), want zero value and false", code, got, ok)
		}
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
