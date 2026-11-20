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
	result, err := client.Text().Translate(context.Background(), volcx.TextTranslateRequest{TargetLanguage: "en", TextList: []string{"你好"}})
	if err != nil {
		fmt.Println("translate:", err)
		return
	}
	fmt.Println(result.Items[0].Translation)
}
