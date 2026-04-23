package user
	
import (
	"github.com/S-nudhana/stray2stay/internal/core/service"
	"github.com/go-playground/validator/v10"
)

type HttpUserHandler struct {
	service service.UserService
	validate *validator.Validate
}

func NewHttpUserHandler(service service.UserService) *HttpUserHandler {
	return &HttpUserHandler{service: service, validate: validator.New()}
}