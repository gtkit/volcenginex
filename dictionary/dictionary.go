// Package dictionary provides offline Chinese-English and English-Chinese lookup.
package dictionary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/gtkit/volcenginex/cedict"
)

// ErrMixedInput means automatic routing found both Han and Latin letters.
var ErrMixedInput = errors.New("dictionary: mixed Chinese and English input")

// LookupRequest specifies the word or short phrase to look up.
type LookupRequest struct {
	_ struct{} // Require keyed literals.

	Word           string // Exact word or phrase. Surrounding whitespace is ignored.
	SourceLanguage string // "zh", "en", or empty for automatic routing.
}

// Entry contains the headword and its translations.
type Entry struct {
	_ struct{} // Require keyed literals.

	Word          string   // Matched word. Chinese entries use the requested spelling.
	Pronunciation string   // Chinese pinyin or English phonetics, when available.
	Definitions   []string // Translations in dictionary order.
	Simplified    string   // Simplified headword, when applicable.
	Traditional   string   // Traditional headword, when applicable.
}

// LookupResult identifies the language used and the matches found.
type LookupResult struct {
	_ struct{} // Require keyed literals.

	SourceLanguage string  // "zh" or "en".
	Entries        []Entry // Nil if the headword was not found.
}

// Dictionary holds the Chinese index and an English SQLite connection pool.
// It is safe for concurrent use. Create it with Open and release it with Close.
type Dictionary struct {
	chinese *cedict.Dictionary
	english *sql.DB
	closed  atomic.Bool
}

// Open loads the CC-CEDICT text file and opens the ECDICT Ultimate database
// read-only. Both arguments are filesystem paths.
func Open(ctx context.Context, cedictPath, ecdictPath string) (*Dictionary, error) {
	if ctx == nil {
		return nil, errors.New("dictionary: context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	chinese, err := cedict.LoadFile(cedictPath)
	if err != nil {
		return nil, fmt.Errorf("dictionary: load CC-CEDICT: %w", err)
	}
	english, err := openEnglish(ctx, ecdictPath)
	if err != nil {
		return nil, err
	}
	return &Dictionary{chinese: chinese, english: english}, nil
}

// Lookup returns independent matches for one word or phrase.
// In automatic mode, Han and Latin letters select the source language. An input
// containing both returns ErrMixedInput. An explicit language takes precedence.
func (d *Dictionary) Lookup(ctx context.Context, req LookupRequest) (LookupResult, error) {
	if ctx == nil {
		return LookupResult{}, errors.New("dictionary: context is required")
	}
	if err := ctx.Err(); err != nil {
		return LookupResult{}, err
	}
	if d == nil || d.english == nil || d.closed.Load() {
		return LookupResult{}, errors.New("dictionary: dictionary is not open")
	}
	word := strings.TrimSpace(req.Word)
	if word == "" || !utf8.ValidString(word) {
		return LookupResult{}, errors.New("dictionary: word must contain non-blank UTF-8 text")
	}
	language, err := resolveSourceLanguage(word, req.SourceLanguage)
	if err != nil {
		return LookupResult{}, err
	}
	if language == "zh" {
		return LookupResult{SourceLanguage: language, Entries: d.lookupChinese(word)}, nil
	}
	entries, err := lookupEnglish(ctx, d.english, word)
	if err != nil {
		return LookupResult{}, err
	}
	return LookupResult{SourceLanguage: language, Entries: entries}, nil
}

// Close releases the SQLite pool. Subsequent calls to Lookup return an error.
func (d *Dictionary) Close() error {
	if d == nil || d.english == nil {
		return nil
	}
	d.closed.Store(true)
	if err := d.english.Close(); err != nil {
		return fmt.Errorf("dictionary: close ECDICT: %w", err)
	}
	return nil
}

func resolveSourceLanguage(word, hint string) (string, error) {
	hint = strings.ToLower(strings.TrimSpace(hint))
	switch hint {
	case "zh", "en":
		return hint, nil
	case "":
	default:
		return "", errors.New("dictionary: source language must be zh or en")
	}

	var han, latin, otherLetters bool
	for _, r := range word {
		switch {
		case unicode.Is(unicode.Han, r):
			han = true
		case unicode.Is(unicode.Latin, r):
			latin = true
		case unicode.IsLetter(r):
			otherLetters = true
		}
	}
	switch {
	case han && latin:
		return "", ErrMixedInput
	case otherLetters || (!han && !latin):
		return "", errors.New("dictionary: cannot determine source language")
	case han:
		return "zh", nil
	default:
		return "en", nil
	}
}

func (d *Dictionary) lookupChinese(word string) []Entry {
	matches := d.chinese.Lookup(word)
	if len(matches) == 0 {
		return nil
	}
	entries := make([]Entry, len(matches))
	for i, match := range matches {
		entries[i] = Entry{
			Word:          word,
			Pronunciation: match.Pinyin,
			Definitions:   match.Definitions,
			Simplified:    match.Simplified,
			Traditional:   match.Traditional,
		}
	}
	return entries
}
