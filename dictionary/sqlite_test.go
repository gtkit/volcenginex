package dictionary

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenEnglishReadOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "english.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.ExecContext(t.Context(), "CREATE TABLE unrelated (id INTEGER)")
	if err := errors.Join(err, fixture.Close()); err != nil {
		t.Fatal(err)
	}
	if db, err := openEnglish(t.Context(), path); db != nil || err == nil || !strings.Contains(err.Error(), "no such table") {
		t.Fatalf("openEnglish without the stardict table = %v, %v", db, err)
	}
	fixture, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.ExecContext(t.Context(), "CREATE TABLE stardict (word TEXT, phonetic TEXT, translation TEXT)")
	if err := errors.Join(err, fixture.Close()); err != nil {
		t.Fatal(err)
	}

	db, err := openEnglish(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := db.ExecContext(t.Context(), "INSERT INTO stardict (word) VALUES ('apple')"); err == nil || !strings.Contains(err.Error(), "readonly") {
		t.Errorf("write attempt = %v, want a read-only error", err)
	}
}
