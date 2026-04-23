package user

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth/gothic"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// BeginOAuth godoc
// @Summary Start OAuth login
// @Description Redirect user to OAuth provider
// @Tags users
// @Param provider path string true "OAuth Provider"
// @Success 302 {string} string
// @Router /api/user/oauth/{provider} [get]
func (h *HttpUserHandler) BeginOAuth(c *fiber.Ctx) error {
	provider := c.Params("provider")

	return adaptor.HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		q.Set("provider", provider)
		r.URL.RawQuery = q.Encode()

		gothic.BeginAuthHandler(w, r)
	})(c)
}

// OAuthCallback godoc
// @Summary OAuth callback
// @Description Handle OAuth provider callback
// @Tags users
// @Param provider path string true "OAuth Provider"
// @Success 302 {string} string
// @Router /api/user/oauth/{provider}/callback [get]
func (h *HttpUserHandler) OAuthCallback(c *fiber.Ctx) error {
	return adaptor.HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, "OAuth failed", http.StatusUnauthorized)
			return
		}

		uid, err := h.service.OAuthLogin(
			r.Context(),
			user.Email,
			user.Provider,
			user.FirstName,
			user.LastName,
		)
		if err != nil {
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
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, os.Getenv("ORIGIN"), http.StatusFound)
	})(c)
}

// Login godoc
// @Summary Login user
// @Description Login using email and password
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body domain.UserLoginRequest true "Login Payload"
// @Success 200 {object} domain.UserLoginResponse
// @Router /api/user/login [post]
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

// Register godoc
// @Summary Register user
// @Description Create new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body domain.UserRegisterRequest true "Register Payload"
// @Success 200 {object} domain.UserRegisterResponse
// @Router /api/user/register [post]
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

// DeleteUser godoc
// @Summary Delete user
// @Description Delete authenticated user
// @Tags users
// @Success 200 {object} domain.UserDeleteResponse
// @Router /api/user/delete [delete]
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
