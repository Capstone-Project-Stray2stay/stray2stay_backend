package user

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

func (h *HttpUserHandler) BeginOAuth(c *fiber.Ctx) error {
	provider := c.Params("provider")

	return adaptor.HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		q.Set("provider", provider)
		r.URL.RawQuery = q.Encode()

		gothic.BeginAuthHandler(w, r)
	})(c)
}

func (h *HttpUserHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")

	return adaptor.HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		q.Set("provider", provider)
		r.URL.RawQuery = q.Encode()

		gothUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			log.Printf("[OAuthCallback] CompleteUserAuth error: %v", err)
			http.Error(w, "OAuth failed", http.StatusUnauthorized)
			return
		}

		firstName, lastName := gothUser.FirstName, gothUser.LastName
		if firstName == "" && lastName == "" && gothUser.Name != "" {
			parts := strings.SplitN(gothUser.Name, " ", 2)
			firstName = parts[0]
			if len(parts) > 1 {
				lastName = parts[1]
			}
		}

		uid, err := h.service.OAuthLogin(
			r.Context(),
			gothUser.Email,
			"OAUTH",
			firstName,
			lastName,
		)
		if err != nil {
			log.Printf("[OAuthCallback] OAuthLogin error: %v", err)
			http.Error(w, "OAuth failed", http.StatusUnauthorized)
			return
		}

		claims := jwt.MapClaims{
			"uid": uid,
			"exp": time.Now().Add(72 * time.Hour).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			log.Printf("[OAuthCallback] JWT signing error: %v", err)
			http.Error(w, "Token error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    signed,
			Expires:  time.Now().Add(72 * time.Hour),
			HttpOnly: true,
			Secure:   os.Getenv("ENV") == "production",
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, os.Getenv("ORIGIN"), http.StatusFound)
	})(c)
}

func (h *HttpUserHandler) Login(c *fiber.Ctx) error {
	userLoginPayload := new(domain.UserLoginRequest)
	if err := c.BodyParser(userLoginPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(userLoginPayload); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}
	uid, err := h.service.Login(context.Background(), userLoginPayload.Email, userLoginPayload.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	claims := jwt.MapClaims{
		"uid": uid,
		"exp": time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    signedToken,
		Expires:  time.Now().Add(72 * time.Hour),
		HTTPOnly: true,
		SameSite: "strict",
		Secure:   os.Getenv("ENV") == "production",
	})

	return c.JSON(fiber.Map{
		"message": "Login successful",
	})
}

func (h *HttpUserHandler) Register(c *fiber.Ctx) error {
	userRegisterPayload := new(domain.UserRegisterRequest)
	if err := c.BodyParser(userRegisterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request payload",
		})
	}

	if err := h.validate.Struct(userRegisterPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect request format",
		})
	}

	err := h.service.Register(context.Background(), userRegisterPayload.Email, userRegisterPayload.Password, userRegisterPayload.Firstname, userRegisterPayload.Lastname)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Registration successful",
	})
}

func (h *HttpUserHandler) DeleteUser(c *fiber.Ctx) error {
	uid := c.Locals("uid").(string)

	err := h.service.DeleteUser(context.Background(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

func (h *HttpUserHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		SameSite: "Strict",
		Secure:   os.Getenv("ENV") == "production",
	})
	return c.JSON(fiber.Map{"message": "Logged out"})
}

func (h *HttpUserHandler) Authorize(c *fiber.Ctx) error {
	cookie := c.Cookies("token")
	if cookie == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"authorized":     false,
			"userFirstname":  "",
			"userCoverImage": "",
		})
	}

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"authorized":     false,
			"userFirstname":  "",
			"userCoverImage": "",
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"authorized":     false,
			"userFirstname":  "",
			"userCoverImage": "",
		})
	}

	uid, ok := claims["uid"].(string)
	if !ok || uid == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"authorized":     false,
			"userFirstname":  "",
			"userCoverImage": "",
		})
	}

	userInfo, err := h.service.UserInfo(context.Background(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"authorized":     true,
		"userFirstname":  userInfo.Firstname,
		"userCoverImage": userInfo.CoverImage,
	})
}
