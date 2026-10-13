package port

import (
	"encoding/json"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type PetSQLRepository interface {
	GetPetsInfo(page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, petLocation string, keyword string, userLat float64, userLong float64) (petData []domain.PetsInfo, totalCount int, err error)
	GetPetInfo(pid int) (petData domain.PetInfo, err error)
	CreatePet(uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, personality json.RawMessage, specialCare json.RawMessage, sterilized bool, vaccination []string, address string, addressLat float64, addressLong float64, status string, note string) (id int, err error)
	UpdatePet(uid string, pid int, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, personality json.RawMessage, specialCare json.RawMessage, sterilized bool, vaccination []string, address string, addressLat float64, addressLong float64, note string) (removedImages []string, err error)
	PostPetAdopt(uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string, answers []domain.CustomAnswerInput) (rid int, err error)
	UpdatePetAdopter(rid int, uid string) (err error)
	GetPetsSuggestion() (petData []domain.PetsInfo, err error)
	GetScreeningAnswer(rid int, uid string) (answer domain.ScreeningAnswer, err error)
	GetAllAdoptors(uid string) (adoptors []domain.PetAdoptorsInfo, err error)
	DeletePet(uid string, pid int) (imageAddresses []string, err error)
	GetPetsByOwner(uid string) (petData []domain.PetsInfo, err error)
	GetMyAdoptionStatus(pid int, uid string) (status string, err error)
	GetMyAdoptionRequests(uid string) (requests []domain.MyAdoptionRequest, err error)
	CancelAdoptionRequest(uid string, rid int) (err error)
	GetScreeningQuestions(pid int) (questions []domain.CustomScreeningQuestion, locked bool, err error)
	SaveScreeningQuestions(uid string, pid int, questions []domain.ScreeningQuestionInput) (err error)

	GetMyDiaryPets(uid string) (pets []domain.DiaryPet, err error)
	// GetDiaryAccess reports how uid relates to the pet. Returns an error when
	// there is no relationship, so callers can refuse before touching entries.
	GetDiaryAccess(pid int, uid string) (role string, since string, err error)
	GetDiaryEntries(pid int, from string, to string) (entries []domain.DiaryEntry, err error)
	GetDiaryEntry(pid int, date string) (entry domain.DiaryEntry, err error)
	// UpsertDiaryEntry writes the day's row. imageAddress empty means "keep the
	// existing photo", which is only valid when a row already exists. The
	// replaced image URL comes back so the service can clean up storage.
	UpsertDiaryEntry(pid int, uid string, date string, imageAddress string, caption string) (entry domain.DiaryEntry, replacedImage string, err error)
	DeleteDiaryEntry(pid int, date string) (removedImage string, err error)
}
type PetMongoRepository interface {
	GetBreeds(petType string) (breedData []string, err error)
	GetBreedBehavior(petType string, petBreed string) (behaviorData string, err error)
	GetBreedColors(petType string, petBreed string) (colorData []domain.PetColorResponse, err error)
	GetBreedImages(petType string) (imageData []domain.PetBreedImageResponse, err error)
}