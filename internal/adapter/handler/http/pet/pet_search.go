package pet

import (
	"context"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// baseUserID strips the ":<provider>" suffix OAuth accounts store on
// user_id/pet_ownerId (see MySQLUserAdapter.OAuthAuthenticateUser), so
// comparing a pet's owner against the JWT's uid claim works regardless of
// which side carries the suffix.
func baseUserID(uid string) string {
	if i := strings.IndexByte(uid, ':'); i != -1 {
		return uid[:i]
	}
	return uid
}

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

	if err := h.validate.Struct(petSearchFilterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	// Set by middleware.OptionalAuth only when a valid session is present —
	// empty for anonymous callers, who get no preference-based fallback.
	uid, _ := c.Locals("uid").(string)

	petData, totalCount, err := h.service.SearchPets(context.Background(), uid, petSearchFilterPayload.Page, petSearchFilterPayload.PageSize, petSearchFilterPayload.PetAgeGroup, petSearchFilterPayload.PetGender, petSearchFilterPayload.PetType, petSearchFilterPayload.PetBreed, petSearchFilterPayload.PetColor, petSearchFilterPayload.PetLocation, petSearchFilterPayload.UserLat, petSearchFilterPayload.UserLong)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get all pets data",
		})
	}

	// Mirrors GetPetsInfo's own default so totalPages lines up with the page
	// size actually used for the query.
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

// PetInfo godoc
// @Summary Get pet info
// @Description Retrieve pet by ID
// @Tags pets
// @Produce json
// @Param pid path string true "Pet ID"
// @Success 200 {object} domain.PetGetInfoByIdResponse
// @Router /api/pet/{pid} [get]
func (h *HttpPetHandler) PetInfo(c *fiber.Ctx) error {
	// Set by middleware.OptionalAuth only when a valid session is present —
	// this route is public, so anonymous callers just get isOwner: false.
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

	// Only worth asking when the viewer isn't the owner — an owner can't
	// adopt their own pet, so there's nothing meaningful to report.
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

// DeletePet godoc
// @Summary Delete pet
// @Description Delete the authenticated user's own pet listing, including its uploaded images
// @Tags pets
// @Produce json
// @Param pid path string true "Pet ID"
// @Success 200 {object} domain.PetDeleteResponse
// @Router /api/pets/{pid} [delete]
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

// MyPets godoc
// @Summary Get the authenticated user's own pets
// @Description List every pet the caller has registered, regardless of status — backs the Profile page's "My Rehoming" list
// @Tags pets
// @Produce json
// @Success 200 {object} domain.PetSearchFilterResponse
// @Router /api/pets/mine [get]
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

// PetRandom godoc
// @Summary Get random pet suggestions
// @Description Get a random selection of available cats and dogs for adoption
// @Tags pets
// @Accept json
// @Produce json
// @Success 200 {object} domain.PetsInfoResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/random [get]
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