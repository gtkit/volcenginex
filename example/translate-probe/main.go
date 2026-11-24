package main

// Run this file with go run from the go-volcengine module directory.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	volcx "github.com/gtkit/volcenginex"
	"github.com/volcengine/volc-sdk-golang/base"
)

const sampleText = "Hello world."

type textItem struct {
	Translation            string
	DetectedSourceLanguage string
}

type wireResponse struct {
	TranslationList  json.RawMessage
	Result           struct{ TranslationList json.RawMessage }
	ResponseMetadata struct {
		RequestID string
		Error     *struct {
			Code    any
			CodeN   json.RawMessage
			Message string
		}
	}
}

type replayTransport struct {
	status int
	body   []byte
}

func (r replayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		if err := req.Body.Close(); err != nil {
			return nil, err
		}
	}
	return &http.Response{
		StatusCode: r.status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(r.body)),
	}, nil
}

func send(ctx context.Context, client *http.Client, credentials base.Credentials, host, version, target string) (int, []byte, error) {
	query := url.Values{"Action": {"TranslateText"}, "Version": {version}}
	endpoint := url.URL{Scheme: "https", Host: host, Path: "/", RawQuery: query.Encode()}
	body, err := json.Marshal(struct {
		TargetLanguage string
		TextList       []string
	}{target, []string{sampleText}})
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if version == "2025-03-01" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := client.Do(credentials.Sign(req))
	if err != nil {
		return 0, nil, err
	}
	const maxBody = 2 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err := errors.Join(err, resp.Body.Close()); err != nil {
		return resp.StatusCode, nil, err
	}
	if len(raw) > maxBody {
		return resp.StatusCode, nil, errors.New("响应体超过 2 MiB")
	}
	return resp.StatusCode, raw, nil
}

func list(raw json.RawMessage) any {
	if raw == nil {
		return "缺失"
	}
	if bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var items []textItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return "无法解码为译文列表"
	}
	return items
}

func replay(ctx context.Context, status int, body []byte, target string) any {
	// Replay only the captured response. The replay transport never opens a connection.
	client, err := volcx.New(volcx.Config{AK: "replay-placeholder-ak", SK: "replay-placeholder-sk", SessionToken: "replay-placeholder-token"},
		volcx.WithHTTPClient(&http.Client{Transport: replayTransport{status, body}}),
		volcx.WithMaxRetries(1),
	)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := client.Text().Translate(ctx, volcx.TextTranslateRequest{
		TargetLanguage: target, TextList: []string{sampleText},
	})
	if provider, ok := errors.AsType[*volcx.Error](err); ok {
		return map[string]any{"code": provider.Code, "httpStatus": provider.HTTPStatus,
			"retryable": provider.Retryable, "requestID": provider.RequestID}
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"items": result.Items, "characterCount": result.CharacterCount}
}

func main() {
	ak, sk := os.Getenv("volcengine_accessKeyId"), os.Getenv("volcengine_secretAccessKey")
	fmt.Printf("---Using credentials: AK=%s, SK=%s\n", ak, sk)

	if strings.TrimSpace(ak) == "" || strings.TrimSpace(sk) == "" {
		fmt.Fprintln(os.Stderr, "请设置 volcengine_accessKeyId 和 volcengine_secretAccessKey")
		os.Exit(2)
	}
	redact := strings.NewReplacer(ak, "[REDACTED]", sk, "[REDACTED]")
	credentials := base.Credentials{AccessKeyID: ak, SecretAccessKey: sk, Region: "cn-north-1", Service: "translate"}
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for _, endpoint := range []struct{ version, host string }{
		{"2020-06-01", "translate.volcengineapi.com"},
		{"2025-03-01", "open.volcengineapi.com"},
	} {
		for _, target := range []string{"zh", "__invalid__"} {
			status, body, err := send(context.Background(), client, credentials, endpoint.host, endpoint.version, target)
			out := map[string]any{"version": endpoint.version, "target": target, "httpStatus": status}
			if err != nil {
				out["error"] = err.Error()
			} else {
				var wire wireResponse
				decoder := json.NewDecoder(bytes.NewReader(body))
				decoder.UseNumber()
				if err := decoder.Decode(&wire); err != nil {
					out["error"] = "响应体无法解码为 JSON"
				} else {
					out["requestID"] = wire.ResponseMetadata.RequestID
					if provider := wire.ResponseMetadata.Error; provider != nil {
						out["Code"], out["Message"] = provider.Code, provider.Message
						out["CodeN"] = any("缺失")
						if provider.CodeN != nil {
							out["CodeN"] = provider.CodeN
						}
					}
					out["TranslationList"] = list(wire.TranslationList)
					out["Result.TranslationList"] = list(wire.Result.TranslationList)
					if endpoint.version == "2025-03-01" {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						out["goLibraryReplay"] = replay(ctx, status, body, target)
						cancel()
					}
				}
			}
			report, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				fmt.Fprintln(os.Stderr, "无法编码诊断结果")
				os.Exit(1)
			}
			fmt.Println(redact.Replace(string(report)))
		}
	}
}
