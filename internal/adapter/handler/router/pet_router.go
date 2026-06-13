package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
)

func PetRouter(app *fiber.App, petHandler *pet.HttpPetHandler) {
	pet := app.Group("/api/pets")

	pet.Get("", petHandler.PetSearchFilter)
	pet.Get("/:pid", petHandler.PetInfo)
	pet.Get("/random", petHandler.PetRandom)

	authPet := pet.Group("", middleware.AuthRequired)

	authPet.Post("", petHandler.Register)
	authPet.Post("/ai/classify", petHandler.AIClassify)
	authPet.Post("/:pid/adopt", petHandler.Adopt)
	authPet.Post("/:pid/select-adopter", petHandler.SelectAdopter)
}