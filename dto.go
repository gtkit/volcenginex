package volcenginex

import "time"

// TextTranslateRequest is a batch text translation request.
type TextTranslateRequest struct {
	_ struct{}

	SourceLanguage string
	TargetLanguage string
	TextList       []string
	// BatchSize limits the number of segments per request. Zero uses the default of 16.
	// Each request is capped at 16 segments and 5000 Unicode characters.
	BatchSize int
}

// TextTranslateResult contains translations in the same order as TextList.
type TextTranslateResult struct {
	_ struct{}

	Items []TextItem
	// CharacterCount is a local rune-based estimate for successful text batches.
	// It is for display or pre-deduction only. Final billing comes from the
	// Volcengine bill and GetUsageCommon.
	CharacterCount int
	Usage          *Usage
	RequestID      string
	Errors         []error
}

// TextItem is one translated text item.
type TextItem struct {
	_ struct{}

	Index                  int
	Source                 string
	Translation            string
	DetectedSourceLanguage string
}

// DetectItem is one language detection result.
type DetectItem struct {
	_ struct{}

	Index      int
	Language   string
	Confidence float64
}

// Usage contains usage points returned by the machine translation service.
type Usage struct {
	_ struct{}

	Points []UsagePoint
}

// UsagePoint is one usage measurement.
type UsagePoint struct {
	_ struct{}

	Timestamp int32
	Value     int32
}

// ImageTranslateRequest is an image translation request.
type ImageTranslateRequest struct {
	_ struct{}

	Image          []byte
	SourceLanguage string
	TargetLanguage string
}

// ImageTranslateResult contains translated image data and text blocks.
type ImageTranslateResult struct {
	_ struct{}

	ImageURL string
	// ImageCount is the number of images represented by this result.
	ImageCount int
	TextBlocks []ImageTextBlock
	RequestID  string
}

// ImageTextBlock is an OCR text block returned by image translation.
type ImageTextBlock struct {
	_ struct{}

	Text             string
	Translation      string
	DetectedLanguage string
	BackColor        []int32
	ForeColor        []int32
	Points           []Point
}

// Point is a point in an image text block.
type Point struct {
	_ struct{}

	X int32
	Y int32
}

// SubmitAudioRequest describes an audio or video file to translate.
type SubmitAudioRequest struct {
	_ struct{} // Require keyed initialization across packages.

	// URI is the media file URL that Volcengine can fetch.
	URI            string
	SourceLanguage string
	TargetLanguage string
}

// SubmitAudioResult identifies an accepted translation task.
type SubmitAudioResult struct {
	_ struct{} // Require keyed initialization across packages.

	TaskID    string
	RequestID string
}

// QueryAudioResult contains the current state and subtitles of a translation task.
type QueryAudioResult struct {
	_ struct{} // Require keyed initialization across packages.

	// Status is the task status returned by Volcengine.
	Status string
	// Confidence is nil when Volcengine omits it.
	Confidence *float64
	Subtitles  []AudioSubtitle
	RequestID  string
}

// AudioSubtitle is a time-aligned segment of recognized and translated text.
type AudioSubtitle struct {
	_ struct{} // Require keyed initialization across packages.

	// StartTime and EndTime use the values returned by Volcengine.
	StartTime   int32
	EndTime     int32
	Text        string
	Translation string
}

// CreateProjectRequest creates a document translation project.
type CreateProjectRequest struct {
	_ struct{}

	ProjectName string
	Members     []string
	Role        string
	Comment     string
}

// Project is a document translation project.
type Project struct {
	_ struct{}

	ProjectID int32
	RequestID string
}

// DocType identifies a document type supported by the document service.
type DocType int

const (
	DocTypeFeishuDoc   DocType = 1
	DocTypeDoc         DocType = 2
	DocTypeDocx        DocType = 3
	DocTypeXls         DocType = 4
	DocTypeXlsx        DocType = 5
	DocTypePpt         DocType = 6
	DocTypePptx        DocType = 7
	DocTypeCsv         DocType = 8
	DocTypeFeishuSheet DocType = 9
	DocTypeFeishuSlide DocType = 10
	DocTypeMd          DocType = 11
	DocTypePdf         DocType = 12
)

// Procedures identifies the document task workflow.
type Procedures int

const (
	ProceduresTranslate          Procedures = 1
	ProceduresProofread          Procedures = 2
	ProceduresTranslateProofread Procedures = 3
)

// DocInput describes one document submitted to a task.
type DocInput struct {
	_ struct{}

	Name      string
	Type      DocType
	URL       string
	Content   string
	WordCount string
	Size      int64
}

// CreateTaskRequest creates a document translation task.
type CreateTaskRequest struct {
	_ struct{}

	ProjectID      int32
	TaskName       string
	Deadline       time.Time
	Docs           []DocInput
	SourceLocale   string
	TargetLocales  []string
	Procedures     Procedures
	TranslatorType int32
	UseTM          bool
	ProofreadType  int32
	Comment        string
	Reference      *DocInput
}

// Task identifies a created document translation task.
type Task struct {
	_ struct{}

	TaskID     int32
	SubtaskIDs []int32
	RequestID  string
}

// TaskState is the state of a document translation task.
type TaskState int

const (
	TaskSubmitted  TaskState = 1
	TaskProcessing TaskState = 2
	TaskDone       TaskState = 3
	TaskStopped    TaskState = 4
	TaskDeleted    TaskState = 5
)

// TaskDetail is the package-owned task detail representation.
type TaskDetail struct {
	_ struct{}

	TaskID        int32
	TaskName      string
	State         TaskState
	SourceLocale  string
	TargetLocales []string
	Deadline      time.Time
	CreatedAt     time.Time
	Docs          []DocResult
	RequestID     string
}

// DocResult contains translated document metadata and subtasks.
type DocResult struct {
	_ struct{}

	DocID     int32
	WordCount int32
	DocName   string
	URL       string
	Locale    string
	DocType   DocType
	Subtasks  []Subtask
}

// Subtask contains one target locale result.
type Subtask struct {
	_ struct{}

	SubtaskID          int32
	TaskID             int32
	State              int32
	TargetLocale       string
	TargetDocID        int32
	TargetDocURL       string
	TotalRuneCount     string
	CompletedRuneCount string
	RawContent         string
	Procedures         []ProcedureInfo
}

// ProcedureInfo contains progress for one task procedure.
type ProcedureInfo struct {
	_ struct{}

	Procedure   int32
	State       int32
	Progress    int32
	Deadline    time.Time
	TargetDocID int32
	Comment     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FileTranslateRequest describes the document full-chain helper.
type FileTranslateRequest struct {
	_ struct{}

	ProjectID      int32
	TaskName       string
	Deadline       time.Time
	Files          []string
	Docs           []DocInput
	SourceLocale   string
	TargetLocales  []string
	Procedures     Procedures
	TranslatorType int32
	UseTM          bool
	ProofreadType  int32
	Comment        string
	Reference      *DocInput
	Destination    string
}

// Translation is the result of the document full-chain helper.
type Translation struct {
	_ struct{}

	Task   *Task
	Detail *TaskDetail
	// EstimatedCharacterCount is a local estimate from caller-provided inline
	// document content. Volcengine extracts uploaded document text internally,
	// so this value can be zero or differ from the final bill.
	EstimatedCharacterCount int
}

// UploadedObject describes an uploaded TOS object.
type UploadedObject struct {
	_ struct{}

	URL       string
	ObjectKey string
	ETag      string
	Size      int64
}
