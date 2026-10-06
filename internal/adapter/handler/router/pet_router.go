package router

import (
	"github.com/gofiber/fiber/v2"
	"log"

	"github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	"github.com/S-nudhana/stray2stay/internal/adapter/middleware"
	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

func PetRouter(app *fiber.App, petHandler *pet.HttpPetHandler) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	pet := app.Group("/api/pets")
	if cfg.Server.IsProduction() {
		pet = app.Group("/pets")
	}

	pet.Get("/random", petHandler.PetRandom)
	pet.Get("/breeds", petHandler.PetBreeds)
	pet.Get("/breeds/images", petHandler.PetBreedImages)
	pet.Get("/breed/color", petHandler.PetColors)
	pet.Get("/breed/behavior", petHandler.PetBehavior)
	pet.Get("/mine", middleware.AuthRequired, petHandler.MyPets)
	pet.Get("/mine/adoptions", middleware.AuthRequired, petHandler.MyAdoptionRequests)
<<<<<<< HEAD
	pet.Get("/mine/adoptors", middleware.AuthRequired, petHandler.AllAdoptors)
=======
	// Registered before "/:pid" so "mine" is never parsed as a pet id.
	pet.Get("/mine/diaries", middleware.AuthRequired, petHandler.MyDiaryPets)
>>>>>>> origin/main
	pet.Delete("/mine/adoptions/:rid", middleware.AuthRequired, petHandler.CancelAdoptionRequest)
	pet.Get("/:pid", middleware.OptionalAuth, petHandler.PetInfo)
	pet.Get("", middleware.OptionalAuth, petHandler.PetSearchFilter)
	pet.Post("/ai/classify", petHandler.AIClassify)
	pet.Get("/:pid/screening-questions", middleware.OptionalAuth, petHandler.GetScreeningQuestions)

	authPet := pet.Group("", middleware.AuthRequired)

<<<<<<< HEAD
=======
	authPet.Get("/:pid/adoptors", petHandler.AllAdoptors)
>>>>>>> origin/main
	authPet.Get("/:pid/screening-answer", petHandler.ScreeningAnswerAdoptor)
	authPet.Put("/:pid/screening-questions", petHandler.SaveScreeningQuestions)
	authPet.Post("/:pid/screening-answer-image", petHandler.UploadScreeningAnswerImage)

<<<<<<< HEAD
=======
	authPet.Get("/:pid/diary", petHandler.DiaryEntries)
	authPet.Put("/:pid/diary/:date", petHandler.SaveDiaryEntry)
	authPet.Delete("/:pid/diary/:date", petHandler.DeleteDiaryEntry)

>>>>>>> origin/main
	authPet.Post("/:pid/select-adopter", petHandler.SelectAdopter)
	authPet.Post("/:pid/adopt", petHandler.Adopt)
	authPet.Post("", petHandler.Register)
	authPet.Delete("/:pid", petHandler.DeletePet)
	authPet.Put("/:pid", petHandler.UpdatePet)
}
