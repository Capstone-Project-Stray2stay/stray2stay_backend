package router

import (
	"github.com/gofiber/fiber/v2"
	"log"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/notification"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

func NotificationRouter(app *fiber.App, notificationHandler *notification.HttpNotificationHandler) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	notifications := app.Group("/api/notifications", middleware.AuthRequired)
	if cfg.Server.IsProduction() {
		notifications = app.Group("/notifications", middleware.AuthRequired)
	}

	notifications.Get("", notificationHandler.List)
	notifications.Put("/read-all", notificationHandler.MarkAllRead)
	notifications.Put("/:nid/read", notificationHandler.MarkRead)
}
