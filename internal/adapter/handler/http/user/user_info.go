package user

import (
	"github.com/gofiber/fiber/v2"

	"context"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// UpdateUser godoc
// @Summary Update user
// @Description Update authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Param user body domain.UserUpdateRequest true "Update Payload"
// @Success 200 {object} domain.UserUpdateResponse
// @Router /api/user/update [put]
func (h *HttpUserHandler) UpdateUser(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	userUpdatePayload := new(domain.UserUpdateRequest)
	if err := c.BodyParser(userUpdatePayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}
	AddressLat := 1.0
	AddressLong := 1.0
	err := h.service.UpdateUser(context.Background(), uid, userUpdatePayload.Firstname, userUpdatePayload.Lastname, userUpdatePayload.PhoneNumber, userUpdatePayload.Address, AddressLat, AddressLong)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Update user data successfully",
	})
}

// UserInfo godoc
// @Summary Get user info
// @Description Get authenticated user profile
// @Tags users
// @Produce json
// @Success 200 {object} domain.UserInfoResponse
// @Router /api/user/info [get]
func (h *HttpUserHandler) UserInfo(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	userData, err := h.service.UserInfo(context.Background(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error to get user info",
		})
	}

	return c.JSON(fiber.Map{
		"userData": userData,
		"message":  "Get user info successfully",
	})
}