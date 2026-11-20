package volcenginex

import (
	"context"
	"net/http"

	"github.com/gtkit/httpc"
	"github.com/gtkit/volcenginex/internal/jsoncodec"
	"github.com/gtkit/volcenginex/internal/trace"
	"github.com/volcengine/volcengine-go-sdk/service/i18nopenapi"
	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/client"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/custom"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// Client owns one shared SDK session and all translation sub-services.
// Client and its services are safe for concurrent use after construction.
type Client struct {
	httpClient *httpc.Client
	session    *session.Session
	logger     Logger
	text       *TextService
	image      *ImageService
	document   *DocumentService
	tos        *TOSService
}

// New constructs a client with one shared official SDK session.
func New(cfg Config, opts ...Option) (*Client, error) {
	for _, opt := range opts {
		if opt == nil {
			return nil, ErrInvalidArgument
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	withConfigDefaults(&cfg)
	if cfg.AK == "" || cfg.SK == "" {
		return nil, ErrMissingCredentials
	}

	hcOptions := []httpc.Option{
		httpc.WithTimeout(cfg.RequestTimeout),
		httpc.WithJSONCodec(jsoncodec.Codec{}),
	}
	if cfg.HTTPClient != nil {
		hcOptions = append(hcOptions, httpc.WithHTTPClient(cfg.HTTPClient))
	}
	hc := httpc.New(hcOptions...)

	sdkConfig := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(cfg.AK, cfg.SK, cfg.SessionToken)).
		WithRegion(cfg.Region).
		WithHTTPClient(hc.HTTPClient()).
		WithMaxRetries(cfg.MaxRetries).
		WithLogLevel(volcengine.LogOff).
		WithLogSensitives([]string{"AK", "SK", "SessionToken", cfg.AK, cfg.SK, cfg.SessionToken}).
		WithExtendHttpRequest(func(ctx context.Context, req *http.Request) {
			if id := trace.RequestID(ctx); id != "" {
				req.Header.Set("X-Client-Request-Id", id)
			}
		})
	if cfg.Endpoint != "" {
		sdkConfig.WithEndpoint(cfg.Endpoint)
	}
	sess, err := session.NewSession(sdkConfig)
	if err != nil {
		return nil, clientError("NewSession", err)
	}

	textClient := translate20250301.New(sess)
	documentClient := i18nopenapi.New(sess)
	logger := loggerOrDiscard(cfg.Logger)
	client := &Client{
		httpClient: hc,
		session:    sess,
		logger:     logger,
	}
	client.text = &TextService{
		client:         client,
		api:            textClient,
		batchSize:      defaultBatchSize,
		partialFailure: cfg.partialFailure,
	}
	client.image = &ImageService{client: client, api: textClient, maxBytes: cfg.ImageMaxBytes}
	client.document = &DocumentService{
		client:       client,
		api:          documentClient,
		poll:         cfg.Poll,
		progressHook: cfg.progressHook,
		sleep:        waitDuration,
	}
	if cfg.TOS != nil {
		tosService, err := newTOSService(hc, *cfg.TOS, cfg.AK, cfg.SK)
		if err != nil {
			return nil, err
		}
		client.tos = tosService
	}
	return client, nil
}

// Text returns the synchronous text translation service.
func (c *Client) Text() *TextService { return c.text }

// Image returns the synchronous image translation service.
func (c *Client) Image() *ImageService { return c.image }

// Document returns the asynchronous document translation service.
func (c *Client) Document() *DocumentService { return c.document }

// TOS returns the optional TOS service, or nil when TOS is not configured.
func (c *Client) TOS() *TOSService { return c.tos }

// Download streams a translation result to a local path.
func (c *Client) Download(ctx context.Context, url, destPath string) (int64, error) {
	return download(ctx, c.httpClient, url, destPath)
}

// Close releases client-owned resources.
func (c *Client) Close() error { return nil }

var (
	_ client.ConfigProvider    = (*session.Session)(nil)
	_ custom.ExtendHttpRequest = func(context.Context, *http.Request) {}
)
