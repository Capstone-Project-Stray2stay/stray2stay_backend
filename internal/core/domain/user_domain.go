package domain

type UserRegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Firstname string `json:"firstName" validate:"required,min=1"`
	Lastname  string `json:"lastName" validate:"required,min=1"`
}

type UserRegisterResponse struct {
	Message string `json:"message"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type UserLoginResponse struct {
	Message string `json:"message"`
}

type UserUpdateRequest struct {
	Firstname   string `json:"firstName" validate:"min=1"`
	Lastname    string `json:"lastName" validate:"min=1"`
	PhoneNumber string `json:"phoneNumber" validate:"min=9,max=10"`
	Address     string `json:"address" validate:"min=1"`
	DogBreed	string `json:"dogBreed"`
	DogColor	string `json:"dogColor"`
	DogAgeGroup	string `json:"dogAgeGroup"`
	DogGender	string `json:"dogGender"`
	CatBreed	string `json:"catBreed"`
	CatColor	string `json:"catColor"`
	CatAgeGroup	string `json:"catAgeGroup"`
	CatGender	string `json:"catGender"`
}

type UserUpdateResponse struct {
	Message string `json:"message"`
}

type UserInfo struct {
	Firstname string
	Lastname  string
	Phone     string
	Address   string
}

type UserInfoResponse struct {
	UserData UserInfo `json:"userData"`
	Message  string   `json:"message"`
}

type UserDeleteResponse struct {
	Message string `json:"message"`
}
