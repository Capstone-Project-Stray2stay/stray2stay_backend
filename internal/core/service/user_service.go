package service

import (
	"context"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

type UserService interface {
	OAuthLogin(ctx context.Context, email string, provider string, firstName string, lastName string) (uid string, err error)
	Login(ctx context.Context, email string, password string) (uid string, err error)
	Register(ctx context.Context, email string, password string, firstName string, lastName string) (err error)
	DeleteUser(ctx context.Context, uid string) (err error)
	UpdateUser(ctx context.Context, uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64) (err error)
	UserInfo(ctx context.Context, uid string) (userData *domain.UserInfo, err error)
}

type UserServiceImpl struct {
	userRepo port.UserRepository
}

func NewUserService(userRepo port.UserRepository) UserService {
	return &UserServiceImpl{userRepo: userRepo}
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

func (s *UserServiceImpl) UpdateUser(ctx context.Context, uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64) (err error) {
	err = s.userRepo.UpdateUserInfo(uid, firstName, lastName, phoneNumber, address, addressLat, addressLong)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) UserInfo(ctx context.Context, uid string) (userData *domain.UserInfo, err error) {
	userInfo, err := s.userRepo.GetUserInfo(uid)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}
