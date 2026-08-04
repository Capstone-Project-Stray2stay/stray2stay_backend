package adapter

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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

func parsePetType(petType string) (string, error) {
	switch petType {
	case "cat":
		return "catBreeds", nil
	case "dog":
		return "dogBreeds", nil
	default:
		return "", errors.New("invalid pet type")
	}
}

func (m *MongoPetAdapter) GetBreeds(petType string) (breedData []string, err error) {
	var collectionName string

	collectionName, err = parsePetType(petType)
	if err != nil {
		return nil, err
	}

	collection := m.collection.Database().Collection(collectionName)	
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	var breeds []domain.PetBreed
	
	if err := cursor.All(context.Background(), &breeds); err != nil {
		return nil, err
	}

	breedData = make([]string, 0, len(breeds))
	for _, breed := range breeds {
		breedData = append(breedData, breed.BreedName)
	}

	return breedData, nil
}

func (m *MongoPetAdapter) GetBreedBehavior(petType string, petBreed string) (breedData string, err error) {
	var collectionName string
	collectionName, err = parsePetType(petType)
	if err != nil {
		return "", err
	}
	collection := m.collection.Database().Collection(collectionName)

	opts := options.FindOne().SetProjection(bson.M{"personality": 1})

	var result struct {
		Personality string `bson:"personality"`
	}

	err = collection.FindOne(context.Background(), bson.M{"breedName": petBreed}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}

	return result.Personality, nil
}

func (m *MongoPetAdapter) GetBreedColors(petType string, petBreed string) (colorData []domain.PetColorResponse, err error) {
	var collectionName string
	collectionName, err = parsePetType(petType)
	if err != nil {
		return nil, err
	}
	collection := m.collection.Database().Collection(collectionName)

	opts := options.FindOne().SetProjection(bson.M{"possibleColors": 1})

	var result struct {
		PossibleColors []domain.PetColorResponse `bson:"possibleColors"`
	}

	err = collection.FindOne(context.Background(), bson.M{"breedName": petBreed}, opts).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result.PossibleColors, nil
}