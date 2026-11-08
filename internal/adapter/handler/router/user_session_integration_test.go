package router_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"

	userhttp "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/user"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

// Covers everything on UserRouter beyond the login path that
// user_router_integration_test.go already pins down: registration, the session probe
// the frontend polls on every page, the new-user flag, logout, and the guard that
// separates the public half of the group from the private half.
func newUserSessionIntegrationApp(t *testing.T, repo *fakeUserRepo, uploader *fakeUploader) *fiber.App {
	t.Helper()

	setIntegrationEnv(t)

	userService := service.NewUserService(repo, uploader)
	userHandler := userhttp.NewHttpUserHandler(userService)
	app := fiber.New()
	router.UserRouter(app, userHandler)
	return app
}

func coverImage(url string) *string { return &url }

func TestUserRouterRegisterForwardsTheAccountToTheRepository(t *testing.T) { // IT-BE-SESS-01
	repo := &fakeUserRepo{}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	body := `{"email":"ada@example.com","password":"password123","firstName":"Ada","lastName":"Lovelace"}`
	response := doRequest(t, app, http.MethodPost, "/api/user/register", body)
	expectStatus(t, response, http.StatusOK)

	if repo.createCalls != 1 || repo.createEmail != "ada@example.com" {
		t.Fatalf("repository received calls=%d email=%q, want one call for ada@example.com", repo.createCalls, repo.createEmail)
	}
	// Registration must not hand back a session; the user still has to log in.
	for _, cookie := range response.Cookies() {
		if cookie.Name == "token" {
			t.Fatal("registration issued a token cookie")
		}
	}
}

func TestUserRouterRegisterRejectsInvalidCredentialsBeforeTheRepository(t *testing.T) { // IT-BE-SESS-02
	for _, tc := range []struct{ name, body string }{
		{"malformed email", `{"email":"not-an-email","password":"password123","firstName":"Ada","lastName":"Lovelace"}`},
		{"password under eight characters", `{"email":"ada@example.com","password":"short","firstName":"Ada","lastName":"Lovelace"}`},
		{"missing first name", `{"email":"ada@example.com","password":"password123","lastName":"Lovelace"}`},
		{"missing last name", `{"email":"ada@example.com","password":"password123","firstName":"Ada"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeUserRepo{}
			app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

			response := doRequest(t, app, http.MethodPost, "/api/user/register", tc.body)

			expectStatus(t, response, http.StatusBadRequest)
			if repo.createCalls != 0 {
				t.Fatal("an invalid registration reached the repository")
			}
		})
	}
}

func TestUserRouterRegisterReportsARepositoryRejection(t *testing.T) { // IT-BE-SESS-01
	repo := &fakeUserRepo{createErr: errRepository}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	body := `{"email":"ada@example.com","password":"password123","firstName":"Ada","lastName":"Lovelace"}`
	response := doRequest(t, app, http.MethodPost, "/api/user/register", body)

	// A duplicate email surfaces here, which is why the frontend shows axios' own
	// message rather than the backend's text for a taken address.
	expectStatus(t, response, http.StatusInternalServerError)
}

func TestUserRouterAuthorizeDescribesTheSignedInUser(t *testing.T) { // IT-BE-SESS-03
	repo := &fakeUserRepo{userInfo: &domain.UserInfo{
		Firstname: "Ada", Lastname: "Lovelace", CoverImage: coverImage("https://cdn.example.test/ada.png"),
	}}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/user/authorize", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	if body["authorized"] != true || body["userFirstname"] != "Ada" {
		t.Fatalf("authorize returned %v, want an authorized Ada", body)
	}
	if body["userCoverImage"] != "https://cdn.example.test/ada.png" {
		t.Fatalf("userCoverImage = %v, want the stored URL", body["userCoverImage"])
	}
	if len(repo.userInfoIDs) != 1 || repo.userInfoIDs[0] != "user-123" {
		t.Fatalf("repository received %v, want one lookup for the uid in the token", repo.userInfoIDs)
	}
}

func TestUserRouterAuthorizeRefusesWithoutAValidToken(t *testing.T) { // IT-BE-SESS-03
	for _, tc := range []struct {
		name   string
		cookie func(t *testing.T) *http.Cookie
	}{
		{"no cookie", func(*testing.T) *http.Cookie { return nil }},
		{"empty cookie", func(*testing.T) *http.Cookie { return &http.Cookie{Name: "token", Value: ""} }},
		{"not a jwt", func(*testing.T) *http.Cookie { return &http.Cookie{Name: "token", Value: "garbage"} }},
		{"signed with another secret", func(t *testing.T) *http.Cookie {
			return cookieSignedWith(t, "user-123", "not-the-server-secret")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeUserRepo{userInfo: &domain.UserInfo{Firstname: "Ada"}}
			app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

			var cookies []*http.Cookie
			if cookie := tc.cookie(t); cookie != nil {
				cookies = append(cookies, cookie)
			}
			response := doRequest(t, app, http.MethodGet, "/api/user/authorize", "", cookies...)

			expectStatus(t, response, http.StatusUnauthorized)
			if decodeJSON(t, response)["authorized"] != false {
				t.Fatal("a rejected request still reported authorized:true")
			}
			if len(repo.userInfoIDs) != 0 {
				t.Fatalf("the profile was loaded for an unauthenticated caller: %v", repo.userInfoIDs)
			}
		})
	}
}

// The group is split: login, register and authorize answer anonymous callers, and
// everything after authUser requires a session. These are the routes on the private
// side; losing the guard on any of them would expose another account's data.
func TestUserRouterGuardsThePrivateHalfOfTheGroup(t *testing.T) { // IT-BE-SESS-04
	for _, tc := range []struct{ method, target string }{
		{http.MethodPost, "/api/user/logout"},
		{http.MethodGet, "/api/user/status"},
		{http.MethodPut, "/api/user/status"},
		{http.MethodGet, "/api/user/info"},
		{http.MethodPut, "/api/user/update"},
		{http.MethodPut, "/api/user/image"},
		{http.MethodDelete, "/api/user/delete"},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			repo := &fakeUserRepo{}
			app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

			response := doRequest(t, app, tc.method, tc.target, "")

			expectStatus(t, response, http.StatusUnauthorized)
		})
	}
}

func TestUserRouterNewUserStatusReportsTheFlagForTheTokensUser(t *testing.T) { // IT-BE-SESS-05
	repo := &fakeUserRepo{newUserStatus: true}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/user/status", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if repo.newUserStatusUID != "user-123" {
		t.Fatalf("repository received uid=%q, want user-123", repo.newUserStatusUID)
	}
	// NewUserRedirect in the frontend reads res.data.userData to decide whether to
	// force the profile-setup page.
	if decodeJSON(t, response)["userData"] != true {
		t.Fatal("userData did not carry the new-user flag")
	}
}

func TestUserRouterUpdateNewUserStatusClearsTheFlag(t *testing.T) { // IT-BE-SESS-05
	repo := &fakeUserRepo{updatedStatus: false}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	response := doRequest(t, app, http.MethodPut, "/api/user/status", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if repo.updateStatusCalls != 1 {
		t.Fatalf("repository update calls = %d, want 1", repo.updateStatusCalls)
	}
	if decodeJSON(t, response)["userData"] != false {
		t.Fatal("userData = true after the setup flag was cleared")
	}
}

func TestUserRouterLogoutExpiresTheTokenCookie(t *testing.T) { // IT-BE-SESS-06
	app := newUserSessionIntegrationApp(t, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodPost, "/api/user/logout", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	var tokenCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "token" {
			tokenCookie = cookie
		}
	}
	if tokenCookie == nil {
		t.Fatal("logout did not rewrite the token cookie")
	}
	// Both halves matter: the value is cleared and the expiry is in the past, so the
	// browser drops it rather than keeping an empty but live cookie.
	if tokenCookie.Value != "" {
		t.Fatalf("token cookie value = %q, want empty", tokenCookie.Value)
	}
	if !tokenCookie.Expires.Before(timeNow()) {
		t.Fatalf("token cookie expires at %v, want a past time", tokenCookie.Expires)
	}
}

func TestUserRouterUserInfoReturnsTheProfileEnvelope(t *testing.T) { // IT-BE-SESS-07
	repo := &fakeUserRepo{userInfo: &domain.UserInfo{
		Firstname: "Ada", Lastname: "Lovelace", Phone: "0812345678",
		Address:     "12 Soi 3, Bang Mod, Thung Khru, Bangkok",
		Preferences: []domain.UserPreference{{PetType: "DOG", Breed: "Shiba Inu"}},
	}}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/user/info", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	userData, ok := decodeJSON(t, response)["userData"].(map[string]any)
	if !ok {
		t.Fatal("userData missing from the profile response")
	}
	// UserInfo carries no json tags, so Go marshals the field names verbatim — the
	// frontend's useUserInfo reads data.Firstname / data.Preferences because of that.
	if userData["Firstname"] != "Ada" || userData["Phone"] != "0812345678" {
		t.Fatalf("profile = %v, want Ada's capitalised field names", userData)
	}
	if _, present := userData["Preferences"]; !present {
		t.Fatalf("Preferences missing from %v", userData)
	}
}

func TestUserRouterUpdateUserRejectsAPreferenceForAnUnknownSpecies(t *testing.T) { // IT-BE-SESS-08
	repo := &fakeUserRepo{}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	body := `{"firstName":"Ada","lastName":"Lovelace","phoneNumber":"0812345678",` +
		`"address":"12 Soi 3","addressLat":13.65,"addressLong":100.49,` +
		`"preferences":[{"petType":"HAMSTER","breed":"Syrian"}]}`
	response := doRequest(t, app, http.MethodPut, "/api/user/update", body, authCookie(t, "user-123"))

	expectStatus(t, response, http.StatusBadRequest)
	if repo.updateInfoCalls != 0 {
		t.Fatal("an unsupported species reached the repository")
	}
}

func TestUserRouterUpdateUserPersistsTheProfileAndItsPreferences(t *testing.T) { // IT-BE-SESS-08
	repo := &fakeUserRepo{}
	app := newUserSessionIntegrationApp(t, repo, &fakeUploader{})

	body := `{"firstName":"Ada","lastName":"Lovelace","phoneNumber":"0812345678",` +
		`"address":"12 Soi 3, Bang Mod, Thung Khru, Bangkok","addressLat":13.65,"addressLong":100.49,` +
		`"preferences":[{"petType":"DOG","breed":"Shiba Inu"},{"petType":"CAT","breed":"Siamese"}]}`
	response := doRequest(t, app, http.MethodPut, "/api/user/update", body, authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if repo.updateInfoCalls != 1 || repo.updateInfoUID != "user-123" {
		t.Fatalf("repository received calls=%d uid=%q, want one update for user-123", repo.updateInfoCalls, repo.updateInfoUID)
	}
	if repo.updateInfoName != "Ada Lovelace" || repo.updateInfoPhone != "0812345678" {
		t.Fatalf("repository received name=%q phone=%q", repo.updateInfoName, repo.updateInfoPhone)
	}
	if len(repo.updateInfoPrefs) != 2 || repo.updateInfoPrefs[1].PetType != "CAT" {
		t.Fatalf("preferences = %+v, want both entries in order", repo.updateInfoPrefs)
	}
}

func TestUserRouterUpdateUserImageRequiresTheFilePart(t *testing.T) { // IT-BE-SESS-04
	uploader := &fakeUploader{}
	app := newUserSessionIntegrationApp(t, &fakeUserRepo{}, uploader)

	response := doRequest(t, app, http.MethodPut, "/api/user/image", "", authCookie(t, "user-123"))

	expectStatus(t, response, http.StatusBadRequest)
	if len(uploader.uploaded) != 0 {
		t.Fatal("storage was called without a file")
	}
}
