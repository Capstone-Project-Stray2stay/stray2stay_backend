package pet

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

func (h *HttpPetHandler) Adopt(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petAdoptPayload := new(domain.PetAdoptRequest)
	if err := c.BodyParser(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	rid, err := h.service.AdoptPet(c.Context(), uid, petAdoptPayload.Pid, petAdoptPayload.Contact)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to adopt pet",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"rid": rid,
	})
}

func (h *HttpPetHandler) SelectAdopter(c *fiber.Ctx) error {
	petSelectAdopterPayload := new(domain.PetSelectAdopterRequest)
	if err := c.BodyParser(petSelectAdopterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	err := h.service.SelectPetAdopter(c.Context(), petSelectAdopterPayload.Rid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to select adopter",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adopter selected successfully",
	})
}