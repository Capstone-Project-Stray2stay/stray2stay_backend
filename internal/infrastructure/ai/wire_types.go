package ai

// Wire types for the Gemini generateContent REST API. They are kept separate
// from the domain types so the provider's shape never leaks into the core.
// See https://ai.google.dev/api/generate-content.

type generateContentRequest struct {
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	Contents          []content         `json:"contents"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inlineData,omitempty"`
}

type inlineData struct {
	MIMEType string `json:"mimeType"`
	// Data is base64-encoded, as the API requires.
	Data string `json:"data"`
}

type generationConfig struct {
	Temperature      *float32        `json:"temperature,omitempty"`
	MaxOutputTokens  int             `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string          `json:"responseMimeType,omitempty"`
	ResponseSchema   any             `json:"responseSchema,omitempty"`
	ThinkingConfig   *thinkingConfig `json:"thinkingConfig,omitempty"`
}

// thinkingConfig carries whichever reasoning control the target model's
// family understands. The 2.5 family takes a token budget; 3.x replaced it
// with a named level and hangs when handed a budget instead. Exactly one of
// these is ever set — see thinkingConfigFor.
type thinkingConfig struct {
	// ThinkingBudget is meaningful at 0 ("don't think"), so it is a pointer:
	// omitempty on a plain int would silently drop the only value we care
	// about sending.
	ThinkingBudget *int   `json:"thinkingBudget,omitempty"`
	ThinkingLevel  string `json:"thinkingLevel,omitempty"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content      content `json:"content"`
		FinishReason string  `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

type apiErrorEnvelope struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		// Details carries google.rpc.RetryInfo on quota errors. A daily
		// quota reset lands here as a delay of many hours, which is the
		// difference between "wait a moment" and "stop trying today".
		Details []struct {
			Type       string `json:"@type"`
			RetryDelay string `json:"retryDelay"`
		} `json:"details"`
	} `json:"error"`
}
