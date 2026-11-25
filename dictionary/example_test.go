package dictionary_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gtkit/volcenginex/dictionary"
)

func ExampleDictionary_Lookup() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "dictionary-example-")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Println(err)
		}
	}()

	englishPath := filepath.Join(dir, "english.db")
	if err := writeEnglishFixture(englishPath); err != nil {
		fmt.Println(err)
		return
	}
	dict, err := dictionary.Open(ctx, "../cedict/testdata/example.u8", englishPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := dict.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	for _, word := range []string{"苹果", "apple"} {
		result, err := dict.Lookup(ctx, dictionary.LookupRequest{Word: word})
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, entry := range result.Entries {
			fmt.Println(result.SourceLanguage, entry.Word, entry.Definitions)
		}
	}
	// Output:
	// zh 苹果 [Apple (technology company)]
	// zh 苹果 [apple (fruit) edible fruit]
	// en apple [苹果 苹果树]
}

func writeEnglishFixture(path string) (err error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close()) }()

	if _, err := db.Exec(`CREATE TABLE stardict (
		word TEXT COLLATE NOCASE UNIQUE NOT NULL,
		phonetic TEXT,
		translation TEXT
	)`); err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO stardict (word, phonetic, translation) VALUES
		('apple', 'æpl', '苹果' || char(10) || '苹果树'),
		('look up', NULL, '查找\n查阅'),
		('without-definition', NULL, NULL)`)
	return err
}
