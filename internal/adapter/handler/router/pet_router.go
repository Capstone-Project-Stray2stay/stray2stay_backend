package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
)

func PetRouter(app *fiber.App, petHandler *pet.HttpPetHandler) {
	pet := app.Group("/api/pets")

	pet.Get("/random", petHandler.PetRandom)
	pet.Get("/breeds", petHandler.PetBreeds)
	pet.Get("/breed/color", petHandler.PetColors)
	pet.Get("/breed/behavior", petHandler.PetBehavior)
	pet.Post("/ai/classify", petHandler.AIClassify)
	pet.Get("/:pid", petHandler.PetInfo)
	pet.Get("", petHandler.PetSearchFilter)

	authPet := pet.Group("", middleware.AuthRequired)

	authPet.Post("", petHandler.Register)
	authPet.Post("/:pid/adopt", petHandler.Adopt)
	authPet.Post("/:pid/select-adopter", petHandler.SelectAdopter)
}