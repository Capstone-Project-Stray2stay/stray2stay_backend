package pet

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// Adopt godoc
// @Summary Submit adoption request
// @Description Submit an adoption questionnaire form for a specific pet (requires authentication)
// @Tags pets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param adoption body domain.PetAdoptRequest true "Adoption Form Payload"
// @Success 200 {object} domain.PetAdoptResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/adopt [post]
func (h *HttpPetHandler) Adopt(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petAdoptPayload := new(domain.PetAdoptRequest)
	if err := c.BodyParser(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}


	rid, err := h.service.AdoptPet(c.Context(), uid, petAdoptPayload.Pid, petAdoptPayload.Q1_1, petAdoptPayload.Q1_2, petAdoptPayload.Q1_3, petAdoptPayload.Q2_1, petAdoptPayload.Q2_2, petAdoptPayload.Q2_3, petAdoptPayload.Q3_1, petAdoptPayload.Q3_2, petAdoptPayload.Q3_3, petAdoptPayload.Q4_1, petAdoptPayload.Q5_1, petAdoptPayload.Q6_1, petAdoptPayload.Q6_2, petAdoptPayload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to adopt pet",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"rid": rid,
	})
}

// SelectAdopter godoc
// @Summary Select an adopter
// @Description Accept a pending adoption request, marking the pet as adopted and denying all other requests
// @Tags pets
// @Accept json
// @Produce json
// @Param adopter body domain.PetSelectAdopterRequest true "Select Adopter Payload"
// @Success 200 {object} domain.PetSelectAdopterResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/adopt/select [patch]
func (h *HttpPetHandler) SelectAdopter(c *fiber.Ctx) error {
	petSelectAdopterPayload := new(domain.PetSelectAdopterRequest)
	if err := c.BodyParser(petSelectAdopterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petSelectAdopterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	if err := h.service.SelectPetAdopter(c.Context(), petSelectAdopterPayload.Rid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to select adopter",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adopter selected successfully",
	})
}

