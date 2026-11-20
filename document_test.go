package volcenginex

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/i18nopenapi"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
)

type fakeDocumentAPI struct {
	i18nopenapi.I18NOPENAPIAPI
	createTaskInput *i18nopenapi.DocumentTaskCreateInput
	details         []*i18nopenapi.DocumentTaskDetailOutput
	detailCalls     int
	detailErr       error
}

func (f *fakeDocumentAPI) DocumentCreateWithContext(_ context.Context, _ *i18nopenapi.DocumentCreateInput, _ ...request.Option) (*i18nopenapi.DocumentCreateOutput, error) {
	return &i18nopenapi.DocumentCreateOutput{ProjectId: new(int32(4))}, nil
}

func (f *fakeDocumentAPI) DocumentTaskCreateWithContext(_ context.Context, input *i18nopenapi.DocumentTaskCreateInput, _ ...request.Option) (*i18nopenapi.DocumentTaskCreateOutput, error) {
	f.createTaskInput = input
	return &i18nopenapi.DocumentTaskCreateOutput{TaskId: new(int32(7)), SubTaskIds: []*int32{new(int32(8))}}, nil
}

func (f *fakeDocumentAPI) DocumentTaskDetailWithContext(_ context.Context, _ *i18nopenapi.DocumentTaskDetailInput, _ ...request.Option) (*i18nopenapi.DocumentTaskDetailOutput, error) {
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	if f.detailCalls >= len(f.details) {
		return f.details[len(f.details)-1], nil
	}
	result := f.details[f.detailCalls]
	f.detailCalls++
	return result, nil
}

func (f *fakeDocumentAPI) DocumentTaskStopWithContext(_ context.Context, _ *i18nopenapi.DocumentTaskStopInput, _ ...request.Option) (*i18nopenapi.DocumentTaskStopOutput, error) {
	return &i18nopenapi.DocumentTaskStopOutput{}, nil
}

func (f *fakeDocumentAPI) DocumentTaskDeleteWithContext(_ context.Context, _ *i18nopenapi.DocumentTaskDeleteInput, _ ...request.Option) (*i18nopenapi.DocumentTaskDeleteOutput, error) {
	return &i18nopenapi.DocumentTaskDeleteOutput{}, nil
}

func TestDocumentProjectStopDelete(t *testing.T) {
	service := &DocumentService{api: &fakeDocumentAPI{}}
	project, err := service.CreateProject(t.Context(), CreateProjectRequest{ProjectName: "project", Role: "manager"})
	if err != nil || project.ProjectID != 4 {
		t.Fatalf("project=%#v err=%v", project, err)
	}
	if err := service.StopTask(t.Context(), 1, 2, []int64{3}); err != nil {
		t.Fatalf("StopTask error: %v", err)
	}
	if err := service.DeleteTask(t.Context(), 2, true); err != nil {
		t.Fatalf("DeleteTask error: %v", err)
	}
}

func TestCreateProjectValidation(t *testing.T) {
	service := &DocumentService{api: &fakeDocumentAPI{}}
	invalid := []CreateProjectRequest{
		{},
		{ProjectName: "0123456789012345678901234567891"},
		{ProjectName: "project", Comment: "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901"},
		{ProjectName: "project", Role: "owner"},
	}
	for _, request := range invalid {
		if _, err := service.CreateProject(t.Context(), request); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("request %#v error = %v", request, err)
		}
	}
}

func TestCreateTaskMapsFieldsAndDeadline(t *testing.T) {
	fake := &fakeDocumentAPI{}
	service := &DocumentService{api: fake}
	deadline := time.Date(2026, 11, 21, 10, 0, 0, 123000000, time.FixedZone("UTC+8", 8*60*60))
	_, err := service.CreateTask(t.Context(), CreateTaskRequest{
		ProjectID:     3,
		TaskName:      "task",
		Deadline:      deadline,
		Docs:          []DocInput{{Name: "a.md", Type: DocTypeMd, URL: "https://example/a.md"}},
		TargetLocales: []string{"en"},
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if fake.createTaskInput.ProjectId == nil || *fake.createTaskInput.ProjectId != 3 {
		t.Fatalf("project ID = %#v", fake.createTaskInput.ProjectId)
	}
	if got := *fake.createTaskInput.Deadline; got != "2026-11-21T02:00:00.123Z" {
		t.Fatalf("deadline = %q", got)
	}
	if got := *fake.createTaskInput.DocUrlList[0].Url; got != "https://example/a.md" {
		t.Fatalf("URL = %q", got)
	}
}

func TestDocumentLimitsRejectUnsupportedType(t *testing.T) {
	err := validateCreateTask(CreateTaskRequest{
		ProjectID: 1,
		TaskName:  "task",
		Docs:      []DocInput{{Name: "a.doc", Type: DocTypeDoc}},
	})
	if !errors.Is(err, ErrDocTypeUnsupported) {
		t.Fatalf("error = %v, want ErrDocTypeUnsupported", err)
	}
}

func TestDocumentLimitsAndPathTypes(t *testing.T) {
	if err := validateCreateTask(CreateTaskRequest{ProjectID: 1, TaskName: "task", Docs: make([]DocInput, 11)}); !errors.Is(err, ErrTooManyDocs) {
		t.Fatalf("too many docs error = %v", err)
	}
	if err := validateCreateTask(CreateTaskRequest{ProjectID: 1, TaskName: "task", Docs: []DocInput{{Type: DocTypePdf, Size: maxFileBytes + 1}}}); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("large file error = %v", err)
	}
	if _, err := docTypeFromPath("file.unknown"); !errors.Is(err, ErrDocTypeUnsupported) {
		t.Fatalf("unknown extension error = %v", err)
	}
	if got, err := docTypeFromPath("file.xlsx"); err != nil || got != DocTypeXlsx {
		t.Fatalf("xlsx type=%v err=%v", got, err)
	}
	for _, path := range []string{"a.docx", "a.xls", "a.ppt", "a.pptx", "a.csv", "a.md", "a.markdown", "a.pdf"} {
		if _, err := docTypeFromPath(path); filepath.Ext(path) == ".xls" || filepath.Ext(path) == ".ppt" || filepath.Ext(path) == ".csv" {
			if !errors.Is(err, ErrDocTypeUnsupported) {
				t.Errorf("path %q error = %v", path, err)
			}
		} else if err != nil {
			t.Errorf("path %q error = %v", path, err)
		}
	}
}

func TestTranslateFileRequiresTOS(t *testing.T) {
	service := &DocumentService{client: &Client{}}
	_, err := service.TranslateFile(t.Context(), FileTranslateRequest{ProjectID: 1})
	if !errors.Is(err, ErrTOSNotConfigured) {
		t.Fatalf("error = %v, want ErrTOSNotConfigured", err)
	}
}

func TestWaitTaskCompletesAndCallsProgress(t *testing.T) {
	fake := &fakeDocumentAPI{details: []*i18nopenapi.DocumentTaskDetailOutput{
		{TaskInfo: &i18nopenapi.TaskInfoForDocumentTaskDetailOutput{State: new(int32(2)), TaskId: new(int32(9))}},
		{TaskInfo: &i18nopenapi.TaskInfoForDocumentTaskDetailOutput{State: new(int32(3)), TaskId: new(int32(9))}},
	}}
	progress := 0
	service := &DocumentService{
		api:          fake,
		progressHook: func(*TaskDetail) { progress++ },
		sleep:        func(context.Context, time.Duration) error { return nil },
	}
	detail, err := service.WaitTask(t.Context(), 1, 9, PollPolicy{Interval: time.Nanosecond, MaxInterval: time.Nanosecond, Timeout: time.Second})
	if err != nil {
		t.Fatalf("WaitTask returned error: %v", err)
	}
	if detail.State != TaskDone || progress != 2 {
		t.Fatalf("detail=%#v progress=%d", detail, progress)
	}
}

func TestWaitTaskAborted(t *testing.T) {
	fake := &fakeDocumentAPI{details: []*i18nopenapi.DocumentTaskDetailOutput{
		{TaskInfo: &i18nopenapi.TaskInfoForDocumentTaskDetailOutput{State: new(int32(4))}},
	}}
	service := &DocumentService{api: fake, sleep: func(context.Context, time.Duration) error { return nil }}
	_, err := service.WaitTask(t.Context(), 1, 9, PollPolicy{Timeout: time.Second})
	if !errors.Is(err, ErrTaskAborted) {
		t.Fatalf("error = %v, want ErrTaskAborted", err)
	}
}

func TestWaitTaskExhaustsFailuresAndHonorsCancellation(t *testing.T) {
	service := &DocumentService{api: &fakeDocumentAPI{detailErr: errors.New("network")}, sleep: func(context.Context, time.Duration) error { return nil }}
	_, err := service.WaitTask(t.Context(), 1, 9, PollPolicy{Timeout: time.Second})
	if !errors.Is(err, ErrPollExhausted) {
		t.Fatalf("error = %v, want ErrPollExhausted", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service.WaitTask(ctx, 1, 9, PollPolicy{Timeout: time.Second})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
}
