package volcenginex_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	volcx "github.com/gtkit/volcenginex"
)

func ExampleNew() {
	client, err := volcx.New(volcx.Config{AK: "example-ak", SK: "example-sk"})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			fmt.Println(closeErr)
		}
	}()
	fmt.Println(client != nil)
	// Output: true
}

func ExampleCountVolcMTChar() {
	fmt.Println(volcx.CountVolcMTChar("你好 🙂"))
	// Output: 4
}

func ExampleClient_GetUsage() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, `{"Result":{"Points":[{"Timestamp":20261124,"Value":3}]}}`); err != nil {
			return
		}
	}))
	defer server.Close()

	client, err := volcx.New(
		volcx.Config{AK: "example-ak", SK: "example-sk"},
		volcx.WithEndpoint(server.URL),
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

	day := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	usage, err := client.GetUsage(context.Background(), volcx.UsageServiceImage, day, day)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, point := range usage.Points {
		fmt.Println(point.Timestamp, point.Value)
	}
	// Output: 20261124 3
}
