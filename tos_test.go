package volcenginex

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gtkit/httpc"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

func TestTOSUploadBytesSendsRawBody(t *testing.T) {
	var got []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		got, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("ETag", "etag-1")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	service := &TOSService{
		httpClient: httpc.New(httpc.WithHTTPClient(server.Client())),
		bucket:     "bucket",
		signer: func(*tos.PreSignedURLInput) (*tos.PreSignedURLOutput, error) {
			return &tos.PreSignedURLOutput{SignedUrl: server.URL}, nil
		},
	}
	result, err := service.UploadBytes(t.Context(), "file.md", []byte("raw bytes"))
	if err != nil {
		t.Fatalf("UploadBytes returned error: %v", err)
	}
	if string(got) != "raw bytes" {
		t.Fatalf("body = %q", got)
	}
	if result.ETag != "etag-1" || result.Size != 9 {
		t.Fatalf("result = %#v", result)
	}
}

func TestTOSUploadReaderSetsContentLength(t *testing.T) {
	contentLength := int64(-1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentLength = r.ContentLength
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	service := &TOSService{
		httpClient: httpc.New(httpc.WithHTTPClient(server.Client())),
		bucket:     "bucket",
		signer: func(*tos.PreSignedURLInput) (*tos.PreSignedURLOutput, error) {
			return &tos.PreSignedURLOutput{SignedUrl: server.URL}, nil
		},
	}
	if _, err := service.UploadReader(t.Context(), "file.md", strings.NewReader("abc"), 3); err != nil {
		t.Fatalf("UploadReader returned error: %v", err)
	}
	if contentLength != 3 {
		t.Fatalf("content length = %d", contentLength)
	}
}

func TestTOSServiceUploadsFileAndBuildsSigner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	service := &TOSService{
		httpClient: httpc.New(httpc.WithHTTPClient(server.Client())),
		bucket:     "bucket",
		signer: func(input *tos.PreSignedURLInput) (*tos.PreSignedURLOutput, error) {
			if input.Bucket != "bucket" || input.Key == "" || input.Expires != 1800 {
				t.Errorf("unexpected signer input: %#v", input)
			}
			return &tos.PreSignedURLOutput{SignedUrl: server.URL}, nil
		},
	}
	path := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(path, []byte("file"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	result, err := service.Upload(t.Context(), path)
	if err != nil || result.Size != 4 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestNewTOSService(t *testing.T) {
	service, err := newTOSService(httpc.New(), TOSConfig{Endpoint: "https://tos.example", Bucket: "bucket"}, "ak", "sk")
	if err != nil {
		t.Fatalf("newTOSService error: %v", err)
	}
	if service.bucket != "bucket" || service.prefix != "volcenginex/" || service.signer == nil {
		t.Fatalf("service = %#v", service)
	}
}
