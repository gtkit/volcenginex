package cedict_test

import (
	"fmt"

	"github.com/gtkit/volcenginex/cedict"
)

func ExampleLoadFile() {
	dictionary, err := cedict.LoadFile("testdata/example.u8")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, entry := range dictionary.Lookup("苹果") {
		fmt.Println(entry.Traditional, entry.Pinyin, entry.Definitions[0])
	}
	// Output:
	// 蘋果 Ping2 guo3 Apple (technology company)
	// 蘋果 ping2 guo3 apple (fruit)
}
