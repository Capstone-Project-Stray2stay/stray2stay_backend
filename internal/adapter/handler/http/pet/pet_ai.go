package pet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
	"strings"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

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
	detectedBreed := aiResp.PetBreed
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"pet_breed": detectedBreed,
	})
}
