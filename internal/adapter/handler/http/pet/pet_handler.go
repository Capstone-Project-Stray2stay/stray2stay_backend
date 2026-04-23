package pet

import (
	"github.com/S-nudhana/stray2stay/internal/core/service"
	"github.com/go-playground/validator/v10"
)

type HttpPetHandler struct {
	service service.PetService
	validate *validator.Validate
}

func NewHttpPetHandler(service service.PetService) *HttpPetHandler {
	return &HttpPetHandler{service: service, validate: validator.New()}
}