package pet

import (
	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// Adopt godoc
// @Summary Submit adoption request
// @Description Submit an adoption questionnaire form for a specific pet (requires authentication)
// @Tags pets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param adoption body domain.PetAdoptRequest true "Adoption Form Payload"
// @Success 200 {object} domain.PetAdoptResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/adopt [post]
func (h *HttpPetHandler) Adopt(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petAdoptPayload := new(domain.PetAdoptRequest)
	if err := c.BodyParser(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petAdoptPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}


	rid, err := h.service.AdoptPet(c.Context(), uid, petAdoptPayload.Pid, petAdoptPayload.Q1_1, petAdoptPayload.Q1_2, petAdoptPayload.Q1_3, petAdoptPayload.Q2_1, petAdoptPayload.Q2_2, petAdoptPayload.Q2_3, petAdoptPayload.Q3_1, petAdoptPayload.Q3_2, petAdoptPayload.Q3_3, petAdoptPayload.Q4_1, petAdoptPayload.Q5_1, petAdoptPayload.Q6_1, petAdoptPayload.Q6_2, petAdoptPayload.Note)
	if err != nil {
		// Surfaced verbatim rather than a generic message: this is where
		// "pet not available for adoption" and "you already have a pending
		// request for this pet" reach the adopter.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"rid": rid,
	})
}

// SelectAdopter godoc
// @Summary Select an adopter
// @Description Accept a pending adoption request, marking the pet as adopted and denying all other requests
// @Tags pets
// @Accept json
// @Produce json
// @Param adopter body domain.PetSelectAdopterRequest true "Select Adopter Payload"
// @Success 200 {object} domain.PetSelectAdopterResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/adopt/select [patch]
func (h *HttpPetHandler) SelectAdopter(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	petSelectAdopterPayload := new(domain.PetSelectAdopterRequest)
	if err := c.BodyParser(petSelectAdopterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(petSelectAdopterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	if err := h.service.SelectPetAdopter(c.Context(), petSelectAdopterPayload.Rid, uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to select adopter",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adopter selected successfully",
	})
}

func (h *HttpPetHandler) ScreeningAnswerAdoptor(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	screeningAnswerAdoptorPayload := new(domain.ScreeningAnswerAdoptorRequest)
	if err := c.QueryParser(screeningAnswerAdoptorPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(screeningAnswerAdoptorPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	// Call the service method to handle the screening answers
	screeningAnswer, err := h.service.ScreeningAnswerAdoptor(c.Context(), screeningAnswerAdoptorPayload, uid);
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query screening answers",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Screening answers query successfully",
		"screeningAnswer": screeningAnswer,
	})
}

func (h *HttpPetHandler) AllAdoptors(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	adoptors, err := h.service.AllAdoptors(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve adoptors",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Retrieved adoptors successfully",
		"adoptors": adoptors,
	})
}

// MyAdoptionRequests godoc
// @Summary List the authenticated user's own adoption requests
// @Description Every request the caller has made as an adoptor, newest first — backs the Profile page's "My Adoptions" list
// @Tags pets
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.PetMyAdoptionRequestsResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pets/mine/adoptions [get]
func (h *HttpPetHandler) MyAdoptionRequests(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	requests, err := h.service.MyAdoptionRequests(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve your adoption requests",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Retrieved your adoption requests successfully",
		"adoptionRequests": requests,
	})
}

// CancelAdoptionRequest godoc
// @Summary Withdraw a pending adoption request
// @Description Deletes the caller's own request, only while it's still PENDING
// @Tags pets
// @Produce json
// @Security BearerAuth
// @Param rid path string true "Adoption request ID"
// @Success 200 {object} domain.PetCancelAdoptionResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pets/mine/adoptions/{rid} [delete]
func (h *HttpPetHandler) CancelAdoptionRequest(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	rid, err := c.ParamsInt("rid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request id",
		})
	}

	if err := h.service.CancelAdoptionRequest(c.Context(), uid, rid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Adoption request cancelled successfully",
	})
}