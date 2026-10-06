package notification

import (
	"strconv"

	"github.com/S-nudhana/stray2stay/internal/core/service"
	"github.com/gofiber/fiber/v2"
)

type HttpNotificationHandler struct {
	service service.NotificationService
}

func NewHttpNotificationHandler(notificationService service.NotificationService) *HttpNotificationHandler {
	return &HttpNotificationHandler{service: notificationService}
}

func (h *HttpNotificationHandler) List(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)
	notifications, unreadCount, err := h.service.GetNotifications(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"unreadCount":   unreadCount,
	})
}

func (h *HttpNotificationHandler) MarkRead(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)
	notificationID, err := strconv.Atoi(c.Params("nid"))
	if err != nil || notificationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification id"})
	}

	if err := h.service.MarkNotificationRead(c.Context(), uid, notificationID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

func (h *HttpNotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)
	if err := h.service.MarkAllNotificationsRead(c.Context(), uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Notifications marked as read"})
}
