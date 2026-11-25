package volcenginex

import "github.com/volcengine/volcengine-go-sdk/service/translate20250301"

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

func int32Values(values []*int32) []int32 {
	result := make([]int32, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
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
