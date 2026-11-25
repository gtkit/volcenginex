package volcenginex

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gtkit/httpc"
)

type testLogger struct{}

func (testLogger) Info(_ context.Context, _ string, _ ...any)  {}
func (testLogger) Error(_ context.Context, _ string, _ ...any) {}

func TestNewBuildsSharedServices(t *testing.T) {
	client, err := New(Config{AK: "ak", SK: "sk"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if client.Text() == nil || client.Image() == nil || client.Audio() == nil {
		t.Fatalf("unexpected services: text=%p image=%p audio=%p", client.Text(), client.Image(), client.Audio())
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestNewRejectsMissingCredentials(t *testing.T) {
	_, err := New(Config{})
	if !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("error = %v, want ErrMissingCredentials", err)
	}
}

func TestOptionsValidateAndApply(t *testing.T) {
	options := []Option{
		WithAKSK("ak", "sk"),
		WithRegion("cn-beijing"),
		WithEndpoint("http://localhost"),
		WithHTTPClient(&http.Client{}),
		WithHTTPC(httpc.New()),
		WithMaxRetries(2),
		WithTimeout(time.Second),
		WithLogger(testLogger{}),
		WithPartialFailure(),
		WithMaxImageBytes(1024),
	}
	cfg := Config{}
	for _, option := range options {
		if err := option(&cfg); err != nil {
			t.Fatalf("option returned error: %v", err)
		}
	}
	if !cfg.partialFailure || cfg.ImageMaxBytes != 1024 {
		t.Fatalf("options were not applied: %#v", cfg)
	}
	invalid := []Option{
		WithRegion(""), WithHTTPClient(nil), WithHTTPC(nil), WithMaxRetries(-1), WithTimeout(0),
		WithMaxImageBytes(0),
	}
	for _, option := range invalid {
		if err := option(&Config{}); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("invalid option error = %v", err)
		}
	}
}
