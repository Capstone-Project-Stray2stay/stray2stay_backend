package pet

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// PetsInfo godoc
// @Summary Get all pets
// @Description Retrieve all pets
// @Tags pets
// @Produce json
// @Success 200 {object} domain.PetSearchFilterResponse
// @Router /api/pet/all [get]
func (h *HttpPetHandler) PetSearchFilter(c *fiber.Ctx) error {
	petSearchFilterPayload := new(domain.PetSearchFilterRequest)
	if err := c.QueryParser(petSearchFilterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}
	petData, err := h.service.SearchPets(context.Background(), petSearchFilterPayload.Page, petSearchFilterPayload.PageSize, petSearchFilterPayload.PetAgeGroup, petSearchFilterPayload.PetGender, petSearchFilterPayload.PetType, petSearchFilterPayload.PetBreed, petSearchFilterPayload.PetColor, petSearchFilterPayload.UserLat, petSearchFilterPayload.UserLong)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get all pets data",
		})
	}

	return c.JSON(fiber.Map{
		"petsInfo": petData,
		"message":  "Get all pets data successfully",
	})
}

// PetInfo godoc
// @Summary Get pet info
// @Description Retrieve pet by ID
// @Tags pets
// @Produce json
// @Param pid path string true "Pet ID"
// @Success 200 {object} domain.PetGetInfoByIdResponse
// @Router /api/pet/{pid} [get]
func (h *HttpPetHandler) PetInfo(c *fiber.Ctx) error {
	petGetInfoByIdPayload := new(domain.PetGetInfoByIdRequest)
	if err := c.ParamsParser(petGetInfoByIdPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	petData, err := h.service.PetInfo(context.Background(), petGetInfoByIdPayload.Pid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get pet data",
		})
	}

	return c.JSON(fiber.Map{
		"petsInfo": petData,
		"message":  "Get pet data successfully",
	})
}