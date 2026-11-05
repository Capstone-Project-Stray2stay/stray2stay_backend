package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// summaryLookupBudget is the slack added on top of the generator's own
// worst case to cover the repository reads that assemble the form. They are
// local queries, so this only has to be comfortably more than a few of them.
const summaryLookupBudget = 30 * time.Second

// fallbackSummaryTimeout bounds a generation when the summarizer cannot
// report a budget of its own.
const fallbackSummaryTimeout = 5 * time.Minute

// summaryTimeout is the deadline for one detached generation, derived from
// what the provider client could actually spend rather than hard-coded.
//
// A deadline shorter than the client's retry budget is worse than no
// deadline: it cuts the retry loop off mid-flight, so every failure reports
// "context deadline exceeded" and the provider's real complaint never
// surfaces. Deriving it means raising GEMINI_MAX_RETRIES or
// GEMINI_TIMEOUT_SECONDS can never leave this stale.
//
// Nobody waits on this goroutine — the adopter already has their
// confirmation — so a long budget costs nothing but a parked goroutine.
func (s *PetServiceImpl) summaryTimeout() time.Duration {
	if s.summarizer == nil {
		return fallbackSummaryTimeout
	}
	budget := s.summarizer.MaxCallDuration()
	if budget <= 0 {
		return fallbackSummaryTimeout
	}
	return budget + summaryLookupBudget
}

// summarizeAdoptionAsync generates the owner-facing briefing for a freshly
// submitted request. answers carry only question ids, so the question text is
// resolved from the pet's screening questions before summarizing.
func (s *PetServiceImpl) summarizeAdoptionAsync(rid int, pid int, answer domain.ScreeningAnswer, answers []domain.CustomAnswerInput) {
	if s.summarizer == nil {
		return
	}

	s.runSummaryJob(rid, func(ctx context.Context) (domain.AdoptionSummaryInput, error) {
		answer.CustomAnswers = s.describeCustomAnswers(pid, answers)
		return s.withPetContext(pid, answer), nil
	})
}

// resummarizeAdoptionAsync regenerates from a stored answer, which already
// carries its custom answers and pet context.
func (s *PetServiceImpl) resummarizeAdoptionAsync(rid int, answer domain.ScreeningAnswer) {
	if s.summarizer == nil {
		return
	}

	s.runSummaryJob(rid, func(ctx context.Context) (domain.AdoptionSummaryInput, error) {
		input := domain.AdoptionSummaryInput{
			PetName:  answer.PetName,
			PetType:  answer.PetType,
			PetBreed: answer.PetBreed,
			Answer:   answer,
		}
		if input.PetName == "" && answer.Pid > 0 {
			return s.withPetContext(answer.Pid, answer), nil
		}
		return input, nil
	})
}

// runSummaryJob performs one generation off the request path. The context is
// deliberately not derived from the caller's: Fiber recycles a request
// context as soon as its handler returns, and an adopter who closes the tab
// should not cancel the owner's summary.
func (s *PetServiceImpl) runSummaryJob(rid int, build func(ctx context.Context) (domain.AdoptionSummaryInput, error)) {
	if _, busy := s.summariesInFlight.LoadOrStore(rid, struct{}{}); busy {
		return
	}

	go func() {
		defer s.summariesInFlight.Delete(rid)
		// This goroutine is outside Fiber's recover middleware, so a panic
		// here would take the whole server down with it.
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[AdoptionSummary] rid %d: recovered from panic: %v", rid, r)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), s.summaryTimeout())
		defer cancel()

		input, err := build(ctx)
		if err != nil {
			log.Printf("[AdoptionSummary] rid %d: failed to assemble form: %v", rid, err)
			s.storeSummary(rid, "", domain.AdoptionSummaryFailed)
			return
		}

		started := time.Now()
		summary, err := s.summarizer.SummarizeAdoptionForm(ctx, input)
		elapsed := time.Since(started).Round(time.Millisecond)
		if err != nil {
			log.Printf("[AdoptionSummary] rid %d: failed after %s: %v", rid, elapsed, err)
			// Outages, quota and timeouts are all "try again later", not a
			// defect in this form, so leave the request PENDING for the
			// owner-view heal rather than marking it permanently failed.
			if errors.Is(err, domain.ErrAIUnavailable) ||
				errors.Is(err, domain.ErrAIRateLimited) ||
				errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, context.Canceled) {
				return
			}
			s.storeSummary(rid, "", domain.AdoptionSummaryFailed)
			return
		}

		log.Printf("[AdoptionSummary] rid %d: generated in %s", rid, elapsed)
		s.storeSummary(rid, summary, domain.AdoptionSummaryReady)
	}()
}

func (s *PetServiceImpl) storeSummary(rid int, summary string, status string) {
	if err := s.mysqlRepo.UpdateAdoptionSummary(rid, summary, status); err != nil {
		log.Printf("[AdoptionSummary] rid %d: failed to store %s summary: %v", rid, status, err)
	}
}

// withPetContext attaches the pet the request is about. A lookup failure is
// not fatal: a summary without the pet's name still helps the owner.
func (s *PetServiceImpl) withPetContext(pid int, answer domain.ScreeningAnswer) domain.AdoptionSummaryInput {
	input := domain.AdoptionSummaryInput{Answer: answer}

	petInfo, err := s.mysqlRepo.GetPetInfo(pid)
	if err != nil {
		log.Printf("[AdoptionSummary] pet %d: failed to load pet context: %v", pid, err)
		return input
	}

	input.PetName = petInfo.PetName
	input.PetType = petInfo.PetType
	input.PetBreed = petInfo.PetBreed
	return input
}

// describeCustomAnswers pairs each submitted answer with its question text so
// the model sees what was actually asked. Answers whose question has since
// been removed are dropped rather than shown without context.
func (s *PetServiceImpl) describeCustomAnswers(pid int, answers []domain.CustomAnswerInput) []domain.CustomAnswerResponse {
	if len(answers) == 0 {
		return nil
	}

	questions, _, err := s.mysqlRepo.GetScreeningQuestions(pid)
	if err != nil {
		log.Printf("[AdoptionSummary] pet %d: failed to load screening questions: %v", pid, err)
		return nil
	}

	byID := make(map[int]domain.CustomScreeningQuestion, len(questions))
	for _, question := range questions {
		byID[question.QuestionID] = question
	}

	described := make([]domain.CustomAnswerResponse, 0, len(answers))
	for _, answer := range answers {
		question, ok := byID[answer.QuestionID]
		if !ok {
			continue
		}
		described = append(described, domain.CustomAnswerResponse{
			QuestionID:   answer.QuestionID,
			QuestionText: question.QuestionText,
			QuestionType: question.QuestionType,
			Value:        answer.Value,
		})
	}

	return described
}
