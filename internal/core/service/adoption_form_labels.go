package service

import "strconv"

// fixedScreeningQuestion describes one of the built-in Q1_1..Q6_2 fields so
// the AI summarizer can show the model a readable form instead of opaque
// field names.
//
// These mirror SCREENING_SECTIONS in the frontend
// (src/utils/screeningForm.ts), which is where the questions adopters
// actually see are defined. Nothing enforces that the two stay in step, so
// edit them together: the summary is only as accurate as this wording, and a
// stale option list silently mislabels an adopter's answer rather than
// failing.
//
// Options are indexed exactly as the frontend sends them, because the
// index is what the database stores.
type fixedScreeningQuestion struct {
	Key   string
	Label string
	// Options maps a stored index to its answer text. Empty means the value
	// is not an index and is rendered as-is.
	Options []string
}

var (
	questionQ1_1 = fixedScreeningQuestion{
		Key:   "1.1",
		Label: "Have you ever owned a dog or cat before?",
	}
	questionQ1_2 = fixedScreeningQuestion{
		Key:   "1.2",
		Label: "Are you familiar with the basic vaccination and medical needs for this type of pet?",
	}
	questionQ1_3 = fixedScreeningQuestion{
		Key:   "1.3",
		Label: "Why have you decided to adopt a pet at this time?",
	}
	questionQ2_1 = fixedScreeningQuestion{
		Key:   "2.1",
		Label: "What is your current type of residence?",
	}
	questionQ2_2 = fixedScreeningQuestion{
		Key:   "2.2",
		Label: "Does your residence explicitly allow pets?",
	}
	questionQ2_3 = fixedScreeningQuestion{
		Key:   "2.3",
		Label: "Do you have a secure, fenced area or a dedicated indoor space for the pet?",
	}
	questionQ3_1 = fixedScreeningQuestion{
		Key:   "3.1",
		Label: "On average, how many hours per day will the pet be left alone?",
	}
	questionQ3_2 = fixedScreeningQuestion{
		Key:   "3.2",
		Label: "Does everyone in your household agree to adopt this pet?",
	}
	questionQ3_3 = fixedScreeningQuestion{
		Key:   "3.3",
		Label: "Do you have other pets? If yes, are they socialized and up-to-date on vaccinations?",
	}
	questionQ4_1 = fixedScreeningQuestion{
		Key:   "4.1",
		Label: "Are you prepared for the ongoing costs of pet ownership (e.g., food, grooming, and routine/emergency veterinary care)?",
		Options: []string{
			"Yes, I have a stable budget for all pet expenses",
			"I am prepared for basics, but might need help with emergencies.",
			"No, I am currently on a tight budget.",
		},
	}
	questionQ5_1 = fixedScreeningQuestion{
		Key:   "5.1",
		Label: "Are you willing and able to manage a pet with Special Needs (e.g., daily medication, frequent vet visits, or physical therapy)?",
		Options: []string{
			"Yes, I am fully prepared and have experience with special needs pets.",
			"Yes, I am willing to learn and provide the necessary care.",
			"It depends on the specific condition of the pet.",
			"No, I am looking for a pet with no major health issues.",
		},
	}
	questionQ6_1 = fixedScreeningQuestion{
		Key:   "6.1",
		Label: "Are you willing to provide regular updates on the pet's well-being through our Diary feature?",
		Options: []string{
			"Yes, I'm happy to provide frequent and consistent updates.",
			"Yes, I can provide updates on a monthly or occasional basis.",
			"I have a busy schedule but will try my best to update whenever possible.",
			"I prefer not to provide regular updates but can respond to direct inquiries.",
			"I am not comfortable sharing updates through the platform.",
		},
	}
	questionQ6_2 = fixedScreeningQuestion{
		Key:   "6.2",
		Label: "If you can no longer care for a pet, do you agree to return it to the finder or rehome it through this platform instead of abandoning it?",
		Options: []string{
			"Yes, I fully agree to return the pet or use this platform to find a new home.",
			"Yes, I agree, but I would like to discuss the return process with the finder first.",
			"I will try to find a responsible friend or family member first, then inform the finder.",
			"I am not sure at the moment.",
			"No, I believe I should have full discretion over the pet's future once adopted.",
		},
	}
)

// choiceLabel renders a stored index using the question's option list. An
// index outside the list means the form changed after this answer was
// submitted, so it is reported as unknown rather than silently matched to
// whatever text now sits at that position.
func (q fixedScreeningQuestion) choiceLabel(index int8) string {
	if index >= 0 && int(index) < len(q.Options) {
		return q.Options[index]
	}
	return "(unrecognised option #" + strconv.Itoa(int(index)) + ")"
}
