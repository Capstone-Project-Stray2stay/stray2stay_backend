package domain

import "encoding/json"

type PetRegisterRequest struct {
	PetName        string   `form:"petName" validate:"required"`
	PetDetail      string   `form:"petDetail"`
	PetAgeGroup    string   `form:"petAgeGroup" validate:"required"`
	PetGender      string   `form:"petGender" validate:"required"`
	PetType        string   `form:"petType" validate:"required"`
	PetBreed       string   `form:"petBreed" validate:"required"`
	PetColor       string   `form:"petColor" validate:"required"`
	PetPersonality []string `form:"petPersonality" validate:"required,min=1"`
	PetSpecialCare string   `form:"petSpecialCare"`
	PetSterilized  bool     `form:"petSterilized"`
	PetVaccination []string `form:"petVaccination" validate:"required,min=1,dive,oneof=DHPPi Rabies FVRCP"`
	PetAddress     string   `form:"petAddress" validate:"required"`
	PetAddressLat  float64  `form:"petAddressLat" validate:"required"`
	PetAddressLong float64  `form:"petAddressLong" validate:"required"`
	Status         string   `form:"status"`
	Note           string   `form:"note"`
}

type PetRegisterResponse struct {
	Pid     int    `json:"pid"`
	Message string `json:"message"`
}

type PetUpdateRequest struct {
	PetName        string   `form:"petName" validate:"required"`
	PetAgeGroup    string   `form:"petAgeGroup" validate:"required"`
	PetGender      string   `form:"petGender" validate:"required"`
	PetType        string   `form:"petType" validate:"required"`
	PetBreed       string   `form:"petBreed" validate:"required"`
	PetColor       string   `form:"petColor" validate:"required"`
	PetPersonality []string `form:"petPersonality" validate:"required,min=1"`
	PetSpecialCare string   `form:"petSpecialCare"`
	PetSterilized  bool     `form:"petSterilized"`
	PetVaccination []string `form:"petVaccination" validate:"required,min=1,dive,oneof=DHPPi Rabies FVRCP"`
	PetAddress     string   `form:"petAddress" validate:"required"`
	PetAddressLat  float64  `form:"petAddressLat" validate:"required"`
	PetAddressLong float64  `form:"petAddressLong" validate:"required"`
	ExistingImages []string `form:"existingImages"`
	Note           string   `form:"note"`
}

type PetUpdateResponse struct {
	Message string `json:"message"`
}

type PetGetInfoByIdRequest struct {
	Pid int `json:"petId" validate:"required,gt=0"`
}
type PetInfo struct {
	PetOwnerID      string          `json:"-"`
	Pid             int             `json:"pid"`
	PetName         string          `json:"petName"`
	PetDetail       string          `json:"petDetail"`
	PetImageAddress []string        `json:"petImageAddress"`
	PetPersonality  []string        `json:"petPersonality"`
	PetSpecialCare  json.RawMessage `json:"petSpecialCare"`
	PetAgeGroup     string          `json:"petAgeGroup"`
	PetGender       string          `json:"petGender"`
	PetType         string          `json:"petType"`
	PetBreed        string          `json:"petBreed"`
	PetColor        string          `json:"petColor"`
	PetSterilized   bool            `json:"petSterilized"`
	PetVaccination  []string        `json:"petVaccination"`
	PetAddress      string          `json:"petAddress"`
	PetAddressLat   float64         `json:"petAddressLat"`
	PetAddressLong  float64         `json:"petAddressLong"`
	Status          string          `json:"status"`
	Note            string          `json:"note"`
}

type PetGetInfoByIdResponse struct {
	PetInfo any    `json:"petInfo"`
	Message string `json:"message"`
}

type PetDeleteResponse struct {
	Message string `json:"message"`
}

type PetSearchFilterRequest struct {
	Page        int     `query:"page"`
	PageSize    int     `query:"pageSize"`
	PetAgeGroup string  `query:"petAgeGroup"`
	PetGender   string  `query:"petGender"`
	PetType     string  `query:"petType"`
	PetBreed    string  `query:"petBreed"`
	PetColor    string  `query:"petColor"`
	PetLocation string  `query:"petLocation"`
	UserLat     float64 `query:"userLat"`
	UserLong    float64 `query:"userLong"`
}

type PetSearchFilterResponse struct {
	PetsInfo   []PetsInfo `json:"petsInfo"`
	TotalCount int        `json:"totalCount"`
	TotalPages int        `json:"totalPages"`
	Message    string     `json:"message"`
}
type PetsInfoResponse struct {
	PetsInfo PetsInfo `json:"petsInfo"`
	Message  string   `json:"message"`
}

type PetsInfo struct {
	Pid             int      `json:"pid"`
	PetName         string   `json:"petName"`
	PetImageAddress []string `json:"petImageAddress"`
	PetAgeGroup     string   `json:"petAgeGroup"`
	PetGender       string   `json:"petGender"`
	PetType         string   `json:"petType"`
	PetBreed        string   `json:"petBreed"`
	PetColor        string   `json:"petColor"`
	PetAddress      string   `json:"petAddress"`
	PetAddressLat   float64  `json:"petAddressLat"`
	PetAddressLong  float64  `json:"petAddressLong"`
}

type PetAdoptRequest struct {
	Pid  int    `json:"pid" validate:"required,gt=0"`
	Q1_1 bool   `json:"q1_1"`
	Q1_2 bool   `json:"q1_2"`
	Q1_3 string `json:"q1_3" validate:"required"`
	Q2_1 string `json:"q2_1" validate:"required"`
	Q2_2 bool   `json:"q2_2"`
	Q2_3 bool   `json:"q2_3"`
	Q3_1 int8   `json:"q3_1" validate:"gte=0"`
	Q3_2 bool   `json:"q3_2"`
	Q3_3 string `json:"q3_3" validate:"required"`
	Q4_1 int8   `json:"q4_1" validate:"gte=0,lte=2"`
	Q5_1 int8   `json:"q5_1" validate:"gte=0,lte=3"`
	Q6_1 int8   `json:"q6_1" validate:"gte=0,lte=4"`
	Q6_2 int8   `json:"q6_2" validate:"gte=0,lte=4"`
	Note string `json:"note"`
}

type PetAdoptResponse struct {
	Rid     int    `json:"rid"`
	Message string `json:"message"`
}

type PetSelectAdopterRequest struct {
	Rid int `json:"rid"`
}

type PetSelectAdopterResponse struct {
	Message string `json:"message"`
}

type PetAIClassifyRequest struct {
	Type string `query:"type"`
}

type PetAIClassifyPrediction struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

type PetAIClassifyResponse struct {
	NumImages   int                       `json:"num_images"`
	Predictions []PetAIClassifyPrediction `json:"predictions"`
}

type PetBreedResponse struct {
	PetBreed string `json:"pet_breed"`
}

type PetColorResponse struct {
	Color string `bson:"color"`
	Image string `bson:"link"`
}

type PetBreed struct {
	BreedName string `bson:"breedName"`
}

type PetGetBreedsRequest struct {
	PetType string `query:"petType" validate:"required,oneof=cat dog"`
}

type PetGetBreedsResponse struct {
	Breeds  []string `json:"breeds"`
	Message string   `json:"message"`
}

type PetGetBehaviorRequest struct {
	PetType  string `query:"petType" validate:"required,oneof=cat dog"`
	PetBreed string `query:"petBreed" validate:"required"`
}

type PetBehaviorResponse struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message"`
}

type PetGetColorRequest struct {
	PetType  string `query:"petType" validate:"required,oneof=cat dog"`
	PetBreed string `query:"petBreed" validate:"required"`
}

type PetGetColorResponse struct {
	Colors  []string `json:"colors"`
	Message string   `json:"message"`
}

type ScreeningAnswerAdoptorRequest struct {
	Rid int `query:"rid" validate:"required"`
}

type MyAdoptionRequest struct {
	Rid             int      `json:"rid"`
	Pid             int      `json:"pid"`
	PetName         string   `json:"petName"`
	PetImageAddress []string `json:"petImageAddress"`
	RehomeStatus    string   `json:"rehomeStatus"`
	OwnerPhone      string   `json:"ownerPhone"`
}

type PetMyAdoptionRequestsResponse struct {
	AdoptionRequests []MyAdoptionRequest `json:"adoptionRequests"`
	Message          string              `json:"message"`
}

type PetCancelAdoptionResponse struct {
	Message string `json:"message"`
}

type ScreeningAnswer struct {
	Q1_1 bool
	Q1_2 bool
	Q1_3 string
	Q2_1 string
	Q2_2 bool
	Q2_3 bool
	Q3_1 int8
	Q3_2 bool
	Q3_3 string
	Q4_1 int8
	Q5_1 int8
	Q6_1 int8
	Q6_2 int8
	Note string
}
