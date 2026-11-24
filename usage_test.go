package volcenginex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/translate20250301"
)

func TestCountVolcMTChar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "billing characters", text: "中A 1!?", want: 6},
		{name: "multibyte rune", text: "🙂", want: 1},
		{name: "newlines and tags", text: "<b>\n你好</b>", want: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CountVolcMTChar(tt.text); got != tt.want {
				t.Fatalf("CountVolcMTChar(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestTextUsageHTTP(t *testing.T) {
	t.Parallel()
	beforeMidnight := time.Date(2026, 11, 23, 15, 59, 59, 0, time.UTC)
	afterMidnight := beforeMidnight.Add(time.Second)
	for _, tc := range []struct {
		name     string
		recent   bool
		from, to time.Time
		dates    [2]int32
		body     string
		want     []UsagePoint
	}{
		{
			name: "explicit range across Beijing midnight",
			from: beforeMidnight, to: afterMidnight,
			dates: [2]int32{20261123, 20261124},
			body:  `{"Result":{"Points":[{"Timestamp":20261123,"Value":20}]},"ResponseMetadata":{"RequestId":"usage-1"}}`,
			want:  []UsagePoint{{Timestamp: 20261123, Value: 20}},
		},
		{
			name: "same day range and empty usage",
			from: afterMidnight, to: afterMidnight.Add(time.Hour),
			dates: [2]int32{20261124, 20261124},
			body:  `{"Result":{"Points":[]},"ResponseMetadata":{"RequestId":"usage-2"}}`,
		},
		{
			name:   "recent thirty calendar days",
			recent: true,
			body:   `{"Result":{"Points":[{"Timestamp":20261124,"Value":30}]},"ResponseMetadata":{"RequestId":"usage-3"}}`,
			want:   []UsagePoint{{Timestamp: 20261124, Value: 30}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got struct {
				Service  string
				From, To int32
			}
			calls := 0
			client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				query := req.URL.Query()
				if req.Method != http.MethodPost || query.Get("Action") != "GetUsage" || query.Get("Version") != "2025-03-01" {
					t.Errorf("unexpected usage request %s %s", req.Method, req.URL)
				}
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					return nil, errors.Join(err, req.Body.Close())
				}
				return providerTestResponse(req, http.StatusOK, tc.body)
			}, WithMaxRetries(0))
			start := time.Now().In(usageLocation)
			var usage *Usage
			var err error
			if tc.recent {
				usage, err = client.Text().Usage(t.Context())
			} else {
				usage, err = client.Text().GetUsageCommon(t.Context(), int32(tc.from.Unix()), int32(tc.to.Unix()))
			}
			if err != nil || usage == nil || !slices.Equal(usage.Points, tc.want) {
				t.Fatalf("usage = %#v, err = %v, want points %v", usage, err, tc.want)
			}
			dates := [2]int32{got.From, got.To}
			wantDates := tc.dates
			if tc.recent {
				end := time.Now().In(usageLocation)
				wantDates = [2]int32{usageDate(start.AddDate(0, 0, -29)), usageDate(start)}
				if dates != wantDates {
					wantDates = [2]int32{usageDate(end.AddDate(0, 0, -29)), usageDate(end)}
				}
			}
			if calls != 1 || got.Service != "text" || dates != wantDates {
				t.Fatalf("calls = %d, service = %q, dates = %v, want one text request for %v", calls, got.Service, dates, wantDates)
			}
		})
	}
}

func TestGetUsageCommonRetainsNumericProviderError(t *testing.T) {
	t.Parallel()
	client := providerTestClient(t, func(req *http.Request) (*http.Response, error) {
		return providerTestResponse(req, http.StatusOK, `{"ResponseMetadata":{"RequestId":"usage-rejected","Error":{"CodeN":-400,"Message":"invalid range"}}}`)
	}, WithMaxRetries(0))
	usage, err := client.Text().GetUsageCommon(t.Context(), 1, 2)
	got, ok := errors.AsType[*Error](err)
	if !ok || usage != nil || got.Op != "GetUsageCommon" || got.RequestID != "usage-rejected" {
		t.Fatalf("usage = %#v, err = %v", usage, err)
	}
	if code, present := got.CodeN(); !present || code != -400 {
		t.Fatalf("CodeN = (%d, %t)", code, present)
	}
}

func TestGetUsageCommonValidationAndProviderError(t *testing.T) {
	called := false
	providerErr := errors.New("provider failure")
	fake := &fakeTextAPI{usage: func(context.Context, *translate20250301.GetUsageInput) (*translate20250301.GetUsageOutput, error) {
		called = true
		return nil, providerErr
	}}
	service := &TextService{api: fake}
	if _, err := service.GetUsageCommon(t.Context(), 2, 1); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("range error = %v", err)
	}
	if called {
		t.Fatal("invalid range invoked provider")
	}
	//nolint:staticcheck // A nil context exercises the public argument validation.
	if _, err := service.GetUsageCommon(nil, 1, 2); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil context error = %v", err)
	}
	if called {
		t.Fatal("nil context invoked provider")
	}
	if _, err := service.GetUsageCommon(t.Context(), 1, 2); err == nil || !errors.Is(err, providerErr) {
		t.Fatalf("provider error = %v", err)
	}
}
