package user

import (
	"github.com/gofiber/fiber/v2"

	"context"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

func (h *HttpUserHandler) UpdateUser(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	userUpdatePayload := new(domain.UserUpdateRequest)
	if err := c.BodyParser(userUpdatePayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}
	err := h.service.UpdateUser(context.Background(), uid, userUpdatePayload.Firstname, userUpdatePayload.Lastname, userUpdatePayload.PhoneNumber, userUpdatePayload.Address, userUpdatePayload.AddressLat, userUpdatePayload.AddressLong, userUpdatePayload.DogBreed, userUpdatePayload.DogColor, userUpdatePayload.DogAgeGroup, userUpdatePayload.DogGender, userUpdatePayload.CatBreed, userUpdatePayload.CatColor, userUpdatePayload.CatAgeGroup, userUpdatePayload.CatGender)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Update user data successfully",
	})
}

func (h *HttpUserHandler) UpdateUserImage(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image file is required",
		})
	}

	imageURL, err := h.service.UpdateUserImage(context.Background(), uid, file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"imageAddress": imageURL,
		"message":      "Update user image successfully",
	})
}

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

func (h *HttpUserHandler) NewUserStatus(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	userData, err := h.service.NewUserStatus(context.Background(), uid)
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

func (h *HttpUserHandler) UpdateNewUserStatus(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	userData, err := h.service.UpdateNewUserStatus(context.Background(), uid)
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
