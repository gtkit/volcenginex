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
	Poll           PollPolicy
	Logger         Logger
	TOS            *TOSConfig
	ImageMaxBytes  int64
	partialFailure bool
	progressHook   func(*TaskDetail)
}

// PollPolicy controls document task polling.
type PollPolicy struct {
	_ struct{}

	Interval    time.Duration
	MaxInterval time.Duration
	Timeout     time.Duration
	Jitter      float64
}

func defaultPollPolicy() PollPolicy {
	return PollPolicy{
		Interval:    3 * time.Second,
		MaxInterval: 15 * time.Second,
		Timeout:     10 * time.Minute,
		Jitter:      0.2,
	}
}

func (p PollPolicy) withDefaults() PollPolicy {
	d := defaultPollPolicy()
	if p.Interval > 0 {
		d.Interval = p.Interval
	}
	if p.MaxInterval > 0 {
		d.MaxInterval = p.MaxInterval
	}
	if p.Timeout > 0 {
		d.Timeout = p.Timeout
	}
	if p.Jitter >= 0 {
		d.Jitter = p.Jitter
	}
	if d.MaxInterval < d.Interval {
		d.MaxInterval = d.Interval
	}
	return d
}

// TOSConfig controls optional TOS uploads.
type TOSConfig struct {
	_ struct{}

	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	Prefix     string
	PublicRead bool
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

// WithMaxRetries sets the official SDK retry count.
func WithMaxRetries(n int) Option {
	return func(c *Config) error {
		if n < 0 {
			return ErrInvalidArgument
		}
		c.MaxRetries = n
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

// WithPollPolicy sets document task polling defaults.
func WithPollPolicy(p PollPolicy) Option {
	return func(c *Config) error {
		if p.Interval < 0 || p.MaxInterval < 0 || p.Timeout < 0 || p.Jitter < 0 || p.Jitter > 1 {
			return ErrInvalidArgument
		}
		c.Poll = p
		return nil
	}
}

// WithTOS enables TOS uploads.
func WithTOS(cfg TOSConfig) Option {
	return func(c *Config) error {
		if strings.TrimSpace(cfg.Bucket) == "" || strings.TrimSpace(cfg.Endpoint) == "" {
			return ErrInvalidArgument
		}
		c.TOS = &cfg
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

// WithProgressHook registers a document polling progress callback.
func WithProgressHook(hook func(*TaskDetail)) Option {
	return func(c *Config) error {
		if hook == nil {
			return ErrInvalidArgument
		}
		c.progressHook = hook
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
	if c.MaxRetries == 0 {
		c.MaxRetries = defaultMaxRetries
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaultTimeout
	}
	c.Poll = c.Poll.withDefaults()
	if c.ImageMaxBytes == 0 {
		c.ImageMaxBytes = defaultImageMaxSize
	}
	if c.TOS != nil && c.TOS.Prefix == "" {
		c.TOS.Prefix = "volcenginex/"
	}
}
