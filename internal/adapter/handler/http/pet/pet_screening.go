package pet

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

func (h *HttpPetHandler) GetScreeningQuestions(c *fiber.Ctx) error {
	pid, err := c.ParamsInt("pid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pet id",
		})
	}

	questions, locked, err := h.service.GetScreeningQuestions(c.Context(), pid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve screening questions",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Retrieved screening questions successfully",
		"questions": questions,
		"locked":    locked,
	})
}

func (h *HttpPetHandler) SaveScreeningQuestions(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	pid, err := c.ParamsInt("pid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pet id",
		})
	}

	payload := new(domain.SaveScreeningQuestionsRequest)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	if err := h.service.SaveScreeningQuestions(c.Context(), uid, pid, payload.Questions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Screening questions saved successfully",
	})
}

func (h *HttpPetHandler) UploadScreeningAnswerImage(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image file is required",
		})
	}

	imageURL, err := h.service.UploadScreeningAnswerImage(c.Context(), uid, file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload image",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"imageUrl": imageURL,
		"message":  "Image uploaded successfully",
	})
}
