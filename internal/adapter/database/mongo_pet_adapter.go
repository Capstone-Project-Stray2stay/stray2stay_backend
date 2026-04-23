package adapter

import (
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type MongoPetAdapter struct {
	collection *mongo.Collection
}

func NewMongoPetAdapter(db *mongo.Database) *MongoPetAdapter {
	return &MongoPetAdapter{
		collection: db.Collection("pets"),
	}
}

func (m *MongoPetAdapter) GetBreeds(petType string) (breedData []string, err error) {
	return nil, nil
}

func (m *MongoPetAdapter) GetBreedBehavior(petType string, petBreed string) (breedData string, err error) {
	return "", nil
}

func (m *MongoPetAdapter) GetPetColor(petType string) (colorData []domain.PetColorResponse, err error) {
	return nil, nil
}