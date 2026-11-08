package router_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	notificationhttp "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/notification"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

func newNotificationRouteIntegrationApp(t *testing.T, repo *fakeNotificationRepo) *fiber.App {
	t.Helper()

	setIntegrationEnv(t)

	notificationService := service.NewNotificationService(repo)
	notificationHandler := notificationhttp.NewHttpNotificationHandler(notificationService)
	app := fiber.New()
	router.NotificationRouter(app, notificationHandler)
	return app
}

func seededNotifications() []domain.Notification {
	petID := 42
	return []domain.Notification{
		{
			ID: 1, Type: "ADOPTION_REQUESTED", Title: "New adoption request",
			Message: "Someone applied to adopt Mochi.", PetID: &petID,
			IsRead: false, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			ID: 2, Type: "ADOPTION_SELECTED", Title: "You were selected",
			Message: "You were chosen to adopt Kuro.",
			IsRead:  true, CreatedAt: time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC),
		},
	}
}

// Every notification route is behind AuthRequired. A regression that drops the
// middleware would hand one user another user's inbox, so each verb is checked.
func TestNotificationRouterRefusesEveryRouteWithoutASession(t *testing.T) { // IT-BE-NOTIF-01
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/api/notifications"},
		{http.MethodPut, "/api/notifications/read-all"},
		{http.MethodPut, "/api/notifications/1/read"},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			repo := &fakeNotificationRepo{}
			app := newNotificationRouteIntegrationApp(t, repo)

			response := doRequest(t, app, tc.method, tc.target, "")

			expectStatus(t, response, http.StatusUnauthorized)
			if len(repo.listUIDs) != 0 || repo.markReadCalls != 0 || repo.markAllCalls != 0 {
				t.Fatal("an unauthenticated request reached the repository")
			}
		})
	}
}

func TestNotificationRouterListReturnsTheInboxAndUnreadCount(t *testing.T) { // IT-BE-NOTIF-02
	repo := &fakeNotificationRepo{notifications: seededNotifications(), unreadCount: 1}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodGet, "/api/notifications", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if len(repo.listUIDs) != 1 || repo.listUIDs[0] != "user-123" {
		t.Fatalf("repository received %v, want one lookup for user-123", repo.listUIDs)
	}

	body := decodeJSON(t, response)
	if body["unreadCount"] != float64(1) {
		t.Fatalf("unreadCount = %v, want 1", body["unreadCount"])
	}
	notifications, ok := body["notifications"].([]any)
	if !ok || len(notifications) != 2 {
		t.Fatalf("notifications = %v, want two entries", body["notifications"])
	}

	first, _ := notifications[0].(map[string]any)
	// These keys are what the frontend's NotificationResponse reads.
	for key, want := range map[string]any{
		"id":     float64(1),
		"type":   "ADOPTION_REQUESTED",
		"title":  "New adoption request",
		"isRead": false,
		"petId":  float64(42),
	} {
		if first[key] != want {
			t.Fatalf("notification %s = %v, want %v", key, first[key], want)
		}
	}
	// RehomeID was nil and is tagged omitempty, so it must be absent rather than null.
	if _, present := first["rehomeId"]; present {
		t.Fatalf("rehomeId was encoded for a notification that has none: %v", first)
	}
}

func TestNotificationRouterListSurfacesARepositoryFailure(t *testing.T) { // IT-BE-NOTIF-05
	repo := &fakeNotificationRepo{listErr: errRepository}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodGet, "/api/notifications", "", authCookie(t, "user-123"))

	expectStatus(t, response, http.StatusInternalServerError)
}

func TestNotificationRouterMarkReadBindsTheIDAndTheSessionUID(t *testing.T) { // IT-BE-NOTIF-03
	repo := &fakeNotificationRepo{}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodPut, "/api/notifications/11/read", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if repo.markReadCalls != 1 || repo.markReadUID != "user-123" || repo.markReadID != 11 {
		t.Fatalf("repository received calls=%d uid=%q id=%d, want one call for user-123 / 11", repo.markReadCalls, repo.markReadUID, repo.markReadID)
	}
	if repo.markAllCalls != 0 {
		t.Fatal("a single mark-read was routed to mark-all")
	}
}

func TestNotificationRouterMarkReadRejectsIDsThatCannotIdentifyARow(t *testing.T) { // IT-BE-NOTIF-03
	for _, tc := range []struct{ name, target string }{
		{"non numeric", "/api/notifications/abc/read"},
		{"zero", "/api/notifications/0/read"},
		{"negative", "/api/notifications/-3/read"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeNotificationRepo{}
			app := newNotificationRouteIntegrationApp(t, repo)

			response := doRequest(t, app, http.MethodPut, tc.target, "", authCookie(t, "user-123"))

			expectStatus(t, response, http.StatusBadRequest)
			if repo.markReadCalls != 0 {
				t.Fatalf("%s reached the repository", tc.target)
			}
		})
	}
}

func TestNotificationRouterMarkReadSurfacesARepositoryFailure(t *testing.T) { // IT-BE-NOTIF-05
	repo := &fakeNotificationRepo{markReadErr: errRepository}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodPut, "/api/notifications/11/read", "", authCookie(t, "user-123"))

	expectStatus(t, response, http.StatusInternalServerError)
}

// "read-all" is a single segment and ":nid/read" is two, but both sit under the same
// group: if "/read-all" ever moved below the wildcard it would still match nothing and
// 404, so this pins the intended handler.
func TestNotificationRouterMarkAllReadIsNotParsedAsANotificationID(t *testing.T) { // IT-BE-NOTIF-04
	repo := &fakeNotificationRepo{}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodPut, "/api/notifications/read-all", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	if repo.markAllCalls != 1 || repo.markAllUID != "user-123" {
		t.Fatalf("repository received calls=%d uid=%q, want one mark-all for user-123", repo.markAllCalls, repo.markAllUID)
	}
	if repo.markReadCalls != 0 {
		t.Fatal("read-all was handled as a single mark-read")
	}
}

func TestNotificationRouterMarkAllReadSurfacesARepositoryFailure(t *testing.T) { // IT-BE-NOTIF-05
	repo := &fakeNotificationRepo{markAllErr: errRepository}
	app := newNotificationRouteIntegrationApp(t, repo)

	response := doRequest(t, app, http.MethodPut, "/api/notifications/read-all", "", authCookie(t, "user-123"))

	expectStatus(t, response, http.StatusInternalServerError)
}

func TestNotificationRouterIsolatesOneUsersInboxFromAnothers(t *testing.T) { // IT-BE-NOTIF-02
	repo := &fakeNotificationRepo{notifications: seededNotifications(), unreadCount: 1}
	app := newNotificationRouteIntegrationApp(t, repo)

	doRequest(t, app, http.MethodGet, "/api/notifications", "", authCookie(t, "user-123"))
	doRequest(t, app, http.MethodGet, "/api/notifications", "", authCookie(t, "user-456"))

	// The uid is never taken from the body or a query parameter, only from the signed
	// cookie, so each lookup is scoped to its own caller.
	if len(repo.listUIDs) != 2 || repo.listUIDs[0] != "user-123" || repo.listUIDs[1] != "user-456" {
		t.Fatalf("repository received %v, want one lookup per caller", repo.listUIDs)
	}
}
