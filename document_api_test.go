package volcenginex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type documentRoundTripper struct {
	requests  []*http.Request
	bodies    [][]byte
	responses []string
}

func (t *documentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, req)
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	t.bodies = append(t.bodies, body)
	responseBody := t.responses[len(t.requests)-1]
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(responseBody)),
		ContentLength: int64(len(responseBody)),
		Request:       req,
	}, nil
}

func newDocumentTestSession(t *testing.T, transport http.RoundTripper) *session.Session {
	t.Helper()
	sess, err := session.NewSession(
		volcengine.NewConfig().
			WithCredentials(credentials.NewStaticCredentials("ak", "sk", "")).
			WithRegion("cn-beijing").
			WithEndpoint("https://example.test").
			WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatalf("NewSession returned error: %v", err)
	}
	return sess
}

func TestDocumentAPIClientUsesOfficialProtocol(t *testing.T) {
	transport := &documentRoundTripper{responses: []string{
		`{"ResponseMetadata":{"RequestId":"project-request"},"Result":{"projectId":4}}`,
		`{"ResponseMetadata":{"RequestId":"task-request"},"Result":{"taskId":7,"subTaskIds":[8]}}`,
		`{"ResponseMetadata":{"RequestId":"detail-request"},"Result":{"taskInfo":{"taskId":7,"state":3}}}`,
		`{"ResponseMetadata":{"RequestId":"stop-request"},"Result":{}}`,
		`{"ResponseMetadata":{"RequestId":"delete-request"},"Result":{}}`,
	}}
	api := newDocumentAPI(newDocumentTestSession(t, transport))
	ctx := context.WithValue(t.Context(), documentContextKey{}, "preserved")

	project, err := api.DocumentCreateWithContext(ctx, &documentCreateInput{ProjectName: new("project")})
	if err != nil || project.Metadata == nil || project.Metadata.RequestId != "project-request" || project.ProjectID == nil || *project.ProjectID != 4 {
		t.Fatalf("project=%#v err=%v", project, err)
	}
	task, err := api.DocumentTaskCreateWithContext(ctx, &documentTaskCreateInput{ProjectID: new(int32(4)), TaskName: new("task")})
	if err != nil || task.TaskID == nil || *task.TaskID != 7 {
		t.Fatalf("task=%#v err=%v", task, err)
	}
	detail, err := api.DocumentTaskDetailWithContext(ctx, &documentTaskDetailInput{ProjectID: new(int32(4)), TaskID: new(int32(7)), Raw: new(int32(1))})
	if err != nil || detail.Metadata == nil || detail.Metadata.RequestId != "detail-request" || detail.TaskInfo == nil || detail.TaskInfo.ID == nil || *detail.TaskInfo.ID != 7 {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	if _, err := api.DocumentTaskStopWithContext(ctx, &documentTaskStopInput{ProjectID: new(int64(4)), TaskID: new(int64(7))}); err != nil {
		t.Fatalf("stop error: %v", err)
	}
	if _, err := api.DocumentTaskDeleteWithContext(ctx, &documentTaskDeleteInput{TaskID: new(int64(7))}); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	wantActions := []string{"DocumentCreate", "DocumentTaskCreate", "DocumentTaskDetail", "DocumentTaskStop", "DocumentTaskDelete"}
	for i, req := range transport.requests {
		if got := req.URL.Query().Get("Action"); got != wantActions[i] {
			t.Errorf("request %d action = %q, want %q", i, got, wantActions[i])
		}
		if got := req.URL.Query().Get("Version"); got != documentAPIVersion {
			t.Errorf("request %d version = %q, want %q", i, got, documentAPIVersion)
		}
		if req.Context().Value(documentContextKey{}) != "preserved" {
			t.Errorf("request %d lost caller context", i)
		}
	}
	if got := transport.requests[2].URL.Query().Get("projectId"); got != "4" {
		t.Errorf("detail projectId = %q, want 4", got)
	}
	if got := transport.requests[2].URL.Query().Get("taskId"); got != "7" {
		t.Errorf("detail taskId = %q, want 7", got)
	}
	if got := transport.requests[2].URL.Query().Get("raw"); got != "1" {
		t.Errorf("detail raw = %q, want 1", got)
	}
	if got := transport.requests[0].Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("create content type = %q", got)
	}
	if got := transport.requests[2].Header.Get("Content-Type"); got != "application/x-www-form-urlencoded; charset=utf-8" {
		t.Errorf("detail content type = %q", got)
	}
	var taskBody map[string]any
	if err := json.Unmarshal(transport.bodies[1], &taskBody); err != nil {
		t.Fatalf("decode task body: %v", err)
	}
	if taskBody["projectId"] != float64(4) || taskBody["taskName"] != "task" {
		t.Fatalf("task body = %#v", taskBody)
	}
}

type documentContextKey struct{}
