package router_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	userhttp "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/port"
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

const integrationJWTSecret = "user-route-integration-test-secret"

type integrationAuthRepository struct {
	port.UserMySQLRepository
	uid      string
	err      error
	calls    int
	email    string
	password string
}

func (r *integrationAuthRepository) AuthenticateUser(email string, password string) (string, error) {
	r.calls++
	r.email = email
	r.password = password
	return r.uid, r.err
}

func newUserRouteIntegrationApp(t *testing.T, repo port.UserMySQLRepository) *fiber.App {
	t.Helper()

	setIntegrationEnv(t)

	userService := service.NewUserService(repo, nil)
	userHandler := userhttp.NewHttpUserHandler(userService)
	app := fiber.New()
	router.UserRouter(app, userHandler)
	return app
}

func TestUserRouterLoginIntegratesHandlerServiceAndRepository(t *testing.T) { // IT-BE-USER-01
	repo := &integrationAuthRepository{uid: "user-123"}
	app := newUserRouteIntegrationApp(t, repo)
	request := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"email":"ada@example.com","password":"password123"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("POST /api/user/login: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, http.StatusOK, body)
	}
	if repo.calls != 1 || repo.email != "ada@example.com" || repo.password != "password123" {
		t.Fatalf("repository received calls=%d email=%q password=%q; want one matching authentication", repo.calls, repo.email, repo.password)
	}

	var responseBody map[string]string
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if responseBody["message"] != "Login successful" {
		t.Fatalf("response message = %q, want %q", responseBody["message"], "Login successful")
	}

	var tokenCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "token" {
			tokenCookie = cookie
			break
		}
	}
	if tokenCookie == nil {
		t.Fatal("successful login did not set the token cookie")
	}
	token, err := jwt.Parse(tokenCookie.Value, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(integrationJWTSecret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("login cookie contains an invalid JWT: token=%v error=%v", token, err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["uid"] != "user-123" {
		t.Fatalf("token uid claim = %v, want user-123", claims["uid"])
	}
}

func TestUserRouterLoginReturnsUnauthorizedWhenRepositoryRejectsCredentials(t *testing.T) { // IT-BE-USER-02
	repo := &integrationAuthRepository{err: errors.New("invalid credentials")}
	app := newUserRouteIntegrationApp(t, repo)
	request := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"email":"ada@example.com","password":"wrong-password"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("POST /api/user/login: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	if repo.calls != 1 || repo.email != "ada@example.com" || repo.password != "wrong-password" {
		t.Fatalf("repository received calls=%d email=%q password=%q; want one matching authentication", repo.calls, repo.email, repo.password)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "token" {
			t.Fatal("failed login must not set a token cookie")
		}
	}
}
