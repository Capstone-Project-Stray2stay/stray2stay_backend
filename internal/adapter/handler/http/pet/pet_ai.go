package pet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// AIClassify godoc
// @Summary Classify pet breed using AI
// @Description Upload 1–2 pet images to detect the breed via an external AI service
// @Tags pets
// @Accept multipart/form-data
// @Produce json
// @Param type query string true "Pet type (e.g. CAT, DOG)"
// @Param images formData file true "Pet image(s) — 1 or 2 files"
// @Success 200 {object} domain.PetAIClassifyResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/pet/ai/classify [post]
func (h *HttpPetHandler) AIClassify(c *fiber.Ctx) error {
	petAIClassifyPayload := new(domain.PetAIClassifyRequest)
	if err := c.QueryParser(petAIClassifyPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	if err := h.validate.Struct(petAIClassifyPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File["images"]
	if len(files) == 0 || len(files) > 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "1 or 2 images required",
		})
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return err
		}

		part, err := writer.CreateFormFile("files", fileHeader.Filename)
		if err != nil {
			file.Close()
			return err
		}

		_, err = io.Copy(part, file)
		file.Close()

		if err != nil {
			return err
		}
	}
	writer.Close()
	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	url := fmt.Sprintf(
		"%s/api/%s/classify",
		aiServiceURL,
		strings.ToLower(petAIClassifyPayload.Type),
	)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": "AI service error",
		})
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var aiResp domain.PetAIClassifyResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return err
	}

	if len(aiResp.Predictions) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No predictions returned",
		})
	}

	detectedBreed := aiResp.Predictions[0].Label
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"pet_breed": detectedBreed,
	})
}
