package pet

import (
	"github.com/gofiber/fiber/v2"

	"context"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// RegisterPet godoc
// @Summary Register pet
// @Description Register a new pet
// @Tags pets
// @Accept json
// @Produce json
// @Param pet body domain.PetRegisterRequest true "Pet Payload"
// @Success 200 {object} domain.PetRegisterResponse
// @Router /api/pet/register [post]
func (h *HttpPetHandler) Register(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petRegisterPayload := new(domain.PetRegisterRequest)
	if err := c.BodyParser(petRegisterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	pid, err := h.service.RegisterPet(context.Background(), uid, petRegisterPayload.PetName, petRegisterPayload.PetImageAddress, petRegisterPayload.PetAgeGroup, petRegisterPayload.PetGender, petRegisterPayload.PetType, petRegisterPayload.PetBreed, petRegisterPayload.PetColor, petRegisterPayload.PetHealthCondition, petRegisterPayload.PetSterilized, petRegisterPayload.PetVaccination, petRegisterPayload.PetAddress, petRegisterPayload.PetAddressLat, petRegisterPayload.PetAddressLong, petRegisterPayload.Status, petRegisterPayload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Couldn't register the pet",
		})
	}

	return c.JSON(fiber.Map{
		"pid":     pid,
		"message": "Pet register successful",
	})
}