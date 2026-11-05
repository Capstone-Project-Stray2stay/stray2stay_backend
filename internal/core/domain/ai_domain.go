package domain

import "errors"

// Errors a generative-AI provider can surface. Callers match on these with
// errors.Is so they can decide between retrying later, degrading the feature,
// or reporting a hard failure, without knowing which provider is behind the
// port.
var (
	// ErrAIUnavailable means the provider could not be reached or answered
	// with a server-side failure.
	ErrAIUnavailable = errors.New("ai provider unavailable")
	// ErrAIRateLimited means the quota for the key or model is exhausted.
	ErrAIRateLimited = errors.New("ai provider rate limited")
	// ErrAIBlocked means the provider refused to answer, usually because its
	// safety filters matched the prompt or the generated text.
	ErrAIBlocked = errors.New("ai provider blocked the request")
	// ErrAIEmptyResponse means the call succeeded but carried no usable text.
	ErrAIEmptyResponse = errors.New("ai provider returned no content")
	// ErrAIInvalidRequest means the provider rejected the request as
	// malformed; retrying the same payload cannot help.
	ErrAIInvalidRequest = errors.New("ai provider rejected the request")
)

// AIImage is one image sent inline with a prompt. Data holds the raw bytes,
// not base64 — the adapter encodes them for the wire.
type AIImage struct {
	MIMEType string
	Data     []byte
}

// AIRequest is one provider-agnostic generation call. Only Prompt is
// required; every other field falls back to the client's configured default.
type AIRequest struct {
	// Model overrides the client's default model when non-empty.
	Model string
	// SystemInstruction sets the model's standing role and rules. Keep
	// untrusted user text out of it — it belongs in Prompt.
	SystemInstruction string
	Prompt            string
	Images            []AIImage
	// Temperature of 0 is not the same as "unset": leave nil to keep the
	// model default.
	Temperature     *float32
	MaxOutputTokens int
	// ThinkingEffort asks for more or less reasoning before the model
	// answers. It is stated as intent, not as a provider knob: the control
	// differs by model family and the adapter translates. Empty leaves the
	// model's own default alone.
	ThinkingEffort string
	// ResponseSchema, when non-nil, constrains the answer to JSON matching
	// the schema. It is marshalled as-is, so pass a JSON-shaped value.
	ResponseSchema any
}

// How hard a model should think before answering. Extraction-style calls
// want Low: it keeps latency down and stops reasoning tokens from eating the
// output allowance.
const (
	AIThinkingLow  = "low"
	AIThinkingHigh = "high"
)

// AIUsage reports what one call cost, for logging and quota tracking.
type AIUsage struct {
	PromptTokens int
	OutputTokens int
	TotalTokens  int
}

// AIResponse is the text a provider generated plus the metadata worth keeping.
type AIResponse struct {
	Text  string
	Model string
	// FinishReason is the provider's own stop reason, e.g. "STOP" or
	// "MAX_TOKENS", kept verbatim for diagnostics.
	FinishReason string
	Usage        AIUsage
}

// Lifecycle of the AI summary attached to an adoption request. The summary is
// written after the request itself is committed, so the owner's view needs to
// tell "not generated yet" apart from "generation failed".
const (
	AdoptionSummaryPending = "PENDING"
	AdoptionSummaryReady   = "READY"
	AdoptionSummaryFailed  = "FAILED"
)

// AdoptionSummaryInput is everything the summarizer needs to describe one
// adoption request to a pet owner: which pet was requested, and the full
// screening form the adopter submitted.
type AdoptionSummaryInput struct {
	PetName  string
	PetType  string
	PetBreed string
	Answer   ScreeningAnswer
}
