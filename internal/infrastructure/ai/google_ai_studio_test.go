package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

// newTestClient points a client at a stub server and strips the backoff so
// retry paths do not slow the suite down.
func newTestClient(t *testing.T, handler http.HandlerFunc, maxRetries int) *GoogleAIStudioClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewGoogleAIStudioClient(config.GoogleAIConfig{
		APIKey:     "test-key",
		Model:      "gemini-2.5-flash",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: maxRetries,
	})
	if err != nil {
		t.Fatalf("NewGoogleAIStudioClient() error = %v", err)
	}
	return client
}

func okResponse(text string) string {
	return `{"candidates":[{"content":{"parts":[{"text":"` + text + `"}],"role":"model"},"finishReason":"STOP"}],` +
		`"usageMetadata":{"promptTokenCount":11,"candidatesTokenCount":7,"totalTokenCount":18},"modelVersion":"gemini-2.5-flash-001"}`
}

func TestNewGoogleAIStudioClientRequiresAPIKey(t *testing.T) { // UT-BE-AI-01
	if _, err := NewGoogleAIStudioClient(config.GoogleAIConfig{Model: "m", BaseURL: "https://example.test"}); err == nil {
		t.Fatal("NewGoogleAIStudioClient() succeeded without an API key; want an error")
	}
}

func TestGenerateSendsKeyInHeaderAndParsesResponse(t *testing.T) { // UT-BE-AI-02
	var gotPath, gotKey, gotQuery string
	var body generateContentRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotKey = r.Header.Get("x-goog-api-key")

		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not valid JSON: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, okResponse("A calm, experienced adopter."))
	}, 0)

	temperature := float32(0.2)
	resp, err := client.Generate(context.Background(), domain.AIRequest{
		SystemInstruction: "be brief",
		Prompt:            "summarize this",
		Temperature:       &temperature,
		MaxOutputTokens:   600,
		ThinkingEffort:    domain.AIThinkingLow,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if gotPath != "/models/gemini-2.5-flash:generateContent" {
		t.Errorf("request path = %q; want the generateContent path for the configured model", gotPath)
	}
	if gotKey != "test-key" {
		t.Errorf("x-goog-api-key = %q; want the configured key", gotKey)
	}
	if gotQuery != "" {
		t.Errorf("query string = %q; the key must never travel in the URL", gotQuery)
	}
	if body.SystemInstruction == nil || body.SystemInstruction.Parts[0].Text != "be brief" {
		t.Errorf("systemInstruction = %+v; want the configured instruction", body.SystemInstruction)
	}
	if body.GenerationConfig == nil || body.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("generationConfig = %+v; want a thinkingConfig", body.GenerationConfig)
	}
	// The stub client is configured with gemini-2.5-flash, which takes a
	// budget rather than a level.
	budget := body.GenerationConfig.ThinkingConfig.ThinkingBudget
	if budget == nil || *budget != 0 {
		t.Errorf("thinkingBudget = %v; want an explicit 0, not an omitted field", budget)
	}

	if resp.Text != "A calm, experienced adopter." {
		t.Errorf("Text = %q; want the candidate text", resp.Text)
	}
	if resp.Model != "gemini-2.5-flash-001" {
		t.Errorf("Model = %q; want the resolved model version from the response", resp.Model)
	}
	if resp.Usage.TotalTokens != 18 {
		t.Errorf("Usage.TotalTokens = %d; want 18", resp.Usage.TotalTokens)
	}
}

func TestGenerateEncodesImagesInline(t *testing.T) { // UT-BE-AI-02
	var body generateContentRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		io.WriteString(w, okResponse("ok"))
	}, 0)

	if _, err := client.Generate(context.Background(), domain.AIRequest{
		Prompt: "what breed is this?",
		Images: []domain.AIImage{{MIMEType: "image/png", Data: []byte{0x01, 0x02, 0x03}}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	parts := body.Contents[0].Parts
	if len(parts) != 2 {
		t.Fatalf("got %d parts; want the prompt plus one image", len(parts))
	}
	if parts[1].InlineData == nil {
		t.Fatal("second part has no inlineData; want the image")
	}
	if parts[1].InlineData.MIMEType != "image/png" {
		t.Errorf("mimeType = %q; want image/png", parts[1].InlineData.MIMEType)
	}
	want := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
	if parts[1].InlineData.Data != want {
		t.Errorf("inline data = %q; want base64 of the raw bytes (%q)", parts[1].InlineData.Data, want)
	}
}

func TestGenerateRetriesServerErrorsThenSucceeds(t *testing.T) { // UT-BE-AI-05
	var calls int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, `{"error":{"code":503,"message":"overloaded","status":"UNAVAILABLE"}}`)
			return
		}
		io.WriteString(w, okResponse("second attempt"))
	}, 2)
	client.httpClient.Timeout = 5 * time.Second

	resp, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Generate() error = %v; want the retry to succeed", err)
	}
	if resp.Text != "second attempt" {
		t.Errorf("Text = %q; want the retried response", resp.Text)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server calls = %d; want 2", got)
	}
}

func TestGenerateDoesNotRetryBadRequests(t *testing.T) { // UT-BE-AI-05
	var calls int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"code":400,"message":"bad schema","status":"INVALID_ARGUMENT"}}`)
	}, 3)

	_, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "hi"})
	if !errors.Is(err, domain.ErrAIInvalidRequest) {
		t.Fatalf("Generate() error = %v; want ErrAIInvalidRequest", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d; a malformed request must not be retried", got)
	}
}

func TestGenerateMapsProviderFailures(t *testing.T) { // UT-BE-AI-05
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{
			name:    "quota exhausted",
			status:  http.StatusTooManyRequests,
			body:    `{"error":{"code":429,"message":"quota","status":"RESOURCE_EXHAUSTED"}}`,
			wantErr: domain.ErrAIRateLimited,
		},
		{
			name:    "key rejected",
			status:  http.StatusForbidden,
			body:    `{"error":{"code":403,"message":"forbidden","status":"PERMISSION_DENIED"}}`,
			wantErr: domain.ErrAIInvalidRequest,
		},
		{
			name:    "prompt blocked by safety filters",
			status:  http.StatusOK,
			body:    `{"promptFeedback":{"blockReason":"SAFETY"}}`,
			wantErr: domain.ErrAIBlocked,
		},
		{
			name:    "answer blocked by safety filters",
			status:  http.StatusOK,
			body:    `{"candidates":[{"content":{"parts":[]},"finishReason":"SAFETY"}]}`,
			wantErr: domain.ErrAIBlocked,
		},
		{
			name:    "no candidates",
			status:  http.StatusOK,
			body:    `{"candidates":[]}`,
			wantErr: domain.ErrAIEmptyResponse,
		},
		{
			name:    "budget spent before any text",
			status:  http.StatusOK,
			body:    `{"candidates":[{"content":{"parts":[]},"finishReason":"MAX_TOKENS"}]}`,
			wantErr: domain.ErrAIEmptyResponse,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				io.WriteString(w, test.body)
			}, 0)

			_, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "hi"})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Generate() error = %v; want %v", err, test.wantErr)
			}
			if strings.Contains(err.Error(), "test-key") {
				t.Errorf("error %q leaks the API key", err)
			}
		})
	}
}

func TestGenerateHonoursContextCancellation(t *testing.T) { // UT-BE-AI-07
	// The handler answers on its own after the caller's deadline has already
	// passed, so the server can always shut down cleanly at the end of the
	// test while the client still has to give up mid-flight.
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(250 * time.Millisecond)
		io.WriteString(w, okResponse("too late"))
	}, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Generate(ctx, domain.AIRequest{Prompt: "hi"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Generate() error = %v; want the caller's deadline to surface", err)
	}
}

func TestGenerateRejectsEmptyPrompt(t *testing.T) { // UT-BE-AI-01
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server was called; an empty prompt must not reach the provider")
	}, 0)

	if _, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "   "}); !errors.Is(err, domain.ErrAIInvalidRequest) {
		t.Fatalf("Generate() error = %v; want ErrAIInvalidRequest", err)
	}
}

func TestBuildRequestOmitsGenerationConfigWhenUnset(t *testing.T) { // UT-BE-AI-03
	got := buildRequest(domain.AIRequest{Prompt: "hi"}, "gemini-2.5-flash")
	if got.GenerationConfig != nil {
		t.Errorf("generationConfig = %+v; want it omitted so the model defaults apply", got.GenerationConfig)
	}
}

func TestBuildRequestSetsJSONModeWithSchema(t *testing.T) { // UT-BE-AI-03
	schema := map[string]any{"type": "object"}
	got := buildRequest(domain.AIRequest{Prompt: "hi", ResponseSchema: schema}, "gemini-2.5-flash")

	if got.GenerationConfig == nil {
		t.Fatal("generationConfig is nil; want JSON mode configured")
	}
	if got.GenerationConfig.ResponseMIMEType != "application/json" {
		t.Errorf("responseMimeType = %q; want application/json", got.GenerationConfig.ResponseMIMEType)
	}
}

func TestThinkingControlMatchesModelFamily(t *testing.T) { // UT-BE-AI-04
	// Sending a 2.5-style token budget to a 3.x model does not fail loudly:
	// the request hangs until the client's timeout. These cases are the only
	// thing standing between that and production.
	tests := []struct {
		name      string
		model     string
		effort    string
		wantLevel string
		wantZero  bool
		wantNil   bool
	}{
		{name: "gemini 3.x takes a named level", model: "gemini-3.8-flash", effort: domain.AIThinkingLow, wantLevel: "low"},
		{name: "gemini 3.x high", model: "gemini-3.1-flash-lite", effort: domain.AIThinkingHigh, wantLevel: "high"},
		{name: "gemini 2.5 takes a zero budget", model: "gemini-2.5-flash", effort: domain.AIThinkingLow, wantZero: true},
		{name: "gemini 2.5 high leaves the default", model: "gemini-2.5-pro", effort: domain.AIThinkingHigh, wantNil: true},
		{name: "unversioned alias counts as current", model: "gemini-flash-latest", effort: domain.AIThinkingLow, wantLevel: "low"},
		{name: "no effort means no thinking config", model: "gemini-3.8-flash", effort: "", wantNil: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := thinkingConfigFor(test.model, test.effort)

			if test.wantNil {
				if got != nil {
					t.Fatalf("thinkingConfigFor(%q, %q) = %+v; want nil", test.model, test.effort, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("thinkingConfigFor(%q, %q) = nil; want a config", test.model, test.effort)
			}
			if got.ThinkingLevel != test.wantLevel {
				t.Errorf("thinkingLevel = %q; want %q", got.ThinkingLevel, test.wantLevel)
			}
			if test.wantZero {
				if got.ThinkingBudget == nil || *got.ThinkingBudget != 0 {
					t.Errorf("thinkingBudget = %v; want an explicit 0", got.ThinkingBudget)
				}
			} else if got.ThinkingBudget != nil {
				t.Errorf("thinkingBudget = %v; a level-based model must not be sent a budget", *got.ThinkingBudget)
			}
		})
	}
}

func TestGeminiMajorVersion(t *testing.T) { // UT-BE-AI-04
	tests := map[string]int{
		"gemini-3.8-flash":       3,
		"gemini-2.5-flash-lite":  2,
		"gemini-3-flash-preview": 3,
		"gemini-4-argon":         4,
		"gemini-flash-latest":    0,
	}
	for model, want := range tests {
		if got := geminiMajorVersion(model); got != want {
			t.Errorf("geminiMajorVersion(%q) = %d; want %d", model, got, want)
		}
	}
}

func TestGenerateStopsWhenQuotaResetIsFarAway(t *testing.T) { // UT-BE-AI-06
	// A daily quota error comes back asking for most of a day. Retrying it
	// is pure waste: every attempt is a guaranteed rejection, and the caller
	// sits there for minutes to learn what the first response already said.
	var calls int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":{"code":429,"message":"You exceeded your current quota","status":"RESOURCE_EXHAUSTED",`+
			`"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"84404s"}]}}`)
	}, 10)

	_, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "hi"})
	if !errors.Is(err, domain.ErrAIRateLimited) {
		t.Fatalf("Generate() error = %v; want ErrAIRateLimited", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d; a day-long quota reset must not be retried", got)
	}
	if !strings.Contains(err.Error(), "retry in") {
		t.Errorf("error %q should say how long the provider asked for", err)
	}
}

func TestGenerateHonoursShortRetryAfter(t *testing.T) { // UT-BE-AI-06
	// A brief delay is the case retrying exists for, so it must still run.
	var calls int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"error":{"code":429,"message":"slow down","status":"RESOURCE_EXHAUSTED"}}`)
			return
		}
		io.WriteString(w, okResponse("after the wait"))
	}, 2)

	resp, err := client.Generate(context.Background(), domain.AIRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Generate() error = %v; want the retry to succeed", err)
	}
	if resp.Text != "after the wait" {
		t.Errorf("Text = %q; want the retried response", resp.Text)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server calls = %d; want 2", got)
	}
}

func TestRetryDelayFrom(t *testing.T) { // UT-BE-AI-06
	tests := []struct {
		name   string
		header string
		body   string
		want   time.Duration
	}{
		{name: "header wins", header: "5", body: `{}`, want: 5 * time.Second},
		{
			name: "falls back to RetryInfo in the body",
			body: `{"error":{"details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"84404s"}]}}`,
			want: 84404 * time.Second,
		},
		{name: "nothing advised", body: `{"error":{"message":"nope"}}`, want: 0},
		{name: "unparseable body", body: `not json`, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryDelayFrom(test.header, []byte(test.body)); got != test.want {
				t.Errorf("retryDelayFrom(%q, %q) = %s; want %s", test.header, test.body, got, test.want)
			}
		})
	}
}
