package domain

type PetRegisterRequest struct {
	PetName            string   `json:"petName" validate:"required,min=0"`
	PetDetail          string   `json:"petDetail"`
	PetImageAddress    []string `json:"petImageAddress" validate:"required" swaggertype:"array,string"`
	PetAgeGroup        string   `json:"petAgeGroup" validate:"required,min=0"`
	PetGender          string   `json:"petGender" validate:"required,min=0"`
	PetType            string   `json:"petType" validate:"required,min=0"`
	PetBreed           string   `json:"petBreed" validate:"required,min=0"`
	PetColor           string   `json:"petColor" validate:"required,min=0"`
	PetPersonality     []string `json:"petPersonality" validate:"required" swaggertype:"array,string"`
	PetSpecialCare     string   `json:"petSpecialCare" validate:"required,min=0"`
	PetHealthCondition string   `json:"petHealthCondition" validate:"required,min=0"`
	PetSterilized      bool     `json:"petSterilized" validate:"required"`
	PetVaccination     bool     `json:"petVaccination" validate:"required"`
	PetAddress         string   `json:"petAddress" validate:"required,min=0"`
	PetAddressLat      float64  `json:"petAddressLat" validate:"required"`
	PetAddressLong     float64  `json:"petAddressLong" validate:"required"`
	Status             bool     `json:"status" validate:"required"`
	Note               string   `json:"note"`
}

type PetRegisterResponse struct {
	Pid     int    `json:"pid"`
	Message string `json:"message"`
}

type PetGetInfoByIdRequest struct {
	Pid int `json:"petId" validate:"required,gt=0"`
}
type PetInfo struct {
	Pid                int      `json:"pid"`
	PetName            string   `json:"petName"`
	PetDetail          string   `json:"petDetail"`
	PetImageAddress    []string `json:"petImageAddress"`
	PetAgeGroup        string   `json:"petAgeGroup"`
	PetGender          string   `json:"petGender"`
	PetType            string   `json:"petType"`
	PetBreed           string   `json:"petBreed"`
	PetColor           string   `json:"petColor"`
	PetHealthCondition string   `json:"petHealthCondition"`
	PetSterilized      bool     `json:"petSterilized"`
	PetVaccination     bool     `json:"petVaccination"`
	PetAddress         string   `json:"petAddress"`
	PetAddressLat      float64  `json:"petAddressLat"`
	PetAddressLong     float64  `json:"petAddressLong"`
	Status             bool     `json:"status"`
	Note               string   `json:"note"`
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
	Q1_1 bool   `json:"q1_1" validate:"required"`
	Q1_2 bool   `json:"q1_2" validate:"required"`
	Q1_3 string `json:"q1_3" validate:"required"`
	Q2_1 string `json:"q2_1" validate:"required"`
	Q2_2 bool   `json:"q2_2" validate:"required"`
	Q2_3 bool   `json:"q2_3" validate:"required"`
	Q3_1 int8   `json:"q3_1" validate:"required"`
	Q3_2 bool   `json:"q3_2" validate:"required"`
	Q3_3 string `json:"q3_3" validate:"required"`
	Q4_1 int8   `json:"q4_1" validate:"required" range:"0,3"`
	Q5_1 int8   `json:"q5_1" validate:"required" range:"0,3"`
	Q6_1 int8   `json:"q6_1" validate:"required" range:"0,3"`
	Q6_2 int8   `json:"q6_2" validate:"required" range:"0,3"`
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
	Color string `json:"color"`
	Image string `json:"image"`
}
