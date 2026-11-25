package dictionary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // Register the SQLite driver with database/sql.
)

const englishLookupSQL = `SELECT word, COALESCE(phonetic, ''), COALESCE(translation, '')
	FROM stardict WHERE word = ?`

func openEnglish(ctx context.Context, path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("dictionary: ECDICT path is required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary: resolve ECDICT path: %w", err)
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath), RawQuery: "mode=ro"}
	if !strings.HasPrefix(uri.Path, "/") {
		uri.Path = "/" + uri.Path
	}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, fmt.Errorf("dictionary: open ECDICT: %w", err)
	}
	db.SetMaxOpenConns(4)

	rows, err := db.QueryContext(ctx, englishLookupSQL+" LIMIT 0", "")
	if err == nil {
		err = rows.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("dictionary: validate ECDICT schema: %w", errors.Join(err, db.Close()))
	}
	return db, nil
}

func lookupEnglish(ctx context.Context, db *sql.DB, word string) ([]Entry, error) {
	var entry Entry
	var translation string
	err := db.QueryRowContext(ctx, englishLookupSQL, word).Scan(&entry.Word, &entry.Pronunciation, &translation)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dictionary: query ECDICT: %w", err)
	}
	for line := range strings.SplitSeq(strings.ReplaceAll(translation, `\n`, "\n"), "\n") {
		if definition := strings.TrimSpace(line); definition != "" {
			entry.Definitions = append(entry.Definitions, definition)
		}
	}
	return []Entry{entry}, nil
}
