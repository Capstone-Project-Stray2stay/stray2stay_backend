package pet

import (
	"github.com/gofiber/fiber/v2"

	"context"
	"log"

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
	payload := new(domain.PetRegisterRequest)
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

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}
	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one image is required",
		})
	}

	uid := c.Locals("uid").(string)

	pid, err := h.service.RegisterPet(context.Background(), uid, payload.PetName, files, payload.PetAgeGroup, payload.PetGender, payload.PetType, payload.PetBreed, payload.PetColor, payload.PetPersonality, payload.PetSpecialCare, payload.PetSterilized, payload.PetVaccination, payload.PetAddress, payload.PetAddressLat, payload.PetAddressLong, payload.Status, payload.Note)
	if err != nil {
		// The underlying cause (Cloudinary upload vs MySQL insert) is the only
		// thing that makes this actionable, so log it instead of dropping it.
		log.Printf("register pet failed for uid %s: %v", uid, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to register pet",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"petId":   pid,
		"message": "Pet registered successfully",
	})
}

func (h *HttpPetHandler) PetBreeds(c *fiber.Ctx) error {
	PetGetBreedsRequest := new(domain.PetGetBreedsRequest)
	if err := c.QueryParser(PetGetBreedsRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}
	if err := h.validate.Struct(PetGetBreedsRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	breedData, err := h.service.AllBreeds(context.Background(), PetGetBreedsRequest.PetType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err,
		})
	}

	return c.JSON(fiber.Map{
		"breedData": breedData,
		"message":   "Get breed data successfully",
	})
}

func (h *HttpPetHandler) PetBehavior(c *fiber.Ctx) error {
	PetGetBehaviorRequest := new(domain.PetGetBehaviorRequest)
	if err := c.QueryParser(PetGetBehaviorRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}
	if err := h.validate.Struct(PetGetBehaviorRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}
	
	behaviorData, err := h.service.PetBehavior(context.Background(), PetGetBehaviorRequest.PetType, PetGetBehaviorRequest.PetBreed)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err,
		})
	}

	return c.JSON(fiber.Map{
		"behaviorData": behaviorData,
		"message":      "Get breed behavior data successfully",
	})
}

func (h *HttpPetHandler) PetColors(c *fiber.Ctx) error {
	PetGetBehaviorRequest := new(domain.PetGetBehaviorRequest)
	if err := c.QueryParser(PetGetBehaviorRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}
	if err := h.validate.Struct(PetGetBehaviorRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	colorData, err := h.service.PetColor(context.Background(), PetGetBehaviorRequest.PetType, PetGetBehaviorRequest.PetBreed)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err,
		})
	}

	return c.JSON(fiber.Map{
		"colorData": colorData,
		"message":   "Get breed color data successfully",
	})
}