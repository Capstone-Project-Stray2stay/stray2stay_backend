package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c *fiber.Ctx) error {
	cookie := c.Cookies("token")
	if cookie == "" {
		return fiber.ErrUnauthorized
	}

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return fiber.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fiber.ErrUnauthorized
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	c.Locals("uid", uid)
	return c.Next()
}

// OptionalAuth sets "uid" in locals when a valid token cookie is present, but
// never blocks the request — for public routes (like pet search) whose
// results should still adapt for a logged-in caller without requiring login.
func OptionalAuth(c *fiber.Ctx) error {
	cookie := c.Cookies("token")
	if cookie == "" {
		return c.Next()
	}

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.Next()
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Next()
	}

	if uid, ok := claims["uid"].(string); ok {
		c.Locals("uid", uid)
	}
	return c.Next()
}