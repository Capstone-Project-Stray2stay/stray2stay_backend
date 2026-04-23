package port

import (
	"encoding/json"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type PetSQLRepository interface {
	GetPetsInfo(page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, userLat float64, userLong float64) (petData []domain.PetsInfo, err error)
	GetPetInfo(pid int) (petData domain.PetInfo, err error)
	CreatePet(uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, healthCondition string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (id int, err error)
	PostPetAdopt(uid string, pid int, contact string) (rid int, err error)
	UpdatePetAdopter(rid int) (err error)
}
type PetMongoRepository interface {
	GetBreeds(petType string) (breedData []string, err error)
	GetBreedBehavior(petType string, petBreed string) (breedData string, err error)
	GetPetColor(petType string) (colorData []domain.PetColorResponse, err error)
}