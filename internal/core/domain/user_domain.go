package domain

import "time"

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

// UserPreference is one "a pet like this" entry. A user may keep any number of them per
// species (Dog #1, Dog #2, Cat #1 …); each is one Users_Preferences row.
type UserPreference struct {
	PetType  string `json:"petType" validate:"oneof=DOG CAT"`
	Breed    string `json:"breed"`
	Color    string `json:"color"`
	AgeGroup string `json:"ageGroup"`
	Gender   string `json:"gender"`
}

type UserUpdateRequest struct {
	Firstname   string           `json:"firstName" validate:"min=1"`
	Lastname    string           `json:"lastName" validate:"min=1"`
	PhoneNumber string           `json:"phoneNumber" validate:"min=9,max=10"`
	Address     string           `json:"address" validate:"min=1"`
	AddressLat  float64          `json:"addressLat" validate:"required"`
	AddressLong float64          `json:"addressLong" validate:"required"`
	Preferences []UserPreference `json:"preferences"`
}

type UserUpdateResponse struct {
	Message string `json:"message"`
}

type UserUpdateImageResponse struct {
	ImageAddress string `json:"imageAddress"`
	Message      string `json:"message"`
}

type UserInfo struct {
	Firstname   string
	Lastname    string
	Phone       string
	Address     string
	AddressLat  float64
	AddressLong float64
	CoverImage  *string
	Preferences []UserPreference
}

type UserInfoResponse struct {
	UserData UserInfo `json:"userData"`
	Message  string   `json:"message"`
}

type UserDeleteResponse struct {
	Message string `json:"message"`
}

type Notification struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	PetID     *int      `json:"petId,omitempty"`
	RehomeID  *int      `json:"rehomeId,omitempty"`
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdoptorInfo struct {
	UserID       string `json:"userId"`
	Firstname    string `json:"firstName"`
	Lastname     string `json:"lastName"`
	PhoneNumber  string `json:"phoneNumber"`
	Address      string `json:"address"`
	ImageAddress string `json:"imageAddress"`
	Rid          int    `json:"rehomeId"`
	RehomeStatus string `json:"rehomeStatus"`
}

type PetAdoptorsInfo struct {
	Pid             int           `json:"petId"`
	PetName         string        `json:"petName"`
	PetImageAddress string        `json:"petImageAddress"`
	AdoptorsInfo    []AdoptorInfo `json:"adoptorsInfo"`
}
