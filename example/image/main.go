package main

import (
	"context"
	"flag"
	"fmt"

	volcx "github.com/gtkit/volcenginex"
)

func main() {
	short := flag.Bool("short", false, "skip network calls")
	flag.Parse()
	if *short {
		return
	}
	client, err := volcx.New(volcx.Config{})
	if err != nil {
		fmt.Println("create client:", err)
		return
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			fmt.Println("close client:", closeErr)
		}
	}()
	result, err := client.Image().Translate(context.Background(), volcx.ImageTranslateRequest{Image: []byte("image-bytes"), TargetLanguage: "en"})
	if err != nil {
		fmt.Println("translate:", err)
		return
	}
	fmt.Println(result.ImageURL)
}
