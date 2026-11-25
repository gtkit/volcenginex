// Package cedict provides offline Chinese-to-English lookup using CC-CEDICT v1.
package cedict

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"
)

// Entry is one CC-CEDICT record.
type Entry struct {
	_ struct{} // Require keyed literals for source compatibility.

	Traditional string   // Traditional Chinese headword.
	Simplified  string   // Simplified Chinese headword.
	Pinyin      string   // Pinyin with tone numbers as stored in the file.
	Definitions []string // Slash-delimited definitions in source order.
}

// Dictionary holds a read-only index of simplified and traditional headwords.
// It is safe for concurrent use.
type Dictionary struct {
	entries map[string][]Entry
}

// LoadFile loads an uncompressed UTF-8 CC-CEDICT v1 file of any extension.
// It returns an error for an unreadable file, malformed entry or empty dictionary.
func LoadFile(path string) (*Dictionary, error) {
	// #nosec G304 -- the caller explicitly supplies the local dictionary path.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cedict: open file: %w", err)
	}

	dictionary, readErr := readDictionary(file)
	if err := file.Close(); err != nil {
		return nil, errors.Join(readErr, fmt.Errorf("cedict: close file: %w", err))
	}
	return dictionary, readErr
}

// Lookup returns independent copies of all entries for word, in file order.
// Surrounding whitespace is ignored. Lookup returns nil when no entry matches.
func (d *Dictionary) Lookup(word string) []Entry {
	if d == nil {
		return nil
	}
	entries := slices.Clone(d.entries[strings.TrimSpace(word)])
	for i := range entries {
		entries[i].Definitions = slices.Clone(entries[i].Definitions)
	}
	return entries
}

func readDictionary(r io.Reader) (*Dictionary, error) {
	entries := make(map[string][]Entry)
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if lineNumber == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		if !utf8.ValidString(line) {
			return nil, fmt.Errorf("cedict: line %d: invalid UTF-8", lineNumber)
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry, ok := parseEntry(line)
		if !ok {
			return nil, fmt.Errorf("cedict: line %d: malformed CC-CEDICT v1 entry", lineNumber)
		}
		entries[entry.Simplified] = append(entries[entry.Simplified], entry)
		if entry.Traditional != entry.Simplified {
			entries[entry.Traditional] = append(entries[entry.Traditional], entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cedict: read line %d: %w", lineNumber+1, err)
	}
	if len(entries) == 0 {
		return nil, errors.New("cedict: no dictionary entries")
	}
	return &Dictionary{entries: entries}, nil
}

func parseEntry(line string) (Entry, bool) {
	headwords, pronunciation, _ := strings.Cut(line, " [")
	pinyin, definitions, hasSeparator := strings.Cut(pronunciation, "] /")
	definitions, hasTerminator := strings.CutSuffix(definitions, "/")
	words := strings.Fields(headwords)
	if len(words) != 2 || !hasSeparator || !hasTerminator ||
		strings.TrimSpace(pinyin) == "" || strings.ContainsAny(pinyin, "[]") {
		return Entry{}, false
	}
	senses := strings.Split(definitions, "/")
	for _, sense := range senses {
		if strings.TrimSpace(sense) == "" {
			return Entry{}, false
		}
	}
	return Entry{
		Traditional: words[0],
		Simplified:  words[1],
		Pinyin:      pinyin,
		Definitions: senses,
	}, true
}
