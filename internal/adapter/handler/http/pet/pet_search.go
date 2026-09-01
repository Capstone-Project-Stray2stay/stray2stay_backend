package pet

import (
	"context"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

func baseUserID(uid string) string {
	if i := strings.IndexByte(uid, ':'); i != -1 {
		return uid[:i]
	}
	return uid
}

func (h *HttpPetHandler) PetSearchFilter(c *fiber.Ctx) error {
	petSearchFilterPayload := new(domain.PetSearchFilterRequest)
	if err := c.QueryParser(petSearchFilterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if err := h.validate.Struct(petSearchFilterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	uid, _ := c.Locals("uid").(string)

	petData, totalCount, err := h.service.SearchPets(context.Background(), uid, petSearchFilterPayload.Page, petSearchFilterPayload.PageSize, petSearchFilterPayload.PetAgeGroup, petSearchFilterPayload.PetGender, petSearchFilterPayload.PetType, petSearchFilterPayload.PetBreed, petSearchFilterPayload.PetColor, petSearchFilterPayload.PetLocation, petSearchFilterPayload.UserLat, petSearchFilterPayload.UserLong)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get all pets data",
		})
	}

	pageSize := petSearchFilterPayload.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return c.JSON(fiber.Map{
		"petsInfo":   petData,
		"totalCount": totalCount,
		"totalPages": totalPages,
		"message":    "Get all pets data successfully",
	})
}

func (h *HttpPetHandler) PetInfo(c *fiber.Ctx) error {
	uid, _ := c.Locals("uid").(string)

	petGetInfoByIdPayload := new(domain.PetGetInfoByIdRequest)
	if err := c.ParamsParser(petGetInfoByIdPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petGetInfoByIdPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	petData, err := h.service.PetInfo(context.Background(), petGetInfoByIdPayload.Pid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get pet data",
		})
	}

	isOwner := uid != "" && petData != nil && baseUserID(petData.PetOwnerID) == baseUserID(uid)

	adoptionStatus := ""
	if uid != "" && !isOwner {
		adoptionStatus, _ = h.service.MyAdoptionStatus(context.Background(), uid, petGetInfoByIdPayload.Pid)
	}

	return c.JSON(fiber.Map{
		"petsInfo":       petData,
		"isOwner":        isOwner,
		"adoptionStatus": adoptionStatus,
		"message":        "Get pet data successfully",
	})
}

func (h *HttpPetHandler) DeletePet(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petGetInfoByIdPayload := new(domain.PetGetInfoByIdRequest)
	if err := c.ParamsParser(petGetInfoByIdPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petGetInfoByIdPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	if err := h.service.DeletePet(context.Background(), uid, petGetInfoByIdPayload.Pid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Pet deleted successfully",
	})
}

func (h *HttpPetHandler) MyPets(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petData, err := h.service.MyPets(context.Background(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get your pets",
		})
	}

	return c.JSON(fiber.Map{
		"petsInfo": petData,
		"message":  "Get your pets successfully",
	})
}

func (h *HttpPetHandler) PetRandom(c *fiber.Ctx) error {
	petData, err := h.service.PetRandom(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get random pet data",
		})
	}

	return c.JSON(fiber.Map{
		"petsInfo": petData,
		"message":  "Get random pet data successfully",
	})
}
