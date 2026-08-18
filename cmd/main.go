// @title Pet Adoption API
// @version 1.0
// @description API for pet adoption platform
// @host localhost:3000
// @BasePath /

package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	adapter "github.com/S-nudhana/stray2stay/internal/adapter/database"
	httpPetHandler "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	httpUserHandler "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/service"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/database"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/storage"

	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/S-nudhana/stray2stay/docs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	store := sessions.NewCookieStore([]byte(cfg.Session.Secret))
	store.MaxAge(86400 * 1)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = cfg.Session.Secure
	store.Options.SameSite = cfg.Session.SameSite
	gothic.Store = store

	goth.UseProviders(
		google.New(
			cfg.Google.ClientID,
			cfg.Google.ClientSecret,
			cfg.Google.CallbackURL,
			"openid", "email", "profile",
		),
	)

	mysql_db, err := database.NewMySQLDatabase(cfg.DB.MySQL)
	if err != nil {
		log.Fatal("failed connecting to db:", err)
	}
	defer mysql_db.Close()

	mongoClient, err := database.NewMongoDatabase(cfg.DB.Mongo)
	if err != nil {
		log.Fatal(err)
	}
	mongo_db := mongoClient.Database(cfg.DB.Mongo.DBName)

	uploader, err := storage.NewCloudinaryUploader(
		cfg.Cloudinary.CloudName,
		cfg.Cloudinary.APIKey,
		cfg.Cloudinary.APISecret,
	)
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format: "${ip}:${port} ${status} - ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	userRepo := adapter.NewMySQLUserAdapter(mysql_db)
	userService := service.NewUserService(userRepo, uploader)
	userHandler := httpUserHandler.NewHttpUserHandler(userService)

	mysqlPetRepo := adapter.NewMySQLPetAdapter(mysql_db)
	mongoPetRepo := adapter.NewMongoPetAdapter(mongo_db)
	petService := service.NewPetService(mysqlPetRepo, mongoPetRepo, uploader)
	petHandler := httpPetHandler.NewHttpPetHandler(petService)

	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "API is working!"})
	})
	app.Get("/api/swagger/*", fiberSwagger.WrapHandler)

	router.UserRouter(app, userHandler)
	router.PetRouter(app, petHandler)

	log.Printf("Server running at http://localhost%s\n", cfg.Server.Addr)

	if err := app.Listen(cfg.Server.Addr); err != nil {
		log.Fatal("Server stopped:", err)
	}
}