package service

import (
	"context"
	"encoding/json"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

type PetService interface {
	RegisterPet(ctx context.Context, uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, healthCondition string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (pid int, err error)
	SearchPets(ctx context.Context, page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, userLat float64, userLong float64) (petData []domain.PetsInfo, err error)
	PetInfo(ctx context.Context, pid int) (petData *domain.PetInfo, err error)
	AdoptPet(ctx context.Context, uid string, pid int, contact string) (rid int, err error)
	SelectPetAdopter(ctx context.Context, rid int) (err error)
	BreedInfo(ctx context.Context, petType string, petBreed string) (breedData string, err error)
	AllBreeds(ctx context.Context, petType string) (breedData []string, err error)
	PetColor(ctx context.Context, petType string) (colorData []domain.PetColorResponse, err error)
}

type PetServiceImpl struct {
	mysqlRepo port.PetSQLRepository
	mongoRepo port.PetMongoRepository
}

func NewPetService(mysqlRepo port.PetSQLRepository, mongoRepo port.PetMongoRepository) PetService {
	return &PetServiceImpl{
		mysqlRepo:   mysqlRepo,
		mongoRepo: mongoRepo,
	}
}

func (s *PetServiceImpl) RegisterPet(ctx context.Context, uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, healthCondition string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (pid int, err error) {
	pid, err = s.mysqlRepo.CreatePet(uid, petName, imageAddress, ageGroup, gender, petType, breed, color, healthCondition, sterilized, vaccination, address, addressLat, addressLong, status, note)
	if err != nil {
		return -1, err
	}
	return pid, nil
}

func (s *PetServiceImpl) SearchPets(ctx context.Context, page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, userLat float64, userLong float64) (petData []domain.PetsInfo, err error) {
	data, err := s.mysqlRepo.GetPetsInfo(page, pageSize, petAgeGroup, petGender, petType, petBreed, petColor, userLat, userLong)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *PetServiceImpl) PetInfo(ctx context.Context, pid int) (petData *domain.PetInfo, err error) {
	data, err := s.mysqlRepo.GetPetInfo(pid)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *PetServiceImpl) AdoptPet(ctx context.Context, uid string, pid int, contact string) (rid int, err error) {
	rid, err = s.mysqlRepo.PostPetAdopt(uid, pid, contact)
	if err != nil {
		return rid, err
	}
	return rid, nil
}

func (s *PetServiceImpl) SelectPetAdopter(ctx context.Context, rid int) (err error) {
	err = s.mysqlRepo.UpdatePetAdopter(rid)
	if err != nil {
		return err
	}
	return nil
}

func (s *PetServiceImpl) BreedInfo(ctx context.Context, petType string, petBreed string) (breedData string, err error) {
	breeds, err := s.mongoRepo.GetBreedBehavior(petType, petBreed)
	if err != nil {
		return "", err
	}
	return breeds, nil
}

func (s *PetServiceImpl) AllBreeds(ctx context.Context, petType string) (breedData []string, err error) {
	breeds, err := s.mongoRepo.GetBreeds(petType)
	if err != nil {
		return nil, err
	}
	return breeds, nil
}

func (s *PetServiceImpl) PetColor(ctx context.Context, petType string) (colorData []domain.PetColorResponse, err error) {
	colors, err := s.mongoRepo.GetPetColor(petType)
	if err != nil {
		return nil, err
	}
	return colors, nil
}