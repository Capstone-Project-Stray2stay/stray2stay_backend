package router

import (
	"github.com/gofiber/fiber/v2"
	"log"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

func UserRouter(app *fiber.App, userHandler *user.HttpUserHandler) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	user := app.Group("/api/user")
	if cfg.Server.IsProduction() {
		user = app.Group("/user")
	}

	user.Post("/login", userHandler.Login)
	user.Post("/register", userHandler.Register)
	user.Get("/oauth/:provider", userHandler.BeginOAuth)
	user.Get("/oauth/:provider/callback", userHandler.OAuthCallback)
	user.Get("/authorize", userHandler.Authorize)
<<<<<<< HEAD

=======
	
>>>>>>> origin/main
	authUser := user.Group("", middleware.AuthRequired)
	authUser.Post("/logout", userHandler.Logout)
	authUser.Get("/status", userHandler.NewUserStatus)
	authUser.Put("/status", userHandler.UpdateNewUserStatus)
	authUser.Delete("/delete", userHandler.DeleteUser)
	authUser.Put("/update", userHandler.UpdateUser)
	authUser.Put("/image", userHandler.UpdateUserImage)
	authUser.Get("/info", userHandler.UserInfo)
}
