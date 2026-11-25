package volcenginex

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
