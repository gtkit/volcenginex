package volcenginex

import (
	"context"

	"github.com/volcengine/volcengine-go-sdk/volcengine/client"
	"github.com/volcengine/volcengine-go-sdk/volcengine/client/metadata"
	"github.com/volcengine/volcengine-go-sdk/volcengine/corehandlers"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"github.com/volcengine/volcengine-go-sdk/volcengine/signer/volc"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcenginequery"
)

const (
	documentServiceName = "i18n_openapi"
	documentAPIVersion  = "2021-05-21"
)

type documentAPI interface {
	DocumentCreateWithContext(context.Context, *documentCreateInput) (*documentCreateOutput, error)
	DocumentTaskCreateWithContext(context.Context, *documentTaskCreateInput) (*documentTaskCreateOutput, error)
	DocumentTaskDetailWithContext(context.Context, *documentTaskDetailInput) (*documentTaskDetailOutput, error)
	DocumentTaskStopWithContext(context.Context, *documentTaskStopInput) (*documentTaskStopOutput, error)
	DocumentTaskDeleteWithContext(context.Context, *documentTaskDeleteInput) (*documentTaskDeleteOutput, error)
}

type documentAPIClient struct {
	sdk *client.Client
}

func newDocumentAPI(sess *session.Session) *documentAPIClient {
	config := sess.ClientConfig(documentServiceName)
	sdk := client.New(
		*config.Config,
		metadata.ClientInfo{
			ServiceName:   documentServiceName,
			ServiceID:     documentServiceName,
			SigningName:   config.SigningName,
			SigningRegion: config.SigningRegion,
			Endpoint:      config.Endpoint,
			APIVersion:    documentAPIVersion,
		},
		config.Handlers,
	)
	sdk.Handlers.Build.PushBackNamed(corehandlers.SDKVersionUserAgentHandler)
	sdk.Handlers.Build.PushBackNamed(corehandlers.AddHostExecEnvUserAgentHandler)
	sdk.Handlers.Sign.PushBackNamed(volc.SignRequestHandler)
	sdk.Handlers.Build.PushBackNamed(volcenginequery.BuildHandler)
	sdk.Handlers.Unmarshal.PushBackNamed(volcenginequery.UnmarshalHandler)
	sdk.Handlers.UnmarshalMeta.PushBackNamed(volcenginequery.UnmarshalMetaHandler)
	sdk.Handlers.UnmarshalError.PushBackNamed(volcenginequery.UnmarshalErrorHandler)
	return &documentAPIClient{sdk: sdk}
}

func (c *documentAPIClient) call(ctx context.Context, action, method string, input, output any, jsonBody bool) error {
	req := c.sdk.NewRequest(&request.Operation{
		Name:       action,
		HTTPMethod: method,
		HTTPPath:   "/",
	}, input, output)
	if jsonBody {
		req.HTTPRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	req.SetContext(ctx)
	return req.Send()
}

func (c *documentAPIClient) DocumentCreateWithContext(ctx context.Context, input *documentCreateInput) (*documentCreateOutput, error) {
	output := &documentCreateOutput{}
	err := c.call(ctx, "DocumentCreate", "POST", input, output, true)
	return output, err
}

func (c *documentAPIClient) DocumentTaskCreateWithContext(ctx context.Context, input *documentTaskCreateInput) (*documentTaskCreateOutput, error) {
	output := &documentTaskCreateOutput{}
	err := c.call(ctx, "DocumentTaskCreate", "POST", input, output, true)
	return output, err
}

func (c *documentAPIClient) DocumentTaskDetailWithContext(ctx context.Context, input *documentTaskDetailInput) (*documentTaskDetailOutput, error) {
	output := &documentTaskDetailOutput{}
	err := c.call(ctx, "DocumentTaskDetail", "GET", input, output, false)
	return output, err
}

func (c *documentAPIClient) DocumentTaskStopWithContext(ctx context.Context, input *documentTaskStopInput) (*documentTaskStopOutput, error) {
	output := &documentTaskStopOutput{}
	err := c.call(ctx, "DocumentTaskStop", "POST", input, output, true)
	return output, err
}

func (c *documentAPIClient) DocumentTaskDeleteWithContext(ctx context.Context, input *documentTaskDeleteInput) (*documentTaskDeleteOutput, error) {
	output := &documentTaskDeleteOutput{}
	err := c.call(ctx, "DocumentTaskDelete", "POST", input, output, true)
	return output, err
}

type documentCreateInput struct {
	Comment     *string   `json:"comment,omitempty"`
	Members     []*string `json:"members,omitempty"`
	ProjectName *string   `json:"projectName,omitempty"`
	Role        *string   `json:"role,omitempty"`
}

type documentCreateOutput struct {
	Metadata  *response.ResponseMetadata
	ProjectID *int32 `json:"projectId,omitempty"`
}

type documentTaskCreateInput struct {
	Comment          *string                            `json:"comment,omitempty"`
	Deadline         *string                            `json:"deadline,omitempty"`
	DocURLList       []*documentTaskCreateDocumentInput `json:"docUrlList,omitempty"`
	Procedures       *int32                             `json:"procedures,omitempty"`
	ProjectID        *int32                             `json:"projectId,omitempty"`
	ProofreadType    *int32                             `json:"proofreadType,omitempty"`
	ReferenceDocItem *documentTaskCreateReferenceInput  `json:"referenceDocItem,omitempty"`
	SourceLocale     *string                            `json:"sourceLocale,omitempty"`
	TargetLocale     []*string                          `json:"targetLocale,omitempty"`
	TaskName         *string                            `json:"taskName,omitempty"`
	TranslatorType   *int32                             `json:"translatorType,omitempty"`
	UseTM            *bool                              `json:"useTM,omitempty"`
}

type documentTaskCreateDocumentInput struct {
	Content   *string `json:"content,omitempty"`
	DocType   *int32  `json:"docType,omitempty"`
	Name      *string `json:"name,omitempty"`
	URL       *string `json:"url,omitempty"`
	WordCount *string `json:"wordCount,omitempty"`
}

type documentTaskCreateReferenceInput struct {
	Name *string `json:"referenceDocName,omitempty"`
	Type *int32  `json:"referenceDocType,omitempty"`
	URL  *string `json:"referenceDocUrl,omitempty"`
}

type documentTaskCreateOutput struct {
	Metadata   *response.ResponseMetadata
	SubtaskIDs []*int32 `json:"subTaskIds,omitempty"`
	TaskID     *int32   `json:"taskId,omitempty"`
}

type documentTaskDetailInput struct {
	ProjectID *int32 `locationName:"projectId"`
	Raw       *int32 `locationName:"raw"`
	TaskID    *int32 `locationName:"taskId"`
}

type documentTaskDetailOutput struct {
	Metadata *response.ResponseMetadata
	List     []*documentTaskDetailList   `json:"list"`
	TaskInfo *documentTaskDetailTaskInfo `json:"taskInfo"`
}

type documentTaskDetailList struct {
	Doc      *documentTaskDetailDocument  `json:"doc"`
	Subtasks []*documentTaskDetailSubtask `json:"subtasks"`
}

type documentTaskDetailDocument struct {
	ID        *int32  `json:"docId"`
	Name      *string `json:"docName"`
	Type      *int32  `json:"docType"`
	Locale    *string `json:"locale"`
	URL       *string `json:"url"`
	WordCount *int32  `json:"wordCount"`
}

type documentTaskDetailSubtask struct {
	CompletedRuneCount *string                        `json:"completedRuneCount"`
	Procedures         []*documentTaskDetailProcedure `json:"procedures"`
	RawContent         *string                        `json:"rawContent"`
	State              *int32                         `json:"state"`
	ID                 *int32                         `json:"subTaskId"`
	TargetDocumentID   *int32                         `json:"targetDocId"`
	TargetDocumentURL  *string                        `json:"targetDocUrl"`
	TargetLocale       *string                        `json:"targetLocale"`
	TaskID             *int32                         `json:"taskId"`
	TotalRuneCount     *string                        `json:"totalRuneCount"`
}

type documentTaskDetailProcedure struct {
	Comment        *string `json:"comment"`
	CreatedAt      *string `json:"createdAt"`
	Deadline       *string `json:"deadline"`
	Procedure      *int32  `json:"procedure"`
	Progress       *int32  `json:"progress"`
	State          *int32  `json:"state"`
	TargetDocument *int32  `json:"targetDocId"`
	UpdatedAt      *string `json:"updatedAt"`
}

type documentTaskDetailTaskInfo struct {
	CreatedAt    *string   `json:"createdAt"`
	Deadline     *string   `json:"deadline"`
	ID           *int32    `json:"taskId"`
	Name         *string   `json:"taskName"`
	SourceLocale *string   `json:"sourceLocale"`
	State        *int32    `json:"state"`
	TargetLocale []*string `json:"targetLocale"`
}

type documentTaskStopInput struct {
	ProjectID  *int64   `json:"projectId,omitempty"`
	SubtaskIDs []*int64 `json:"subTaskIds,omitempty"`
	TaskID     *int64   `json:"taskId,omitempty"`
}

type documentTaskStopOutput struct {
	Metadata *response.ResponseMetadata
}

type documentTaskDeleteInput struct {
	DeleteDocument *int64 `json:"deleteDoc,omitempty"`
	TaskID         *int64 `json:"taskId,omitempty"`
}

type documentTaskDeleteOutput struct {
	Metadata *response.ResponseMetadata
}
