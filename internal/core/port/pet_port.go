package port

import (
	"encoding/json"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type PetSQLRepository interface {
	GetPetsInfo(page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, userLat float64, userLong float64) (petData []domain.PetsInfo, err error)
	GetPetInfo(pid int) (petData domain.PetInfo, err error)
	CreatePet(uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, personality json.RawMessage, specialCare string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (id int, err error)
	PostPetAdopt(uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string) (rid int, err error)
	UpdatePetAdopter(rid int) (err error)
	GetPetsSuggestion() (petData []domain.PetsInfo, err error)
}
type PetMongoRepository interface {
	GetBreeds(petType string) (breedData []string, err error)
	GetBreedBehavior(petType string, petBreed string) (breedData string, err error)
	GetPetColor(petType string) (colorData []domain.PetColorResponse, err error)
}