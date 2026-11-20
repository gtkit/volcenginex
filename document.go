package volcenginex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gtkit/volcenginex/internal/trace"
)

// DocumentService wraps the asynchronous internationalization document APIs.
type DocumentService struct {
	client       *Client
	api          documentAPI
	poll         PollPolicy
	progressHook func(*TaskDetail)
	sleep        func(context.Context, time.Duration) error
}

// CreateProject creates an internationalization project.
func (s *DocumentService) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if strings.TrimSpace(req.ProjectName) == "" || len([]rune(req.ProjectName)) > 30 {
		return nil, wrapSentinel("DocumentCreate", ErrInvalidArgument, "project name is required and must be at most 30 characters")
	}
	if len([]rune(req.Comment)) > 140 {
		return nil, wrapSentinel("DocumentCreate", ErrInvalidArgument, "comment must be at most 140 characters")
	}
	if req.Role != "" && req.Role != "manager" && req.Role != "user" {
		return nil, wrapSentinel("DocumentCreate", ErrInvalidArgument, "role must be manager or user")
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.DocumentCreateWithContext(ctx, createProjectInput(req))
	if err != nil {
		return nil, providerError("DocumentCreate", err)
	}
	project := &Project{}
	if out != nil {
		project.RequestID = requestIDFromMetadata(out.Metadata)
	}
	if out != nil && out.ProjectID != nil {
		project.ProjectID = *out.ProjectID
	}
	return project, nil
}

// CreateTask creates a document translation task.
func (s *DocumentService) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if req.Deadline.IsZero() {
		req.Deadline = time.Now().Add(24 * time.Hour)
	}
	if err := validateCreateTask(req); err != nil {
		return nil, err
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.DocumentTaskCreateWithContext(ctx, createTaskInput(req))
	if err != nil {
		return nil, providerError("DocumentTaskCreate", err)
	}
	task := &Task{}
	if out == nil {
		return task, nil
	}
	task.RequestID = requestIDFromMetadata(out.Metadata)
	task.TaskID = valueOrZero(out.TaskID)
	task.SubtaskIDs = int32Values(out.SubtaskIDs)
	return task, nil
}

// TaskDetail returns the current task detail.
func (s *DocumentService) TaskDetail(ctx context.Context, projectID, taskID int32, raw bool) (*TaskDetail, error) {
	if ctx == nil || projectID <= 0 || taskID <= 0 {
		return nil, wrapSentinel("DocumentTaskDetail", ErrInvalidArgument, "project ID and task ID must be positive")
	}
	rawValue := int32(0)
	if raw {
		rawValue = 1
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	out, err := s.api.DocumentTaskDetailWithContext(ctx, &documentTaskDetailInput{
		ProjectID: new(projectID),
		TaskID:    new(taskID),
		Raw:       new(rawValue),
	})
	if err != nil {
		return nil, providerError("DocumentTaskDetail", err)
	}
	return taskDetailOutput(out), nil
}

// StopTask stops an asynchronous document task.
func (s *DocumentService) StopTask(ctx context.Context, projectID, taskID int64, subTaskIDs []int64) error {
	if ctx == nil || projectID <= 0 || taskID <= 0 {
		return wrapSentinel("DocumentTaskStop", ErrInvalidArgument, "project ID and task ID must be positive")
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	_, err := s.api.DocumentTaskStopWithContext(ctx, &documentTaskStopInput{
		ProjectID:  new(projectID),
		TaskID:     new(taskID),
		SubtaskIDs: int64Pointers(subTaskIDs),
	})
	return providerError("DocumentTaskStop", err)
}

// DeleteTask deletes an asynchronous document task and optionally its documents.
func (s *DocumentService) DeleteTask(ctx context.Context, taskID int64, deleteDoc bool) error {
	if ctx == nil || taskID <= 0 {
		return wrapSentinel("DocumentTaskDelete", ErrInvalidArgument, "task ID must be positive")
	}
	deleteValue := int64(0)
	if deleteDoc {
		deleteValue = 1
	}
	ctx = trace.WithRequestID(ctx, trace.NewRequestID())
	_, err := s.api.DocumentTaskDeleteWithContext(ctx, &documentTaskDeleteInput{
		TaskID:         new(taskID),
		DeleteDocument: new(deleteValue),
	})
	return providerError("DocumentTaskDelete", err)
}

// WaitTask waits for a task to reach a terminal state.
func (s *DocumentService) WaitTask(ctx context.Context, projectID, taskID int32, policy PollPolicy) (*TaskDetail, error) {
	return s.waitTask(ctx, projectID, taskID, policy.withDefaults())
}

// TranslateFile uploads local files, creates a task, and waits for its result.
func (s *DocumentService) TranslateFile(ctx context.Context, req FileTranslateRequest) (*Translation, error) {
	if ctx == nil {
		return nil, ErrInvalidArgument
	}
	if s.client.tos == nil {
		return nil, ErrTOSNotConfigured
	}
	docs := append([]DocInput(nil), req.Docs...)
	for _, path := range req.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, clientError("TOSUpload", err)
		}
		if info.Size() > maxFileBytes {
			return nil, formatLimitError(ErrFileTooLarge, "document %q exceeds %d bytes", path, maxFileBytes)
		}
		docType, err := docTypeFromPath(path)
		if err != nil {
			return nil, err
		}
		uploaded, err := s.client.tos.Upload(ctx, path)
		if err != nil {
			return nil, err
		}
		docs = append(docs, DocInput{Name: filepath.Base(path), Type: docType, URL: uploaded.URL, Size: uploaded.Size})
	}
	task, err := s.CreateTask(ctx, CreateTaskRequest{
		ProjectID:      req.ProjectID,
		TaskName:       req.TaskName,
		Deadline:       req.Deadline,
		Docs:           docs,
		SourceLocale:   req.SourceLocale,
		TargetLocales:  req.TargetLocales,
		Procedures:     req.Procedures,
		TranslatorType: req.TranslatorType,
		UseTM:          req.UseTM,
		ProofreadType:  req.ProofreadType,
		Comment:        req.Comment,
		Reference:      req.Reference,
	})
	if err != nil {
		return nil, err
	}
	detail, err := s.WaitTask(ctx, req.ProjectID, task.TaskID, s.poll)
	if err != nil {
		return nil, err
	}
	return &Translation{
		Task:                    task,
		Detail:                  detail,
		EstimatedCharacterCount: estimateDocumentCharacters(docs),
	}, nil
}

func int64Pointers(values []int64) []*int64 {
	result := make([]*int64, len(values))
	for i := range values {
		result[i] = new(values[i])
	}
	return result
}
