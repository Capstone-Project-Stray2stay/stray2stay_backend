package pet

import (
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

type HttpPetHandler struct {
	service service.PetService
}

func NewHttpPetHandler(service service.PetService) *HttpPetHandler {
	return &HttpPetHandler{service: service}
}