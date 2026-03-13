package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
)

func UserRouter(app *fiber.App, userHandler *user.HttpUserHandler) {
	user := app.Group("/api/user")

	user.Post("/login", userHandler.Login)
	user.Post("/register", userHandler.Register)
	user.Get("/oauth/:provider", userHandler.BeginOAuth)
	user.Get("/oauth/:provider/callback", userHandler.OAuthCallback)

	authUser := user.Group("", middleware.AuthRequired)
	authUser.Delete("/delete", userHandler.DeleteUser)
	authUser.Put("/update", userHandler.UpdateUser)
	authUser.Get("/info", userHandler.UserInfo)
}
