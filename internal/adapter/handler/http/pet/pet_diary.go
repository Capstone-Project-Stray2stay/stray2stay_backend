package pet

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// MyDiaryPets lists every pet whose diary the caller may open, with the
// contact card and write permission already resolved per pet.
func (h *HttpPetHandler) MyDiaryPets(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	pets, err := h.service.MyDiaryPets(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve your diaries",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Retrieved your diaries successfully",
		"diaryPets": pets,
	})
}

// DiaryEntries returns one pet's entries between the `from` and `to` query
// dates, inclusive — the page asks for the visible month.
func (h *HttpPetHandler) DiaryEntries(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	pid, err := c.ParamsInt("pid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pet id",
		})
	}

	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "from and to are required",
		})
	}

	entries, role, since, err := h.service.DiaryEntries(c.Context(), uid, pid, from, to)
	if err != nil {
		return c.Status(diaryErrorStatus(err)).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Retrieved diary entries successfully",
		"entries":  entries,
		"role":     role,
		"since":    since,
		"canWrite": role == "ADOPTER",
	})
}

// SaveDiaryEntry upserts the day named in the path. The multipart body carries
// an optional `image` file and a `caption`; omitting the file keeps whatever
// photo the day already has.
func (h *HttpPetHandler) SaveDiaryEntry(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	pid, err := c.ParamsInt("pid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pet id",
		})
	}

	// FormFile reports an error both when no file was sent and when the part
	// is unreadable; treating either as "no new photo" is right here, because
	// the service rejects a missing photo on a brand-new entry anyway.
	file, _ := c.FormFile("image")

	entry, err := h.service.SaveDiaryEntry(c.Context(), uid, pid, c.Params("date"), file, c.FormValue("caption"))
	if err != nil {
		return c.Status(diaryErrorStatus(err)).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Diary entry saved successfully",
		"entry":   entry,
	})
}

func (h *HttpPetHandler) DeleteDiaryEntry(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	pid, err := c.ParamsInt("pid")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pet id",
		})
	}

	if err := h.service.DeleteDiaryEntry(c.Context(), uid, pid, c.Params("date")); err != nil {
		return c.Status(diaryErrorStatus(err)).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Diary entry deleted successfully",
	})
}

// diaryErrorStatus maps the service's errors onto status codes so the frontend
// can tell "you may not" from "that was malformed" from "we broke". The
// service returns plain errors, so this matches on their text.
func diaryErrorStatus(err error) int {
	message := err.Error()
	switch {
	case strings.Contains(message, "no diary access"),
		strings.Contains(message, "only the adopter"):
		return fiber.StatusForbidden
	case strings.Contains(message, "not found"):
		return fiber.StatusNotFound
	case strings.Contains(message, "cannot write"),
		strings.Contains(message, "required"),
		strings.Contains(message, "must be formatted"),
		strings.Contains(message, "range ends"):
		return fiber.StatusBadRequest
	default:
		return fiber.StatusInternalServerError
	}
}
