package volcenginex

import (
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

func textInput(req TextTranslateRequest, texts []string) *translate20250301.TranslateTextInput {
	values := make([]*string, len(texts))
	for i := range texts {
		values[i] = new(texts[i])
	}
	input := &translate20250301.TranslateTextInput{
		TargetLanguage: new(req.TargetLanguage),
		TextList:       values,
	}
	if req.SourceLanguage != "" {
		input.SourceLanguage = new(req.SourceLanguage)
	}
	return input
}

func langDetectInput(texts []string) *translate20250301.LangDetectInput {
	values := make([]*string, len(texts))
	for i := range texts {
		values[i] = new(texts[i])
	}
	return &translate20250301.LangDetectInput{TextList: values}
}

func imageInput(req ImageTranslateRequest, image string) *translate20250301.TranslateImageInput {
	input := &translate20250301.TranslateImageInput{
		Image:          new(image),
		TargetLanguage: new(req.TargetLanguage),
	}
	if req.SourceLanguage != "" {
		input.SourceLanguage = new(req.SourceLanguage)
	}
	return input
}

func createProjectInput(req CreateProjectRequest) *documentCreateInput {
	return &documentCreateInput{
		ProjectName: new(req.ProjectName),
		Members:     stringPointers(req.Members),
		Role:        optionalString(req.Role),
		Comment:     optionalString(req.Comment),
	}
}

func createTaskInput(req CreateTaskRequest) *documentTaskCreateInput {
	deadline := req.Deadline.UTC().Format("2006-01-02T15:04:05.000Z")
	sourceLocale := req.SourceLocale
	if sourceLocale == "" {
		sourceLocale = "zh"
	}
	targetLocales := req.TargetLocales
	if len(targetLocales) == 0 {
		targetLocales = []string{"en"}
	}
	procedures := req.Procedures
	if procedures == 0 {
		procedures = ProceduresTranslate
	}
	input := &documentTaskCreateInput{
		ProjectID:      new(req.ProjectID),
		TaskName:       new(req.TaskName),
		Deadline:       new(deadline),
		DocURLList:     docPointers(req.Docs),
		TranslatorType: optionalInt32(req.TranslatorType),
		Procedures:     new(int32(procedures)),
		SourceLocale:   new(sourceLocale),
		TargetLocale:   stringPointers(targetLocales),
		Comment:        optionalString(req.Comment),
		UseTM:          new(req.UseTM),
		ProofreadType:  optionalInt32(req.ProofreadType),
	}
	if req.Reference != nil {
		input.ReferenceDocItem = &documentTaskCreateReferenceInput{
			Name: optionalString(req.Reference.Name),
			Type: new(int32(req.Reference.Type)),
			URL:  optionalString(req.Reference.URL),
		}
	}
	return input
}

func docPointers(docs []DocInput) []*documentTaskCreateDocumentInput {
	values := make([]*documentTaskCreateDocumentInput, len(docs))
	for i := range docs {
		doc := docs[i]
		values[i] = &documentTaskCreateDocumentInput{
			Name:      optionalString(doc.Name),
			DocType:   new(int32(doc.Type)),
			URL:       optionalString(doc.URL),
			Content:   optionalString(doc.Content),
			WordCount: optionalString(doc.WordCount),
		}
	}
	return values
}

func taskDetailOutput(out *documentTaskDetailOutput) *TaskDetail {
	detail := &TaskDetail{}
	if out == nil {
		return detail
	}
	detail.RequestID = requestIDFromMetadata(out.Metadata)
	if info := out.TaskInfo; info != nil {
		detail.TaskID = valueOrZero(info.ID)
		detail.TaskName = valueOrEmpty(info.Name)
		detail.State = TaskState(valueOrZero(info.State))
		detail.SourceLocale = valueOrEmpty(info.SourceLocale)
		detail.TargetLocales = stringValues(info.TargetLocale)
		detail.Deadline = parseSDKTime(valueOrEmpty(info.Deadline))
		detail.CreatedAt = parseSDKTime(valueOrEmpty(info.CreatedAt))
	}
	detail.Docs = make([]DocResult, 0, len(out.List))
	for _, item := range out.List {
		if item == nil {
			continue
		}
		doc := DocResult{}
		if item.Doc != nil {
			doc.DocID = valueOrZero(item.Doc.ID)
			doc.WordCount = valueOrZero(item.Doc.WordCount)
			doc.DocName = valueOrEmpty(item.Doc.Name)
			doc.URL = valueOrEmpty(item.Doc.URL)
			doc.Locale = valueOrEmpty(item.Doc.Locale)
			doc.DocType = DocType(valueOrZero(item.Doc.Type))
		}
		doc.Subtasks = make([]Subtask, 0, len(item.Subtasks))
		for _, source := range item.Subtasks {
			if source == nil {
				continue
			}
			subtask := Subtask{
				SubtaskID:          valueOrZero(source.ID),
				TaskID:             valueOrZero(source.TaskID),
				State:              valueOrZero(source.State),
				TargetLocale:       valueOrEmpty(source.TargetLocale),
				TargetDocID:        valueOrZero(source.TargetDocumentID),
				TargetDocURL:       valueOrEmpty(source.TargetDocumentURL),
				TotalRuneCount:     valueOrEmpty(source.TotalRuneCount),
				CompletedRuneCount: valueOrEmpty(source.CompletedRuneCount),
				RawContent:         valueOrEmpty(source.RawContent),
			}
			subtask.Procedures = make([]ProcedureInfo, 0, len(source.Procedures))
			for _, procedure := range source.Procedures {
				if procedure == nil {
					continue
				}
				subtask.Procedures = append(subtask.Procedures, ProcedureInfo{
					Procedure:   valueOrZero(procedure.Procedure),
					State:       valueOrZero(procedure.State),
					Progress:    valueOrZero(procedure.Progress),
					Deadline:    parseSDKTime(valueOrEmpty(procedure.Deadline)),
					TargetDocID: valueOrZero(procedure.TargetDocument),
					Comment:     valueOrEmpty(procedure.Comment),
					CreatedAt:   parseSDKTime(valueOrEmpty(procedure.CreatedAt)),
					UpdatedAt:   parseSDKTime(valueOrEmpty(procedure.UpdatedAt)),
				})
			}
			doc.Subtasks = append(doc.Subtasks, subtask)
		}
		detail.Docs = append(detail.Docs, doc)
	}
	return detail
}

func imageOutput(out *translate20250301.TranslateImageOutput) *ImageTranslateResult {
	result := &ImageTranslateResult{}
	if out == nil {
		return result
	}
	result.ImageURL = valueOrEmpty(out.Image)
	result.RequestID = requestIDFromMetadata(out.Metadata)
	result.TextBlocks = make([]ImageTextBlock, 0, len(out.TextBlocks))
	for _, block := range out.TextBlocks {
		if block == nil {
			continue
		}
		item := ImageTextBlock{
			Text:             valueOrEmpty(block.Text),
			Translation:      valueOrEmpty(block.Translation),
			DetectedLanguage: valueOrEmpty(block.DetectedLanguage),
			BackColor:        int32Values(block.BackColor),
			ForeColor:        int32Values(block.ForeColor),
		}
		item.Points = make([]Point, 0, len(block.Points))
		for _, point := range block.Points {
			if point != nil {
				item.Points = append(item.Points, Point{X: valueOrZero(point.X), Y: valueOrZero(point.Y)})
			}
		}
		result.TextBlocks = append(result.TextBlocks, item)
	}
	return result
}

func usageOutput(out *translate20250301.GetUsageOutput) *Usage {
	usage := &Usage{}
	if out == nil {
		return usage
	}
	usage.Points = make([]UsagePoint, 0, len(out.Points))
	for _, point := range out.Points {
		if point != nil {
			usage.Points = append(usage.Points, UsagePoint{Timestamp: valueOrZero(point.Timestamp), Value: valueOrZero(point.Value)})
		}
	}
	return usage
}

func detectOutput(out *translate20250301.LangDetectOutput, indexes []int) []DetectItem {
	if out == nil {
		return nil
	}
	items := make([]DetectItem, 0, len(out.DetectedLanguageList))
	for i, detected := range out.DetectedLanguageList {
		if detected == nil {
			continue
		}
		index := i
		if i < len(indexes) {
			index = indexes[i]
		}
		items = append(items, DetectItem{Index: index, Language: valueOrEmpty(detected.Language), Confidence: valueOrFloat(detected.Confidence)})
	}
	return items
}

func stringPointers(values []string) []*string {
	result := make([]*string, len(values))
	for i := range values {
		result[i] = new(values[i])
	}
	return result
}

func stringValues(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func int32Values(values []*int32) []int32 {
	result := make([]int32, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return new(value)
}

func optionalInt32(value int32) *int32 {
	if value == 0 {
		return nil
	}
	return new(value)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func valueOrFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func parseSDKTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
