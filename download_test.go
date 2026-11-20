package volcenginex

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gtkit/httpc"
)

func TestDownloadStreamsAndRenames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		_, _ = w.Write([]byte("hello world"))
	}))
	defer server.Close()
	client := httpc.New(httpc.WithHTTPClient(server.Client()))
	destination := filepath.Join(t.TempDir(), "result.txt")
	written, err := download(t.Context(), client, server.URL, destination)
	if err != nil {
		t.Fatalf("download returned error: %v", err)
	}
	if written != 11 {
		t.Fatalf("written = %d", written)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("destination = %q", data)
	}
}

func TestDownloadRejectsIncompleteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		_, _ = w.Write([]byte("bad"))
	}))
	defer server.Close()
	client := httpc.New(httpc.WithHTTPClient(server.Client()))
	destination := filepath.Join(t.TempDir(), "result.txt")
	_, err := download(t.Context(), client, server.URL, destination)
	if !errors.Is(err, ErrIncompleteDownload) {
		t.Fatalf("error = %v, want ErrIncompleteDownload", err)
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("destination stat error = %v", statErr)
	}
}

func TestDownloadRejectsInvalidInputsAndHTTPStatus(t *testing.T) {
	client := httpc.New()
	if _, err := download(t.Context(), client, "", "dest"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid URL error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	if _, err := download(t.Context(), client, server.URL, filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("bad status returned nil error")
	}
}
