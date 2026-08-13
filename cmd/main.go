// @title Pet Adoption API
// @version 1.0
// @description API for pet adoption platform
// @host localhost:3000
// @BasePath /

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	adapter "github.com/S-nudhana/stray2stay/internal/adapter/database"
	httpPetHandler "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	httpUserHandler "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/service"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/database"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/storage"

	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/S-nudhana/stray2stay/docs"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	store.MaxAge(86400 * 1)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = os.Getenv("ENV") == "production"
	store.Options.SameSite = http.SameSiteLaxMode
	gothic.Store = store

	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_CALLBACK_URL"),
			"openid", "email", "profile",
		),
	)
	mysql_db, err := database.NewMySQLDatabase()
	if err != nil {
		log.Fatal("failed connecting to db:", err)
	}
	defer mysql_db.Close()

	mongoClient, err := database.NewMongoDatabase()
	if err != nil {
		log.Fatal(err)
	}
	mongo_db := mongoClient.Database(os.Getenv("MONGO_DB_NAME"))

	uploader, err := storage.NewCloudinaryUploader(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(logger.New(logger.Config{
		Format: "${ip}:${port} ${status} - ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("ORIGIN"),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Authorization",
		AllowCredentials: true,
		MaxAge:           300,
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

	addr := ":3000"
	log.Printf("Server running at http://localhost%s\n", addr)

	if err := app.Listen(addr); err != nil {
		log.Fatal("Server stopped:", err)
	}
}
