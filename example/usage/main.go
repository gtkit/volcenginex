package main

import (
	"context"
	"flag"
	"fmt"
	"time"

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
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	usage, err := client.GetUsage(context.Background(), volcx.UsageServiceText, from, now)
	if err != nil {
		fmt.Println("usage:", err)
		return
	}
	fmt.Println(len(usage.Points))
}
