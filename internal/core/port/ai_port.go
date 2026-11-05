package port

import (
	"context"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// AIGenerator is a text generator backed by some provider. The core depends
// on this rather than on Google AI Studio directly, so the provider can be
// swapped or faked in tests.
type AIGenerator interface {
	Generate(ctx context.Context, req domain.AIRequest) (domain.AIResponse, error)
	// MaxCallDuration is how long one Generate can run in the worst case,
	// including every retry and the waits between them. Callers that impose
	// their own deadline derive it from this instead of guessing: a deadline
	// shorter than the retry budget silently truncates the retries and
	// reports a context error in place of the provider's real complaint.
	MaxCallDuration() time.Duration
}

// AdoptionSummarizer condenses an adopter's screening form into the short
// paragraph shown at the top of the owner's view of that request.
type AdoptionSummarizer interface {
	SummarizeAdoptionForm(ctx context.Context, in domain.AdoptionSummaryInput) (summary string, err error)
	// MaxCallDuration mirrors AIGenerator.MaxCallDuration for the caller
	// that schedules the work.
	MaxCallDuration() time.Duration
}
