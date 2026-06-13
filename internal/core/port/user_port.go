package port

import "github.com/S-nudhana/stray2stay/internal/core/domain"

type UserRepository interface {
	CreateUser(email string, password string, firstName string, lastName string) (err error)
	RemoveUser(uid string) (err error)
	AuthenticateUser(email string, password string) (uid string, err error)
	OAuthAuthenticateUser(email string, provider string, firstName string, lastName string) (uid string, err error)
	UpdateUserInfo(uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, dogBreed string, dogColor string, dogAgeGroup string, dogGender string, catBreed string, catColor string, catAgeGroup string, catGender string) (err error)
	GetUserInfo(uid string) (userInfo *domain.UserInfo, err error)
}