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
	usage, err := client.Text().Usage(context.Background())
	if err != nil {
		fmt.Println("usage:", err)
		return
	}
	fmt.Println(len(usage.Points))
}
