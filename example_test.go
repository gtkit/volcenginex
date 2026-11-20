package volcenginex_test

import (
	"fmt"

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

func ExampleDocType() {
	fmt.Println(volcx.DocTypeDocx, volcx.DocTypePdf)
	// Output: 3 12
}

func ExampleCountVolcMTChar() {
	fmt.Println(volcx.CountVolcMTChar("你好 🙂"))
	// Output: 4
}
