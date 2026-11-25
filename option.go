package volcenginex

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gtkit/httpc"
)

const (
	defaultRegion       = "cn-north-1"
	defaultMaxRetries   = 3
	defaultTimeout      = 30 * time.Second
	defaultBatchSize    = 16
	defaultImageMaxSize = 5 << 20
)

// Config controls client construction and service defaults.
type Config struct {
	_ struct{}

	AK             string
	SK             string
	SessionToken   string
	Region         string
	Endpoint       string
	HTTPClient     *http.Client
	MaxRetries     int
	RequestTimeout time.Duration
	Logger         Logger
	ImageMaxBytes  int64
	maxRetriesSet  bool
	partialFailure bool
}

// Option changes client construction behavior.
type Option func(*Config) error

// WithAKSK sets the access key and secret key.
func WithAKSK(ak, sk string) Option {
	return func(c *Config) error {
		c.AK, c.SK = strings.TrimSpace(ak), strings.TrimSpace(sk)
		return nil
	}
}

// WithRegion sets the Volcengine region.
func WithRegion(region string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(region) == "" {
			return ErrInvalidArgument
		}
		c.Region = strings.TrimSpace(region)
		return nil
	}
}

// WithEndpoint overrides the official SDK endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Config) error {
		c.Endpoint = strings.TrimSpace(endpoint)
		return nil
	}
}

// WithHTTPClient sets the underlying HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Config) error {
		if hc == nil {
			return ErrInvalidArgument
		}
		c.HTTPClient = hc
		return nil
	}
}

// WithHTTPC reuses an existing httpc client and its HTTP transport.
func WithHTTPC(hc *httpc.Client) Option {
	return func(c *Config) error {
		if hc == nil {
			return ErrInvalidArgument
		}
		c.HTTPClient = hc.HTTPClient()
		return nil
	}
}

// WithMaxRetries sets the official SDK retry count. Zero disables retries.
// An unset retry count defaults to three.
func WithMaxRetries(n int) Option {
	return func(c *Config) error {
		if n < 0 {
			return ErrInvalidArgument
		}
		c.MaxRetries = n
		c.maxRetriesSet = true
		return nil
	}
}

// WithTimeout sets the single-call HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) error {
		if d <= 0 {
			return ErrInvalidArgument
		}
		c.RequestTimeout = d
		return nil
	}
}

// WithLogger sets the optional structured logger adapter.
func WithLogger(l Logger) Option {
	return func(c *Config) error {
		c.Logger = l
		return nil
	}
}

// WithPartialFailure makes text translation return successful items and batch errors.
func WithPartialFailure() Option {
	return func(c *Config) error {
		c.partialFailure = true
		return nil
	}
}

// WithMaxImageBytes changes the image byte limit.
func WithMaxImageBytes(n int64) Option {
	return func(c *Config) error {
		if n <= 0 {
			return ErrInvalidArgument
		}
		c.ImageMaxBytes = n
		return nil
	}
}

func withConfigDefaults(c *Config) {
	if c.AK == "" {
		c.AK = os.Getenv("VOLCENGINE_ACCESS_KEY")
	}
	if c.SK == "" {
		c.SK = os.Getenv("VOLCENGINE_SECRET_KEY")
	}
	if c.SessionToken == "" {
		c.SessionToken = os.Getenv("VOLCENGINE_SESSION_TOKEN")
	}
	if c.Region == "" {
		c.Region = defaultRegion
	}
	if c.MaxRetries == 0 && !c.maxRetriesSet {
		c.MaxRetries = defaultMaxRetries
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaultTimeout
	}
	if c.ImageMaxBytes == 0 {
		c.ImageMaxBytes = defaultImageMaxSize
	}
}
