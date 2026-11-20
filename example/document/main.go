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
	project, err := client.Document().CreateProject(context.Background(), volcx.CreateProjectRequest{ProjectName: "example"})
	if err != nil {
		fmt.Println("create project:", err)
		return
	}
	translation, err := client.Document().TranslateFile(context.Background(), volcx.FileTranslateRequest{ProjectID: project.ProjectID, TaskName: "example", Files: []string{"input.docx"}})
	if err != nil {
		fmt.Println("translate document:", err)
		return
	}
	fmt.Println(translation.Detail.TaskID)
}
