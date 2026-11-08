package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

// summaryUpdate is one call to UpdateAdoptionSummary, captured so the
// detached job can be asserted from the test goroutine.
type summaryUpdate struct {
	rid     int
	summary string
	status  string
}

// fakePetRepo implements only the handful of repository methods the summary
// path touches. The embedded nil interface makes any other call panic
// loudly, which is what we want if the path starts reaching further.
type fakePetRepo struct {
	port.PetSQLRepository

	rid       int
	petInfo   domain.PetInfo
	questions []domain.CustomScreeningQuestion
	updates   chan summaryUpdate
}

func newFakePetRepo(rid int) *fakePetRepo {
	return &fakePetRepo{
		rid:     rid,
		petInfo: domain.PetInfo{Pid: 7, PetName: "Mochi", PetType: "CAT", PetBreed: "Siamese"},
		updates: make(chan summaryUpdate, 4),
	}
}

func (f *fakePetRepo) PostPetAdopt(uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string, answers []domain.CustomAnswerInput) (int, error) {
	return f.rid, nil
}

func (f *fakePetRepo) GetPetInfo(pid int) (domain.PetInfo, error) {
	return f.petInfo, nil
}

func (f *fakePetRepo) GetScreeningQuestions(pid int) ([]domain.CustomScreeningQuestion, bool, error) {
	return f.questions, false, nil
}

func (f *fakePetRepo) UpdateAdoptionSummary(rid int, summary string, status string) error {
	f.updates <- summaryUpdate{rid: rid, summary: summary, status: status}
	return nil
}

// stubSummarizer adapts a function to port.AdoptionSummarizer.
type stubSummarizer struct {
	fn func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error)
}

func summarizerFunc(fn func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error)) *stubSummarizer {
	return &stubSummarizer{fn: fn}
}

func (s *stubSummarizer) SummarizeAdoptionForm(ctx context.Context, in domain.AdoptionSummaryInput) (string, error) {
	return s.fn(ctx, in)
}

func (s *stubSummarizer) MaxCallDuration() time.Duration { return 30 * time.Second }

func adoptWithStubs(t *testing.T, repo *fakePetRepo, summarizer port.AdoptionSummarizer, answers []domain.CustomAnswerInput) {
	t.Helper()

	service := NewPetService(repo, nil, nil, nil, summarizer)
	if _, err := service.AdoptPet(context.Background(), "user-1", 7,
		true, false, "lives with partner",
		"Condo", true, false,
		2, true, "two rescue cats",
		1, 2, 3, 4, "weekend visits ok", answers); err != nil {
		t.Fatalf("AdoptPet() error = %v", err)
	}
}

func waitForUpdate(t *testing.T, repo *fakePetRepo) summaryUpdate {
	t.Helper()
	select {
	case update := <-repo.updates:
		return update
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the summary to be stored")
		return summaryUpdate{}
	}
}

func TestAdoptPetStoresGeneratedSummary(t *testing.T) { // UT-BE-ADSM-01
	repo := newFakePetRepo(42)
	repo.questions = []domain.CustomScreeningQuestion{
		{QuestionID: 9, QuestionText: "Who else lives with you?", QuestionType: "ESSAY"},
	}

	seen := make(chan domain.AdoptionSummaryInput, 1)
	adoptWithStubs(t, repo, summarizerFunc(func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error) {
		seen <- in
		return "A condo adopter with two rescue cats.", nil
	}), []domain.CustomAnswerInput{
		{QuestionID: 9, Value: json.RawMessage(`"My partner"`)},
		// Answer to a question that no longer exists; it must be dropped
		// rather than shown to the model without its wording.
		{QuestionID: 404, Value: json.RawMessage(`"orphan"`)},
	})

	update := waitForUpdate(t, repo)
	if update.rid != 42 {
		t.Errorf("stored against rid %d; want 42", update.rid)
	}
	if update.status != domain.AdoptionSummaryReady {
		t.Errorf("status = %q; want %q", update.status, domain.AdoptionSummaryReady)
	}
	if update.summary != "A condo adopter with two rescue cats." {
		t.Errorf("summary = %q; want the generated text", update.summary)
	}

	in := <-seen
	if in.PetName != "Mochi" || in.PetBreed != "Siamese" {
		t.Errorf("pet context = %+v; want it loaded from the repository", in)
	}
	if in.Answer.Q2_1 != "Condo" || in.Answer.Note != "weekend visits ok" {
		t.Errorf("answer = %+v; want the submitted form", in.Answer)
	}
	if len(in.Answer.CustomAnswers) != 1 {
		t.Fatalf("custom answers = %d; want only the one whose question still exists", len(in.Answer.CustomAnswers))
	}
	if in.Answer.CustomAnswers[0].QuestionText != "Who else lives with you?" {
		t.Errorf("custom answer text = %q; want it resolved from the question", in.Answer.CustomAnswers[0].QuestionText)
	}
}

func TestAdoptPetMarksSummaryFailedOnPermanentError(t *testing.T) { // UT-BE-ADSM-02
	repo := newFakePetRepo(43)

	adoptWithStubs(t, repo, summarizerFunc(func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error) {
		return "", domain.ErrAIBlocked
	}), nil)

	update := waitForUpdate(t, repo)
	if update.status != domain.AdoptionSummaryFailed {
		t.Errorf("status = %q; want %q", update.status, domain.AdoptionSummaryFailed)
	}
	if update.summary != "" {
		t.Errorf("summary = %q; a failed generation must not store text", update.summary)
	}
}

func TestAdoptPetLeavesSummaryPendingOnTransientError(t *testing.T) { // UT-BE-ADSM-02
	repo := newFakePetRepo(44)

	done := make(chan struct{})
	adoptWithStubs(t, repo, summarizerFunc(func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error) {
		defer close(done)
		return "", domain.ErrAIUnavailable
	}), nil)

	<-done
	select {
	case update := <-repo.updates:
		t.Fatalf("stored status %q; a provider outage should leave the request PENDING so it can be retried", update.status)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAdoptPetWithoutSummarizerDoesNothing(t *testing.T) { // UT-BE-ADSM-03
	repo := newFakePetRepo(45)

	adoptWithStubs(t, repo, nil, nil) // nil interface: AI disabled

	select {
	case update := <-repo.updates:
		t.Fatalf("stored status %q; a deployment without AI must not touch the summary", update.status)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSummaryTimeoutCoversTheClientRetryBudget(t *testing.T) { // UT-BE-ADSM-04
	// A deadline shorter than the generator's own budget truncates its
	// retries and hides the real error, which is exactly the failure this
	// derivation exists to prevent.
	service := NewPetService(newFakePetRepo(1), nil, nil, nil,
		summarizerFunc(func(ctx context.Context, in domain.AdoptionSummaryInput) (string, error) {
			return "", nil
		})).(*PetServiceImpl)

	got := service.summaryTimeout()
	if budget := service.summarizer.MaxCallDuration(); got <= budget {
		t.Errorf("summaryTimeout() = %s; must exceed the generator budget of %s", got, budget)
	}
}

func TestSummaryTimeoutFallsBackWithoutASummarizer(t *testing.T) { // UT-BE-ADSM-04
	service := NewPetService(newFakePetRepo(1), nil, nil, nil, nil).(*PetServiceImpl)

	if got := service.summaryTimeout(); got != fallbackSummaryTimeout {
		t.Errorf("summaryTimeout() = %s; want the fallback %s", got, fallbackSummaryTimeout)
	}
}
