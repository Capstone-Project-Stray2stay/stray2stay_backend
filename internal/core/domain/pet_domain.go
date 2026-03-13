package domain

import "encoding/json"

type PetRegisterRequest struct {
	PetName            string          `json:"petName" validate:"required,min=0"`
	PetDetail          string          `json:"petDetail"`
	PetImageAddress    json.RawMessage `json:"petImageAddress" validate:"required"`
	PetAgeGroup        string          `json:"petAgeGroup" validate:"required,min=0"`
	PetGender          string          `json:"petGender" validate:"required,min=0"`
	PetType            string          `json:"petType" validate:"required,min=0"`
	PetBreed           string          `json:"petBreed" validate:"required,min=0"`
	PetColor           string          `json:"petColor" validate:"required,min=0"`
	PetHealthCondition string          `json:"petHealthCondition" validate:"required,min=0"`
	PetSterilized      bool            `json:"petSterilized" validate:"required"`
	PetVaccination     bool            `json:"petVaccination" validate:"required"`
	PetAddress         string          `json:"petAddress" validate:"required,min=0"`
	PetAddressLat      float64         `json:"petAddressLat" validate:"required"`
	PetAddressLong     float64         `json:"petAddressLong" validate:"required"`
	Status             bool            `json:"status" validate:"required"`
	Note               string          `json:"note"`
}

type PetRegisterResponse struct {
	Pid     int    `json:"pid"`
	Message string `json:"message"`
}

type PetGetInfoByIdRequest struct {
	Pid int `json:"petId" validate:"required,gt=0"`
}
type PetInfo struct {
	Pid                int             `json:"pid"`
	PetName            string          `json:"petName"`
	PetDetail          string          `json:"petDetail"`
	PetImageAddress    json.RawMessage `json:"petImageAddress"`
	PetAgeGroup        string          `json:"petAgeGroup"`
	PetGender          string          `json:"petGender"`
	PetType            string          `json:"petType"`
	PetBreed           string          `json:"petBreed"`
	PetColor           string          `json:"petColor"`
	PetHealthCondition string          `json:"petHealthCondition"`
	PetSterilized      bool            `json:"petSterilized"`
	PetVaccination     bool            `json:"petVaccination"`
	PetAddress         string          `json:"petAddress"`
	PetAddressLat      float64         `json:"petAddressLat"`
	PetAddressLong     float64         `json:"petAddressLong"`
	Status             bool            `json:"status"`
	Note               string          `json:"note"`
}

type PetGetInfoByIdResponse struct {
	PetInfo any    `json:"petInfo"`
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
	UserLat     float64 `query:"userLat"`
	UserLong    float64 `query:"userLong"`
}

type PetSearchFilterResponse struct {
	PetsInfo PetsInfo `json:"petsInfo"`
	Message  string   `json:"message"`
}

type PetsInfo struct {
	Pid             int             `json:"pid"`
	PetName         string          `json:"petName"`
	PetImageAddress json.RawMessage `json:"petImageAddress"`
	PetAgeGroup     string          `json:"petAgeGroup"`
	PetGender       string          `json:"petGender"`
	PetType         string          `json:"petType"`
	PetBreed        string          `json:"petBreed"`
	PetColor        string          `json:"petColor"`
	PetAddress      string          `json:"petAddress"`
	PetAddressLat   float64         `json:"petAddressLat"`
	PetAddressLong  float64         `json:"petAddressLong"`
}

type PetAdoptRequest struct {
	Pid     int    `json:"pid" validate:"required,gt=0"`
	Contact string `json:"contact" validate:"required,min=0"`
}

type PetSelectAdopterRequest struct {
	Rid     int    `json:"rid"`
}