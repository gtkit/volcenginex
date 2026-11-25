package volcenginex

import (
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

func TestImageOutput(t *testing.T) {
	image := imageOutput(&translate20250301.TranslateImageOutput{TextBlocks: []*translate20250301.TextBlockForTranslateImageOutput{{Text: new("a"), Translation: new("b"), DetectedLanguage: new("zh"), BackColor: []*int32{new(int32(1))}, ForeColor: []*int32{new(int32(2))}, Points: []*translate20250301.PointForTranslateImageOutput{{X: new(int32(3)), Y: new(int32(4))}}}}})
	if len(image.TextBlocks) != 1 || image.TextBlocks[0].Points[0].X != 3 {
		t.Fatalf("image = %#v", image)
	}
}
