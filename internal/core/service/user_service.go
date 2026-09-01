package service

import (
	"context"
	"log"
	"mime/multipart"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

type UserService interface {
	OAuthLogin(ctx context.Context, email string, provider string, firstName string, lastName string) (uid string, err error)
	Login(ctx context.Context, email string, password string) (uid string, err error)
	Register(ctx context.Context, email string, password string, firstName string, lastName string) (err error)
	DeleteUser(ctx context.Context, uid string) (err error)
	UpdateUser(ctx context.Context, uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, dogBreed string, dogColor string, dogAgeGroup string, dogGender string, catBreed string, catColor string, catAgeGroup string, catGender string) (err error)
	UpdateUserImage(ctx context.Context, uid string, file *multipart.FileHeader) (imageURL string, err error)
	UserInfo(ctx context.Context, uid string) (userData *domain.UserInfo, err error)
	NewUserStatus(ctx context.Context, uid string) (userStatus bool, err error)
	UpdateNewUserStatus(ctx context.Context, uid string) (userStatus bool, err error)
}

type UserServiceImpl struct {
	userRepo port.UserMySQLRepository
	uploader port.ImageUploader
}

func NewUserService(userRepo port.UserMySQLRepository, uploader port.ImageUploader) UserService {
	return &UserServiceImpl{userRepo: userRepo, uploader: uploader}
}

func (s *UserServiceImpl) OAuthLogin(ctx context.Context, email string, provider string, firstName string, lastName string) (uid string, err error) {
	uid, err = s.userRepo.OAuthAuthenticateUser(email, provider, firstName, lastName)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (s *UserServiceImpl) Login(ctx context.Context, email string, password string) (uid string, err error) {
	uid, err = s.userRepo.AuthenticateUser(email, password)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (s *UserServiceImpl) Register(ctx context.Context, email string, password string, firstName string, lastName string) (err error) {
	err = s.userRepo.CreateUser(email, password, firstName, lastName)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, uid string) (err error) {
	err = s.userRepo.RemoveUser(uid)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, dogBreed string, dogColor string, dogAgeGroup string, dogGender string, catBreed string, catColor string, catAgeGroup string, catGender string) (err error) {
	err = s.userRepo.UpdateUserInfo(uid, firstName, lastName, phoneNumber, address, addressLat, addressLong, dogBreed, dogColor, dogAgeGroup, dogGender, catBreed, catColor, catAgeGroup, catGender)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) UpdateUserImage(ctx context.Context, uid string, file *multipart.FileHeader) (imageURL string, err error) {
	oldImageURL, err := s.userRepo.GetUserImage(uid)
	if err != nil {
		return "", err
	}

	urls, err := s.uploader.UploadImages([]*multipart.FileHeader{file}, "users")
	if err != nil {
		return "", err
	}

	imageURL = urls[0]
	err = s.userRepo.UpdateUserImage(uid, imageURL)
	if err != nil {
		return "", err
	}

	if oldImageURL != nil && *oldImageURL != "" {
		if err := s.uploader.DeleteImage(*oldImageURL); err != nil {
			log.Printf("[UpdateUserImage] failed to delete old image %q: %v", *oldImageURL, err)
		}
	}

	return imageURL, nil
}

func (s *UserServiceImpl) UserInfo(ctx context.Context, uid string) (userData *domain.UserInfo, err error) {
	userInfo, err := s.userRepo.GetUserInfo(uid)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (s *UserServiceImpl) NewUserStatus(ctx context.Context, uid string) (userStatus bool, err error) {
	userInfo, err := s.userRepo.GetNewUserStatus(uid)
	if err != nil {
		return false, err
	}
	return userInfo, nil
}

func (s *UserServiceImpl) UpdateNewUserStatus(ctx context.Context, uid string) (userStatus bool, err error) {
	userInfo, err := s.userRepo.UpdateNewUserStatus(uid)
	if err != nil {
		return false, err
	}
	return userInfo, nil
}
