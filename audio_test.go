package volcenginex

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	gtjson "github.com/gtkit/json/v2"
)

func audioTestInput() SubmitAudioRequest {
	return SubmitAudioRequest{
		URI:            "https://media.example.com/a%2Fb.mp4?signature=abc%2B123%3D&expires=60",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	}
}

func TestAudioSDKProtocol(t *testing.T) {
	t.Parallel()
	input := audioTestInput()
	const taskID = "task-01"
	var submitCalls, queryCalls atomic.Int32
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		query := req.URL.Query()
		if req.Method != http.MethodPost || req.URL.Host != "open.volcengineapi.com" ||
			req.URL.Path != "/" || query.Get("Version") != "2025-03-01" {
			t.Errorf("unexpected SDK request: %s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") == "" || req.Header.Get("X-Client-Request-Id") == "" {
			t.Error("SDK signing or client request ID is missing")
		}
		var body map[string]string
		if err := gtjson.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, errors.Join(err, req.Body.Close())
		}
		switch query.Get("Action") {
		case "SubmitAudio":
			submitCalls.Add(1)
			want := map[string]string{"Uri": input.URI, "SourceLanguage": "en", "TargetLanguage": "zh"}
			if !maps.Equal(body, want) {
				t.Errorf("submit payload = %v, want %v", body, want)
			}
			return providerTestResponse(req, http.StatusOK, `{"ResponseMetadata":{"RequestId":"submit-01"},"Result":{"TaskId":"task-01"}}`)
		case "QueryAudio":
			queryCalls.Add(1)
			if !maps.Equal(body, map[string]string{"TaskId": taskID}) {
				t.Errorf("query payload = %v, want task ID %q", body, taskID)
			}
			return providerTestResponse(req, http.StatusOK, `{
				"ResponseMetadata":{"RequestId":"query-01"},
				"Result":{"Status":"success","Confidence":0.875,"Subtitles":[
					{"StartTime":1200,"EndTime":2400,"Text":"Thanks","Translation":"谢谢"},
					null,
					{"StartTime":0,"EndTime":1199,"Text":"Hello","Translation":"你好"}
				]}
			}`)
		default:
			t.Errorf("unexpected action in %s", req.URL)
			return providerTestResponse(req, http.StatusBadRequest, `{}`)
		}
	}, WithMaxRetries(0))

	task, err := client.Audio().SubmitAudio(t.Context(), input)
	if err != nil || task == nil || *task != (SubmitAudioResult{TaskID: taskID, RequestID: "submit-01"}) {
		t.Fatalf("SubmitAudio = %+v, %v", task, err)
	}
	got, err := client.Audio().QueryAudio(t.Context(), task.TaskID)
	if err != nil || got == nil || got.Status != "success" || got.RequestID != "query-01" ||
		got.Confidence == nil || *got.Confidence != 0.875 {
		t.Fatalf("QueryAudio = %+v, %v", got, err)
	}
	want := []AudioSubtitle{
		{StartTime: 1200, EndTime: 2400, Text: "Thanks", Translation: "谢谢"},
		{StartTime: 0, EndTime: 1199, Text: "Hello", Translation: "你好"},
	}
	if !slices.Equal(got.Subtitles, want) || submitCalls.Load() != 1 || queryCalls.Load() != 1 {
		t.Fatalf("subtitles = %+v, submit calls = %d, query calls = %d", got.Subtitles, submitCalls.Load(), queryCalls.Load())
	}
}

func TestAudioQueryPreservesStatus(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"created", "download_failed", "future_status"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				body := `{"Result":{"Status":` + strconv.Quote(status)
				if status == "future_status" {
					body += `,"Confidence":0`
				}
				return providerTestResponse(req, http.StatusOK, body+`}}`)
			}, WithMaxRetries(0))
			got, err := client.Audio().QueryAudio(t.Context(), "task-01")
			if err != nil || got == nil || got.Status != status || len(got.Subtitles) != 0 {
				t.Fatalf("QueryAudio = %+v, %v, want status %q without subtitles", got, err, status)
			}
			if status == "future_status" {
				if got.Confidence == nil || *got.Confidence != 0 {
					t.Fatalf("confidence = %v, want explicit zero", got.Confidence)
				}
			} else if got.Confidence != nil {
				t.Fatalf("confidence = %v, want nil", got.Confidence)
			}
		})
	}
}

func TestAudioRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		t.Error("invalid input reached the transport")
		return providerTestResponse(req, http.StatusBadRequest, `{}`)
	}, WithMaxRetries(0))

	input := audioTestInput()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		req  SubmitAudioRequest
	}{
		{name: "context", req: input},
		{name: "URI", ctx: t.Context(), req: SubmitAudioRequest{URI: "\t", SourceLanguage: "en", TargetLanguage: "zh"}},
		{name: "source", ctx: t.Context(), req: SubmitAudioRequest{URI: input.URI, SourceLanguage: " ", TargetLanguage: "zh"}},
		{name: "target", ctx: t.Context(), req: SubmitAudioRequest{URI: input.URI, SourceLanguage: "en", TargetLanguage: "\n"}},
	} {
		t.Run("submit "+tc.name, func(t *testing.T) {
			t.Parallel()
			if got, err := client.Audio().SubmitAudio(tc.ctx, tc.req); got != nil || !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("SubmitAudio = %+v, %v, want ErrInvalidArgument", got, err)
			}
		})
	}
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		taskID string
	}{
		{name: "context", taskID: "task-01"},
		{name: "task ID", ctx: t.Context(), taskID: " \t"},
	} {
		t.Run("query "+tc.name, func(t *testing.T) {
			t.Parallel()
			if got, err := client.Audio().QueryAudio(tc.ctx, tc.taskID); got != nil || !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("QueryAudio = %+v, %v, want ErrInvalidArgument", got, err)
			}
		})
	}
}

func TestAudioResponseErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, action, body, code string
		codeN                    int
	}{
		{"submit provider error", "SubmitAudio", `{"ResponseMetadata":{"RequestId":"audio-error","Error":{"CodeN":-400,"Message":"provider rejected request"}}}`, "-400", -400},
		{"query provider error", "QueryAudio", `{"ResponseMetadata":{"RequestId":"audio-error","Error":{"CodeN":-400,"Message":"provider rejected request"}}}`, "-400", -400},
		{"missing task ID", "SubmitAudio", `{"ResponseMetadata":{"RequestId":"audio-error"},"Result":{}}`, "InvalidResponse", 0},
		{"blank task status", "QueryAudio", `{"ResponseMetadata":{"RequestId":"audio-error"},"Result":{"Status":" "}}`, "InvalidResponse", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				return providerTestResponse(req, http.StatusOK, tc.body)
			}, WithMaxRetries(0))

			var err error
			if tc.action == "SubmitAudio" {
				var got *SubmitAudioResult
				got, err = client.Audio().SubmitAudio(t.Context(), audioTestInput())
				if got != nil {
					t.Fatalf("submission with error returned %+v", got)
				}
			} else {
				var got *QueryAudioResult
				got, err = client.Audio().QueryAudio(t.Context(), "task-01")
				if got != nil {
					t.Fatalf("query with error returned %+v", got)
				}
			}
			apiErr, ok := errors.AsType[*Error](err)
			if !ok || apiErr.Op != tc.action || apiErr.Code != tc.code || apiErr.RequestID != "audio-error" {
				t.Fatalf("provider error = %+v, want action %q and code %q", apiErr, tc.action, tc.code)
			}
			if codeN, ok := apiErr.CodeN(); codeN != tc.codeN || ok != (tc.codeN != 0) {
				t.Fatalf("CodeN() = %d, %t, want %d", codeN, ok, tc.codeN)
			}
		})
	}
}

func TestAudioCancellation(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"SubmitAudio", "QueryAudio"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				cancel()
				return nil, errors.Join(req.Context().Err(), req.Body.Close())
			}, WithMaxRetries(0))

			var err error
			if action == "SubmitAudio" {
				_, err = client.Audio().SubmitAudio(ctx, audioTestInput())
			} else {
				_, err = client.Audio().QueryAudio(ctx, "task-01")
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("%s returned %v, want context.Canceled", action, err)
			}
		})
	}
}

func TestAudioRetryPolicy(t *testing.T) {
	t.Parallel()
	var submitCalls, queryCalls atomic.Int32
	const failure = `{"ResponseMetadata":{"RequestId":"retry-error","Error":{"Code":"InternalError","Message":"try again"}}}`
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		switch action := req.URL.Query().Get("Action"); action {
		case "SubmitAudio":
			submitCalls.Add(1)
			return providerTestResponse(req, http.StatusServiceUnavailable, failure)
		case "QueryAudio":
			if queryCalls.Add(1) == 1 {
				return providerTestResponse(req, http.StatusServiceUnavailable, failure)
			}
			return providerTestResponse(req, http.StatusOK, `{"Result":{"Status":"created"}}`)
		default:
			t.Errorf("unexpected action %q", action)
			return providerTestResponse(req, http.StatusBadRequest, `{}`)
		}
	}, WithMaxRetries(1))

	var calls sync.WaitGroup
	calls.Go(func() {
		got, err := client.Audio().SubmitAudio(t.Context(), audioTestInput())
		apiErr, ok := errors.AsType[*Error](err)
		if got != nil || !ok || apiErr.HTTPStatus != http.StatusServiceUnavailable || submitCalls.Load() != 1 {
			t.Errorf("submit = %+v, %v after %d calls, want one failed attempt", got, err, submitCalls.Load())
		}
	})
	calls.Go(func() {
		got, err := client.Audio().QueryAudio(t.Context(), "task-01")
		if err != nil || got == nil || got.Status != "created" || queryCalls.Load() != 2 {
			t.Errorf("query = %+v, %v after %d calls, want success on the second attempt", got, err, queryCalls.Load())
		}
	})
	calls.Wait()
}
