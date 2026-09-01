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

	if err := h.validate.Struct(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	rid, err := h.service.AdoptPet(c.Context(), uid, petAdoptPayload.Pid, petAdoptPayload.Q1_1, petAdoptPayload.Q1_2, petAdoptPayload.Q1_3, petAdoptPayload.Q2_1, petAdoptPayload.Q2_2, petAdoptPayload.Q2_3, petAdoptPayload.Q3_1, petAdoptPayload.Q3_2, petAdoptPayload.Q3_3, petAdoptPayload.Q4_1, petAdoptPayload.Q5_1, petAdoptPayload.Q6_1, petAdoptPayload.Q6_2, petAdoptPayload.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"rid": rid,
	})
}

func (h *HttpPetHandler) SelectAdopter(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

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

	if err := h.service.SelectPetAdopter(c.Context(), petSelectAdopterPayload.Rid, uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to select adopter",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adopter selected successfully",
	})
}

func (h *HttpPetHandler) ScreeningAnswerAdoptor(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	screeningAnswerAdoptorPayload := new(domain.ScreeningAnswerAdoptorRequest)
	if err := c.QueryParser(screeningAnswerAdoptorPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(screeningAnswerAdoptorPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	screeningAnswer, err := h.service.ScreeningAnswerAdoptor(c.Context(), screeningAnswerAdoptorPayload, uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query screening answers",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "Screening answers query successfully",
		"screeningAnswer": screeningAnswer,
	})
}

func (h *HttpPetHandler) AllAdoptors(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	adoptors, err := h.service.AllAdoptors(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve adoptors",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Retrieved adoptors successfully",
		"adoptors": adoptors,
	})
}

func (h *HttpPetHandler) MyAdoptionRequests(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	requests, err := h.service.MyAdoptionRequests(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve your adoption requests",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Retrieved your adoption requests successfully",
		"adoptionRequests": requests,
	})
}

func (h *HttpPetHandler) CancelAdoptionRequest(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	rid, err := c.ParamsInt("rid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request id",
		})
	}

	if err := h.service.CancelAdoptionRequest(c.Context(), uid, rid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adoption request cancelled successfully",
	})
}
