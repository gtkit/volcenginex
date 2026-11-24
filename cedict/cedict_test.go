package cedict_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gtkit/volcenginex/cedict"
)

func TestLoadFileAndLookup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "dictionary.txt")
	data := "\uFEFF# example\r\n\r\n蘋果 苹果 [Ping2 guo3] /Apple (company)/\r\n" +
		"蘋果 苹果 [ping2 guo3] /apple (fruit)/edible fruit/\r\n" +
		"你好 你好 [ni3 hao3] /hello; hi/"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	dict, err := cedict.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []cedict.Entry{
		{Traditional: "蘋果", Simplified: "苹果", Pinyin: "Ping2 guo3", Definitions: []string{"Apple (company)"}},
		{Traditional: "蘋果", Simplified: "苹果", Pinyin: "ping2 guo3", Definitions: []string{"apple (fruit)", "edible fruit"}},
	}
	for _, word := range []string{"苹果", "蘋果", "  苹果\t"} {
		t.Run(word, func(t *testing.T) {
			t.Parallel()
			got := dict.Lookup(word)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Lookup(%q) = %+v, want %+v", word, got, want)
			}
			got[0].Definitions[0] = "modified"
			got[1] = cedict.Entry{}
			if again := dict.Lookup(word); !reflect.DeepEqual(again, want) {
				t.Errorf("lookup after mutating the result = %+v, want %+v", again, want)
			}
		})
	}

	hello := dict.Lookup("你好")
	if len(hello) != 1 || !reflect.DeepEqual(hello[0].Definitions, []string{"hello; hi"}) {
		t.Errorf("Lookup(你好) = %+v, want one entry with its original gloss", hello)
	}
	for _, word := range []string{"", " \n", "苹果很好吃"} {
		if got := dict.Lookup(word); got != nil {
			t.Errorf("Lookup(%q) = %+v, want nil", word, got)
		}
	}
}

func TestLoadFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if dict, err := cedict.LoadFile(filepath.Join(dir, "missing.u8")); dict != nil || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("LoadFile(missing) = %v, %v, want nil and a missing-file error", dict, err)
	}
	if dict, err := cedict.LoadFile(dir); dict != nil || err == nil {
		t.Errorf("LoadFile(directory) = %v, %v, want nil and a file error", dict, err)
	}

	for _, tc := range []struct {
		name string
		data string
		want string
	}{
		{name: "empty", data: "# header\n\n", want: "no dictionary entries"},
		{name: "invalid UTF-8", data: "# header\n\xff", want: "line 2: invalid UTF-8"},
		{name: "truncated definitions", data: "你好 你好 [ni3 hao3] /hello/\n蘋果 苹果 [ping2 guo3] /apple", want: "line 2: malformed"},
		{name: "missing word", data: "苹果 [ping2 guo3] /apple/", want: "line 1: malformed"},
		{name: "empty definition", data: "苹果 苹果 [ping2 guo3] /apple//", want: "line 1: malformed"},
		{name: "v2 pronunciation", data: "苹果 苹果 [[ping2guo3]] /apple/", want: "line 1: malformed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "bad.u8")
			if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
				t.Fatal(err)
			}
			dict, err := cedict.LoadFile(path)
			if dict != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("LoadFile() = %v, %v, want nil and an error containing %q", dict, err, tc.want)
			}
		})
	}
}

func TestLookupUninitialized(t *testing.T) {
	t.Parallel()

	for _, dict := range []*cedict.Dictionary{nil, {}} {
		if got := dict.Lookup("苹果"); got != nil {
			t.Errorf("Lookup() = %+v, want nil", got)
		}
	}
}

func BenchmarkLookup(b *testing.B) {
	dict, err := cedict.LoadFile("testdata/example.u8")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("serial", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = dict.Lookup("苹果")
		}
	})
	b.Run("parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = dict.Lookup("苹果")
			}
		})
	})
}
