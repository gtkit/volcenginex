package dictionary_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gtkit/volcenginex/dictionary"
)

const chineseFixture = "../cedict/testdata/example.u8"

func openFixture(t *testing.T) *dictionary.Dictionary {
	t.Helper()
	englishPath := filepath.Join(t.TempDir(), "english #%.db")
	if err := writeEnglishFixture(englishPath); err != nil {
		t.Fatal(err)
	}
	dict, err := dictionary.Open(t.Context(), chineseFixture, englishPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dict.Close(); err != nil {
			t.Error(err)
		}
	})
	return dict
}

func TestLookup(t *testing.T) {
	t.Parallel()
	dict := openFixture(t)
	for _, tc := range []struct {
		name, word, hint, language, firstWord string
		firstDefinitions                      []string
		count                                 int
	}{
		{name: "simplified", word: "  苹果 ", language: "zh", firstWord: "苹果", firstDefinitions: []string{"Apple (technology company)"}, count: 2},
		{name: "traditional", word: "蘋果", hint: "zh", language: "zh", firstWord: "蘋果", firstDefinitions: []string{"Apple (technology company)"}, count: 2},
		{name: "english", word: "APPLE", language: "en", firstWord: "apple", firstDefinitions: []string{"苹果", "苹果树"}, count: 1},
		{name: "phrase", word: "look up", hint: "en", language: "en", firstWord: "look up", firstDefinitions: []string{"查找", "查阅"}, count: 1},
		{name: "null fields", word: "without-definition", language: "en", firstWord: "without-definition", count: 1},
		{name: "mixed with explicit language", word: "Go 语言", hint: "zh", language: "zh"},
		{name: "english miss", word: "absent", language: "en"},
		{name: "chinese miss", word: "词条未收录", language: "zh"},
		{name: "literal punctuation", word: "x' OR 1=1 --", hint: "en", language: "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := dict.Lookup(t.Context(), dictionary.LookupRequest{Word: tc.word, SourceLanguage: tc.hint})
			if err != nil || got.SourceLanguage != tc.language || len(got.Entries) != tc.count {
				t.Fatalf("Lookup() = %+v, %v, want %s with %d entries", got, err, tc.language, tc.count)
			}
			if tc.count == 0 {
				return
			}
			entry := got.Entries[0]
			if entry.Word != tc.firstWord || !slices.Equal(entry.Definitions, tc.firstDefinitions) {
				t.Errorf("first entry = %+v, want %s with %v", entry, tc.firstWord, tc.firstDefinitions)
			}
			if tc.language == "zh" && (entry.Simplified != "苹果" || entry.Traditional != "蘋果" || entry.Pronunciation != "Ping2 guo3") {
				t.Errorf("first Chinese entry = %+v, want both spellings and pinyin", entry)
			}
		})
	}
}

func TestLookupErrorsAndClose(t *testing.T) {
	t.Parallel()
	dict := openFixture(t)
	for _, tc := range []struct {
		word, hint, message string
	}{
		{word: "Go 语言", message: "mixed Chinese and English"},
		{word: "123!?", message: "cannot determine source language"},
		{word: "これは", message: "cannot determine source language"},
		{word: "苹果", hint: "ja", message: "source language must be zh or en"},
		{word: "  ", message: "non-blank UTF-8 text"},
		{word: "\xff", hint: "en", message: "non-blank UTF-8 text"},
	} {
		_, err := dict.Lookup(t.Context(), dictionary.LookupRequest{Word: tc.word, SourceLanguage: tc.hint})
		if err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Errorf("Lookup(%q, %q) = %v, want %q", tc.word, tc.hint, err, tc.message)
		}
		if tc.word == "Go 语言" && !errors.Is(err, dictionary.ErrMixedInput) {
			t.Errorf("Lookup(%q) = %v, want ErrMixedInput", tc.word, err)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := dict.Lookup(ctx, dictionary.LookupRequest{Word: "apple"}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled Lookup() = %v, want context.Canceled", err)
	}
	if err := dict.Close(); err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{"苹果", "apple"} {
		if _, err := dict.Lookup(t.Context(), dictionary.LookupRequest{Word: word}); err == nil {
			t.Errorf("Lookup(%q) after Close succeeded", word)
		}
	}
}

func TestOpenRejectsInvalidFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	badDatabase := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(badDatabase, []byte("not SQLite"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ chinese, english, message string }{
		{filepath.Join(dir, "missing.u8"), badDatabase, "load CC-CEDICT"},
		{chineseFixture, filepath.Join(dir, "missing.db"), "validate ECDICT schema"},
		{chineseFixture, badDatabase, "validate ECDICT schema"},
		{chineseFixture, "", "ECDICT path is required"},
	} {
		dict, err := dictionary.Open(t.Context(), tc.chinese, tc.english)
		if dict != nil || err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Errorf("Open(%q, %q) = %v, %v, want %q", tc.chinese, tc.english, dict, err, tc.message)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.db")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open created a missing database: %v", err)
	}
}
