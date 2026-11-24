package volcenginex_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	volcx "github.com/gtkit/volcenginex"
)

func ExampleClient_Audio() {
	client, err := volcx.New(
		volcx.Config{AK: "example-ak", SK: "example-sk"},
		volcx.WithHTTPClient(&http.Client{Transport: audioExampleTransport{}}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	ctx := context.Background()
	task, err := client.Audio().SubmitAudio(ctx, volcx.SubmitAudioRequest{
		URI:            "https://media.example.com/sample.mp4",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := client.Audio().QueryAudio(ctx, task.TaskID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(task.TaskID, result.Status)
	for _, subtitle := range result.Subtitles {
		fmt.Println(subtitle.Translation)
	}
	// Output:
	// audio-task-1 success
	// 你好
}

type audioExampleTransport struct{}

func (audioExampleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	var body string
	switch action := req.URL.Query().Get("Action"); action {
	case "SubmitAudio":
		body = `{"Result":{"TaskId":"audio-task-1"}}`
	case "QueryAudio":
		body = `{"Result":{"Status":"success","Subtitles":[{"Text":"Hello","Translation":"你好"}]}}`
	default:
		return nil, fmt.Errorf("unexpected action %q", action)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}
