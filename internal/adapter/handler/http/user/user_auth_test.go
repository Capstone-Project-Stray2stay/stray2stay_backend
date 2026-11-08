package user

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type authServiceStub struct {
	loginUID      string
	loginErr      error
	loginCalls    int
	loginEmail    string
	loginPassword string

	registerErr      error
	registerCalls    int
	registerEmail    string
	registerPassword string
	registerFirst    string
	registerLast     string
}

func (s *authServiceStub) OAuthLogin(context.Context, string, string, string, string) (string, error) {
	return "", nil
}

func (s *authServiceStub) Login(_ context.Context, email string, password string) (string, error) {
	s.loginCalls++
	s.loginEmail, s.loginPassword = email, password
	return s.loginUID, s.loginErr
}

func (s *authServiceStub) Register(_ context.Context, email string, password string, firstName string, lastName string) error {
	s.registerCalls++
	s.registerEmail, s.registerPassword = email, password
	s.registerFirst, s.registerLast = firstName, lastName
	return s.registerErr
}

func (*authServiceStub) DeleteUser(context.Context, string) error { return nil }

func (*authServiceStub) UpdateUser(context.Context, string, string, string, string, string, float64, float64, []domain.UserPreference) error {
	return nil
}

func (*authServiceStub) UpdateUserImage(context.Context, string, *multipart.FileHeader) (string, error) {
	return "", nil
}

func (*authServiceStub) UserInfo(context.Context, string) (*domain.UserInfo, error) {
	return nil, nil
}

func (*authServiceStub) NewUserStatus(context.Context, string) (bool, error) { return false, nil }

func (*authServiceStub) UpdateNewUserStatus(context.Context, string) (bool, error) { return false, nil }

func newAuthTestApp(t *testing.T, stub *authServiceStub) *fiber.App {
	t.Helper()
	t.Setenv("JWT_SECRET", "auth-handler-test-secret")
	t.Setenv("ENV", "test")

	app := fiber.New()
	handler := NewHttpUserHandler(stub)
	app.Post("/login", handler.Login)
	app.Post("/register", handler.Register)
	return app
}

func postJSON(t *testing.T, app *fiber.App, path string, body string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func TestLoginValidatesRequestBeforeCallingService(t *testing.T) { // UT-BE-AUTH-01
	for _, body := range []string{
		`{"email":"bad-email","password":"password123"}`,
		`{"email":"adopter@example.com","password":"short"}`,
		`{"email":`,
	} {
		t.Run(body, func(t *testing.T) {
			stub := &authServiceStub{}
			response := postJSON(t, newAuthTestApp(t, stub), "/login", body)
			defer response.Body.Close()
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
			}
			if stub.loginCalls != 0 {
				t.Fatalf("login service called %d times for invalid request", stub.loginCalls)
			}
		})
	}
}

func TestLoginReturnsSignedHttpOnlyCookie(t *testing.T) { // UT-BE-AUTH-02
	stub := &authServiceStub{loginUID: "user-123"}
	response := postJSON(t, newAuthTestApp(t, stub), "/login", `{"email":"adopter@example.com","password":"password123"}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if stub.loginCalls != 1 || stub.loginEmail != "adopter@example.com" || stub.loginPassword != "password123" {
		t.Fatalf("login service received unexpected request: calls=%d email=%q", stub.loginCalls, stub.loginEmail)
	}

	var cookie *http.Cookie
	for _, candidate := range response.Cookies() {
		if candidate.Name == "token" {
			cookie = candidate
			break
		}
	}
	if cookie == nil {
		t.Fatal("login response did not set the token cookie")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("token cookie security flags = HttpOnly:%t SameSite:%v", cookie.HttpOnly, cookie.SameSite)
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte("auth-handler-test-secret"), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("token is invalid: token=%v error=%v", token, err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["uid"] != "user-123" {
		t.Fatalf("token uid claim = %v, want user-123", claims["uid"])
	}
}

func TestLoginReturnsUnauthorizedWhenCredentialsFail(t *testing.T) { // UT-BE-AUTH-03
	stub := &authServiceStub{loginErr: errors.New("invalid credentials")}
	response := postJSON(t, newAuthTestApp(t, stub), "/login", `{"email":"adopter@example.com","password":"password123"}`)
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	if stub.loginCalls != 1 {
		t.Fatalf("login service calls = %d, want 1", stub.loginCalls)
	}
}

func TestRegisterValidatesAndForwardsRequest(t *testing.T) { // UT-BE-REG-01
	t.Run("valid registration", func(t *testing.T) {
		stub := &authServiceStub{}
		response := postJSON(t, newAuthTestApp(t, stub), "/register", `{"email":"ada@example.com","password":"password123","firstName":"Ada","lastName":"Lovelace"}`)
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
		if stub.registerCalls != 1 || stub.registerEmail != "ada@example.com" || stub.registerPassword != "password123" || stub.registerFirst != "Ada" || stub.registerLast != "Lovelace" {
			t.Fatalf("registration service received unexpected data: calls=%d email=%q first=%q last=%q", stub.registerCalls, stub.registerEmail, stub.registerFirst, stub.registerLast)
		}
	})

	t.Run("invalid registration", func(t *testing.T) {
		stub := &authServiceStub{}
		response := postJSON(t, newAuthTestApp(t, stub), "/register", `{"email":"bad-email","password":"short","firstName":"","lastName":"Lovelace"}`)
		defer response.Body.Close()

		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
		}
		if stub.registerCalls != 0 {
			t.Fatalf("registration service called %d times for invalid request", stub.registerCalls)
		}
	})
}
