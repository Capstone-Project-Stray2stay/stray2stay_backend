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