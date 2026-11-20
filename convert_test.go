package volcenginex

import (
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

func TestConvertOutputs(t *testing.T) {
	detail := taskDetailOutput(&documentTaskDetailOutput{
		TaskInfo: &documentTaskDetailTaskInfo{
			ID: new(int32(1)), Name: new("task"), State: new(int32(3)), SourceLocale: new("zh"),
			TargetLocale: []*string{new("en")}, Deadline: new("2026-11-21T10:00:00.000Z"), CreatedAt: new("2026-11-20T10:00:00.000Z"),
		},
		List: []*documentTaskDetailList{{
			Doc: &documentTaskDetailDocument{ID: new(int32(2)), WordCount: new(int32(3)), Name: new("a"), URL: new("url"), Locale: new("zh"), Type: new(int32(3))},
			Subtasks: []*documentTaskDetailSubtask{{
				ID: new(int32(4)), TaskID: new(int32(1)), State: new(int32(2)), TargetLocale: new("en"), TargetDocumentID: new(int32(5)), TargetDocumentURL: new("target"), TotalRuneCount: new("10"), CompletedRuneCount: new("5"), RawContent: new("raw"),
				Procedures: []*documentTaskDetailProcedure{{Procedure: new(int32(1)), State: new(int32(2)), Progress: new(int32(50)), Deadline: new("2026-11-21T10:00:00.000Z"), TargetDocument: new(int32(5)), Comment: new("c"), CreatedAt: new("2026-11-20T10:00:00.000Z"), UpdatedAt: new("2026-11-20T10:00:00.000Z")}},
			}},
		}},
	})
	if detail.State != TaskDone || len(detail.Docs) != 1 || detail.Docs[0].Subtasks[0].TargetDocURL != "target" || detail.Docs[0].Subtasks[0].Procedures[0].Progress != 50 {
		t.Fatalf("detail = %#v", detail)
	}

	image := imageOutput(&translate20250301.TranslateImageOutput{TextBlocks: []*translate20250301.TextBlockForTranslateImageOutput{{Text: new("a"), Translation: new("b"), DetectedLanguage: new("zh"), BackColor: []*int32{new(int32(1))}, ForeColor: []*int32{new(int32(2))}, Points: []*translate20250301.PointForTranslateImageOutput{{X: new(int32(3)), Y: new(int32(4))}}}}})
	if len(image.TextBlocks) != 1 || image.TextBlocks[0].Points[0].X != 3 {
		t.Fatalf("image = %#v", image)
	}
}
