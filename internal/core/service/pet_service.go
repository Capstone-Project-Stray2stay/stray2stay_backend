package service

import (
	"context"
	"encoding/json"
	"log"
	"mime/multipart"
	"strings"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

type PetService interface {
	RegisterPet(ctx context.Context, uid string, petName string, files []*multipart.FileHeader, ageGroup string, gender string, petType string, breed string, color string, personality []string, specialCare string, sterilized bool, vaccination []string, address string, addressLat float64, addressLong float64, status string, note string) (pid int, err error)
	SearchPets(ctx context.Context, uid string, page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, petLocation string, userLat float64, userLong float64) (petData []domain.PetsInfo, totalCount int, err error)
	PetInfo(ctx context.Context, pid int) (petData *domain.PetInfo, err error)
	AdoptPet(ctx context.Context, uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string) (rid int, err error)
	SelectPetAdopter(ctx context.Context, rid int) (err error)
	AllBreeds(ctx context.Context, petType string) (breedData []string, err error)
	PetColor(ctx context.Context, petType string, petBreed string) (colorData []domain.PetColorResponse, err error)
	PetRandom(ctx context.Context) (petData []domain.PetsInfo, err error)
	PetBehavior(ctx context.Context, petType string, petBreed string) (behaviorData string, err error)
	ScreeningAnswerAdoptor(ctx context.Context, screeningAnswerAdoptorPayload *domain.ScreeningAnswerAdoptorRequest) (answer domain.ScreeningAnswer, err error)
	AllAdoptors(ctx context.Context, uid string) (adoptors []domain.PetAdoptorsInfo, err error)
	DeletePet(ctx context.Context, uid string, pid int) (err error)
}

type PetServiceImpl struct {
	mysqlRepo port.PetSQLRepository
	mongoRepo port.PetMongoRepository
	uploader  port.ImageUploader
	userRepo  port.UserMySQLRepository
}

func NewPetService(mysqlRepo port.PetSQLRepository, mongoRepo port.PetMongoRepository, uploader port.ImageUploader, userRepo port.UserMySQLRepository) PetService {
	return &PetServiceImpl{
		mysqlRepo: mysqlRepo,
		mongoRepo: mongoRepo,
		uploader:  uploader,
		userRepo:  userRepo,
	}
}

func (s *PetServiceImpl) RegisterPet(ctx context.Context, uid string, petName string, files []*multipart.FileHeader, ageGroup string, gender string, petType string, breed string, color string, personality []string, specialCare string, sterilized bool, vaccination []string, address string, addressLat float64, addressLong float64, status string, note string) (pid int, err error) {
	urls, err := s.uploader.UploadImages(files, "pets")
	if err != nil {
		return -1, err
	}

	imageJSON, err := json.Marshal(urls)
	if err != nil {
		return -1, err
	}

	personalityJSON, err := json.Marshal(personality)
	if err != nil {
		return -1, err
	}

	specialCareJSON, err := json.Marshal(specialCare)
	if err != nil {
		return -1, err
	}

	pid, err = s.mysqlRepo.CreatePet(uid, petName, imageJSON, ageGroup, gender, petType, breed, color, personalityJSON, specialCareJSON, sterilized, vaccination, address, addressLat, addressLong, "AVALIABLE", note)
	if err != nil {
		return -1, err
	}
	return pid, nil
}

func (s *PetServiceImpl) SearchPets(ctx context.Context, uid string, page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, petLocation string, userLat float64, userLong float64) (petData []domain.PetsInfo, totalCount int, err error) {
	if uid != "" {
		petAgeGroup, petGender, petBreed, petColor, petLocation = s.applyUserDefaults(
			uid, petType, petAgeGroup, petGender, petBreed, petColor, petLocation,
		)
	}

	data, totalCount, err := s.mysqlRepo.GetPetsInfo(page, pageSize, petAgeGroup, petGender, petType, petBreed, petColor, petLocation, userLat, userLong)
	if err != nil {
		return nil, 0, err
	}
	return data, totalCount, nil
}

// applyUserDefaults fills any blank filter with the logged-in user's saved
// defaults: breed/color/gender/ageGroup from Users_Preferences (species-
// specific, so these only apply once petType is "dog" or "cat" — with no
// species picked there's no single preference row to fall back to), and
// location from the user's own profile address. A lookup failure is treated
// as "no defaults available" rather than failing the whole search.
func (s *PetServiceImpl) applyUserDefaults(uid string, petType string, petAgeGroup string, petGender string, petBreed string, petColor string, petLocation string) (string, string, string, string, string) {
	userInfo, err := s.userRepo.GetUserInfo(uid)
	if err != nil {
		return petAgeGroup, petGender, petBreed, petColor, petLocation
	}

	switch petType {
	case "dog":
		if petBreed == "" {
			petBreed = userInfo.DogBreed
		}
		if petColor == "" {
			petColor = userInfo.DogColor
		}
		if petAgeGroup == "" {
			petAgeGroup = userInfo.DogAgeGroup
		}
		if petGender == "" {
			petGender = userInfo.DogGender
		}
	case "cat":
		if petBreed == "" {
			petBreed = userInfo.CatBreed
		}
		if petColor == "" {
			petColor = userInfo.CatColor
		}
		if petAgeGroup == "" {
			petAgeGroup = userInfo.CatAgeGroup
		}
		if petGender == "" {
			petGender = userInfo.CatGender
		}
	}

	if petLocation == "" {
		petLocation = provinceFromAddress(userInfo.Address)
	}

	return petAgeGroup, petGender, petBreed, petColor, petLocation
}

// provinceFromAddress pulls the province out of an address built by the
// frontend's joinAddress/resolveLocation as "street, subDistrict, district,
// province" — the province is always the last comma-separated segment.
func provinceFromAddress(address string) string {
	parts := strings.Split(address, ",")
	last := strings.TrimSpace(parts[len(parts)-1])
	return last
}

func (s *PetServiceImpl) PetInfo(ctx context.Context, pid int) (petData *domain.PetInfo, err error) {
	data, err := s.mysqlRepo.GetPetInfo(pid)
	if err != nil {
		return nil, err
	}

	// pet_detail isn't a MySQL column — the write-up text lives in Mongo's
	// per-breed personality field, keyed by breedName against Pets.pet_breed.
	// A lookup failure here shouldn't fail the whole request, same reasoning
	// as applyUserDefaults: missing enrichment data isn't worth a 500.
	// GetBreedBehavior's collection lookup only recognizes lowercase
	// "dog"/"cat", but pet_type is stored as "DOG"/"CAT".
	if detail, err := s.mongoRepo.GetBreedBehavior(strings.ToLower(data.PetType), data.PetBreed); err == nil {
		data.PetDetail = detail
	}

	return &data, nil
}

func (s *PetServiceImpl) AdoptPet(ctx context.Context, uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string) (rid int, err error) {
	rid, err = s.mysqlRepo.PostPetAdopt(uid, pid, q1_1, q1_2, q1_3, q2_1, q2_2, q2_3, q3_1, q3_2, q3_3, q4_1, q5_1, q6_1, q6_2, note)
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

func (s *PetServiceImpl) AllBreeds(ctx context.Context, petType string) (breedData []string, err error) {
	breeds, err := s.mongoRepo.GetBreeds(petType)
	if err != nil {
		return nil, err
	}
	return breeds, nil
}

func (s *PetServiceImpl) PetColor(ctx context.Context, petType string, petBreed string) (colorData []domain.PetColorResponse, err error) {
	colors, err := s.mongoRepo.GetBreedColors(petType, petBreed)
	if err != nil {
		return nil, err
	}
	return colors, nil
}

func (s *PetServiceImpl) PetRandom(ctx context.Context) (petData []domain.PetsInfo, err error) {
	data, err := s.mysqlRepo.GetPetsSuggestion()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *PetServiceImpl) PetBehavior(ctx context.Context, petType string, petBreed string) (behaviorData string, err error) {
	behaviors, err := s.mongoRepo.GetBreedBehavior(petType, petBreed)
	if err != nil {
		return "", err
	}
	return behaviors, nil
}

func (s *PetServiceImpl) ScreeningAnswerAdoptor(ctx context.Context, screeningAnswerAdoptorPayload *domain.ScreeningAnswerAdoptorRequest) (answer domain.ScreeningAnswer, err error) {
	screeningAnswer, err := s.mysqlRepo.GetScreeningAnswer(screeningAnswerAdoptorPayload.Rid)
	if err != nil {
		return domain.ScreeningAnswer{}, err
	}
	return screeningAnswer, nil
}

func (s *PetServiceImpl) AllAdoptors(ctx context.Context, uid string) (adoptors []domain.PetAdoptorsInfo, err error) {
	adoptors, err = s.mysqlRepo.GetAllAdoptors(uid)
	if err != nil {
		return nil, err
	}
	return adoptors, nil
}

func (s *PetServiceImpl) DeletePet(ctx context.Context, uid string, pid int) (err error) {
	imageAddresses, err := s.mysqlRepo.DeletePet(uid, pid)
	if err != nil {
		return err
	}

	// Best-effort: the pet row is already gone, so a stray Cloudinary asset
	// shouldn't fail the request — same reasoning as UpdateUserImage's old
	// image cleanup.
	for _, imageURL := range imageAddresses {
		if deleteErr := s.uploader.DeleteImage(imageURL); deleteErr != nil {
			log.Printf("[DeletePet] failed to delete image %q for pet %d: %v", imageURL, pid, deleteErr)
		}
	}

	return nil
}