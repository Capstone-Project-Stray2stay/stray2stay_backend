package user

import (
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

type HttpUserHandler struct {
	service service.UserService
}

func NewHttpUserHandler(service service.UserService) *HttpUserHandler {
	return &HttpUserHandler{service: service}
}