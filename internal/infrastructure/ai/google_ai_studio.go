// Package ai holds the outbound adapters for generative-AI providers. The
// only one implemented today is Google AI Studio (the Gemini REST API), which
// satisfies port.AIGenerator.
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

const (
	// Responses larger than this are almost certainly a misconfigured
	// endpoint rather than a real answer, so cap what we read into memory.
	maxResponseBytes = 4 << 20 // 4MB

	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 8 * time.Second

	// maxServerAdvisedWait is the longest provider-requested delay we are
	// willing to sit through. Beyond it, retrying is not patience but waste:
	// an exhausted daily quota comes back asking for the better part of a
	// day, and every further attempt burns time for a guaranteed rejection.
	maxServerAdvisedWait = 60 * time.Second
)

// GoogleAIStudioClient calls the Gemini generateContent endpoint over plain
// HTTP. It is safe for concurrent use.
type GoogleAIStudioClient struct {
	apiKey     string
	model      string
	baseURL    string
	maxRetries int
	httpClient *http.Client
}

// NewGoogleAIStudioClient builds a client from config. It fails fast when no
// API key is configured; callers that want the server to boot without AI
// should check cfg.Enabled() first.
func NewGoogleAIStudioClient(cfg config.GoogleAIConfig) (*GoogleAIStudioClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("google ai studio: GEMINI_API_KEY is not set")
	}
	if cfg.Model == "" {
		return nil, errors.New("google ai studio: model is not set")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("google ai studio: base url is not set")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	retries := cfg.MaxRetries
	if retries < 0 {
		retries = 0
	}

	return &GoogleAIStudioClient{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		maxRetries: retries,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// MaxCallDuration reports the worst case for one Generate: every attempt
// running to its full timeout, plus every backoff between them. It is derived
// from the same schedule Generate uses, so raising GEMINI_MAX_RETRIES or
// GEMINI_TIMEOUT_SECONDS cannot leave a caller's deadline stale.
func (c *GoogleAIStudioClient) MaxCallDuration() time.Duration {
	total := c.httpClient.Timeout * time.Duration(c.maxRetries+1)
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		total += backoffFor(attempt, nil)
	}
	return total
}

// Generate sends one prompt and returns the model's text.
func (c *GoogleAIStudioClient) Generate(ctx context.Context, req domain.AIRequest) (domain.AIResponse, error) {
	if strings.TrimSpace(req.Prompt) == "" && len(req.Images) == 0 {
		return domain.AIResponse{}, fmt.Errorf("%w: prompt is empty", domain.ErrAIInvalidRequest)
	}

	model := req.Model
	if model == "" {
		model = c.model
	}

	payload, err := json.Marshal(buildRequest(req, model))
	if err != nil {
		return domain.AIResponse{}, fmt.Errorf("google ai studio: encode request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, model)

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := wait(ctx, backoffFor(attempt, lastErr)); err != nil {
				return domain.AIResponse{}, err
			}
		}

		resp, retryAfter, err := c.do(ctx, url, payload)
		if err == nil {
			return c.decode(resp, model)
		}

		lastErr = err
		if !retryable(err) {
			return domain.AIResponse{}, err
		}
		// The provider knows better than our schedule does. If it asks for
		// longer than we are prepared to wait, stop now and say why.
		if retryAfter > maxServerAdvisedWait {
			return domain.AIResponse{}, fmt.Errorf("%w (provider asked to retry in %s, so no further attempts were made)", err, retryAfter.Round(time.Second))
		}
		if retryAfter > 0 {
			lastErr = retryHint{err: err, after: retryAfter}
		}
		// The caller's deadline outranks our retry budget.
		if ctx.Err() != nil {
			return domain.AIResponse{}, ctx.Err()
		}
	}

	return domain.AIResponse{}, lastErr
}

// do performs one attempt and returns the raw response body on success.
func (c *GoogleAIStudioClient) do(ctx context.Context, url string, payload []byte) (body []byte, retryAfter time.Duration, err error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("google ai studio: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// The key travels in a header, never the query string, so it stays out
	// of proxy and server access logs.
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, 0, ctxErr
		}
		return nil, 0, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: read response: %v", domain.ErrAIUnavailable, err)
	}

	if resp.StatusCode == http.StatusOK {
		return raw, 0, nil
	}
	return nil, retryDelayFrom(resp.Header.Get("Retry-After"), raw), statusError(resp.StatusCode, raw)
}

// decode turns a 200 body into the domain response, mapping the ways Gemini
// can answer successfully while still giving us nothing usable.
func (c *GoogleAIStudioClient) decode(raw []byte, model string) (domain.AIResponse, error) {
	var parsed generateContentResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.AIResponse{}, fmt.Errorf("%w: decode response: %v", domain.ErrAIUnavailable, err)
	}

	if parsed.PromptFeedback != nil && parsed.PromptFeedback.BlockReason != "" {
		return domain.AIResponse{}, fmt.Errorf("%w: prompt blocked (%s)", domain.ErrAIBlocked, parsed.PromptFeedback.BlockReason)
	}
	if len(parsed.Candidates) == 0 {
		return domain.AIResponse{}, domain.ErrAIEmptyResponse
	}

	candidate := parsed.Candidates[0]
	switch candidate.FinishReason {
	case "SAFETY", "PROHIBITED_CONTENT", "BLOCKLIST", "SPII":
		return domain.AIResponse{}, fmt.Errorf("%w: response blocked (%s)", domain.ErrAIBlocked, candidate.FinishReason)
	}

	var text strings.Builder
	for _, p := range candidate.Content.Parts {
		text.WriteString(p.Text)
	}
	answer := strings.TrimSpace(text.String())
	if answer == "" {
		// MAX_TOKENS with no text means the budget went entirely to
		// reasoning; say so, because the fix is a config change.
		if candidate.FinishReason == "MAX_TOKENS" {
			return domain.AIResponse{}, fmt.Errorf("%w: output token budget exhausted before any text was produced", domain.ErrAIEmptyResponse)
		}
		return domain.AIResponse{}, domain.ErrAIEmptyResponse
	}

	resolvedModel := parsed.ModelVersion
	if resolvedModel == "" {
		resolvedModel = model
	}

	return domain.AIResponse{
		Text:         answer,
		Model:        resolvedModel,
		FinishReason: candidate.FinishReason,
		Usage: domain.AIUsage{
			PromptTokens: parsed.UsageMetadata.PromptTokenCount,
			OutputTokens: parsed.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  parsed.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func buildRequest(req domain.AIRequest, model string) generateContentRequest {
	parts := make([]part, 0, 1+len(req.Images))
	if req.Prompt != "" {
		parts = append(parts, part{Text: req.Prompt})
	}
	for _, img := range req.Images {
		mimeType := img.MIMEType
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		parts = append(parts, part{InlineData: &inlineData{
			MIMEType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(img.Data),
		}})
	}

	out := generateContentRequest{
		Contents: []content{{Role: "user", Parts: parts}},
	}
	if req.SystemInstruction != "" {
		out.SystemInstruction = &content{Parts: []part{{Text: req.SystemInstruction}}}
	}

	cfg := generationConfig{
		Temperature:     req.Temperature,
		MaxOutputTokens: req.MaxOutputTokens,
	}
	if req.ResponseSchema != nil {
		cfg.ResponseMIMEType = "application/json"
		cfg.ResponseSchema = req.ResponseSchema
	}
	cfg.ThinkingConfig = thinkingConfigFor(model, req.ThinkingEffort)
	// Comparing the struct directly would panic once ResponseSchema holds a
	// map, so test the fields that can set it.
	if cfg.Temperature != nil || cfg.MaxOutputTokens > 0 || cfg.ResponseSchema != nil || cfg.ThinkingConfig != nil {
		out.GenerationConfig = &cfg
	}

	return out
}

// thinkingConfigFor translates requested effort into the control the model's
// family actually accepts. Getting this wrong is not a graceful degradation:
// sending a 2.5-style thinkingBudget to a 3.x model makes the request hang
// until the client times out, with no error to explain why.
func thinkingConfigFor(model string, effort string) *thinkingConfig {
	if effort == "" {
		return nil
	}

	if usesThinkingLevel(model) {
		return &thinkingConfig{ThinkingLevel: effort}
	}

	// The 2.5 family has no named levels: a zero budget is how you ask it
	// not to think, and anything else is left to the model's own default.
	if effort == domain.AIThinkingLow {
		noThinking := 0
		return &thinkingConfig{ThinkingBudget: &noThinking}
	}
	return nil
}

// usesThinkingLevel reports whether a model takes the named thinkingLevel
// control rather than a token budget. Unversioned aliases such as
// "gemini-flash-latest" track current models, so they are treated as new.
func usesThinkingLevel(model string) bool {
	major := geminiMajorVersion(model)
	return major == 0 || major >= 3
}

// geminiMajorVersion pulls the family number out of a model id:
// "gemini-3.8-flash" gives 3, "gemini-2.5-flash" gives 2, and a name with no
// version at all gives 0.
func geminiMajorVersion(model string) int {
	for _, segment := range strings.Split(model, "-") {
		digits := segment
		if dot := strings.Index(digits, "."); dot != -1 {
			digits = digits[:dot]
		}
		if major, err := strconv.Atoi(digits); err == nil {
			return major
		}
	}
	return 0
}

// statusError maps an HTTP failure onto a domain error, carrying the
// provider's own message but never the API key.
func statusError(status int, raw []byte) error {
	message := strings.TrimSpace(string(raw))
	var parsed apiErrorEnvelope
	if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Error.Message != "" {
		message = parsed.Error.Message
		if parsed.Error.Status != "" {
			message = parsed.Error.Status + ": " + message
		}
	}
	if message == "" {
		message = http.StatusText(status)
	}

	switch {
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", domain.ErrAIRateLimited, message)
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Errorf("%w: credentials rejected (%s)", domain.ErrAIInvalidRequest, message)
	case status >= 500:
		return fmt.Errorf("%w: http %d: %s", domain.ErrAIUnavailable, status, message)
	default:
		return fmt.Errorf("%w: http %d: %s", domain.ErrAIInvalidRequest, status, message)
	}
}

// retryHint carries a provider-supplied Retry-After alongside the error it
// came with, so the backoff can honour it without changing the error's
// identity for errors.Is.
type retryHint struct {
	err   error
	after time.Duration
}

func (r retryHint) Error() string { return r.err.Error() }
func (r retryHint) Unwrap() error { return r.err }

// retryable reports whether another attempt could plausibly succeed. Bad
// requests and rejected credentials never become valid on their own.
func retryable(err error) bool {
	return errors.Is(err, domain.ErrAIUnavailable) || errors.Is(err, domain.ErrAIRateLimited)
}

func backoffFor(attempt int, lastErr error) time.Duration {
	var hint retryHint
	if errors.As(lastErr, &hint) && hint.after > 0 {
		if hint.after > maxBackoff {
			return maxBackoff
		}
		return hint.after
	}

	delay := initialBackoff << (attempt - 1)
	if delay > maxBackoff || delay <= 0 {
		return maxBackoff
	}
	return delay
}

// retryDelayFrom reads how long the provider wants us to wait, preferring
// the Retry-After header and falling back to the RetryInfo detail that quota
// errors carry in the body.
func retryDelayFrom(header string, raw []byte) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	var parsed apiErrorEnvelope
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0
	}
	for _, detail := range parsed.Error.Details {
		if detail.RetryDelay == "" {
			continue
		}
		if delay, err := time.ParseDuration(detail.RetryDelay); err == nil && delay > 0 {
			return delay
		}
	}
	return 0
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
