package router_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

// Shared scaffolding for the router integration tests.
//
// These tests wire the real router, the real handler and the real service together and
// drive them over HTTP through fiber's app.Test. Only the repositories, the image
// uploader and the AI summarizer are substituted — everything the request touches
// between the socket and the port boundary is production code, so route ordering,
// middleware, body/query/param binding, validation, status codes and response envelopes
// are all exercised for real.

// setIntegrationEnv fills in every variable config.Load requires. The routers call
// config.Load at construction time, so without these the test binary would die inside
// log.Fatalf instead of failing a test.
func setIntegrationEnv(t *testing.T) {
	t.Helper()

	for key, value := range map[string]string{
		"ENV":                   "development",
		"ORIGIN":                "http://localhost:5173",
		"MYSQL_DB_HOST":         "127.0.0.1",
		"MYSQL_DB_PORT":         "3306",
		"MYSQL_DB_USER":         "test-user",
		"MYSQL_DB_PASSWORD":     "test-password",
		"MYSQL_DB_NAME":         "test-database",
		"MONGO_DB_URI":          "mongodb://127.0.0.1:27017",
		"MONGO_DB_NAME":         "test-database",
		"SESSION_SECRET":        "session-integration-test-secret",
		"GOOGLE_CLIENT_ID":      "test-google-client",
		"GOOGLE_CLIENT_SECRET":  "test-google-secret",
		"GOOGLE_CALLBACK_URL":   "http://localhost/callback",
		"CLOUDINARY_CLOUD_NAME": "test-cloud",
		"CLOUDINARY_API_KEY":    "test-cloudinary-key",
		"CLOUDINARY_API_SECRET": "test-cloudinary-secret",
		"JWT_SECRET":            integrationJWTSecret,
	} {
		t.Setenv(key, value)
	}
}

// authCookie mints the session cookie the handlers and middleware expect, signed with
// the same secret setIntegrationEnv installs.
func authCookie(t *testing.T, uid string) *http.Cookie {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(integrationJWTSecret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return &http.Cookie{Name: "token", Value: signed}
}

// cookieSignedWith mints a token with some other secret, to prove the middleware checks
// the signature rather than merely the cookie's presence.
func cookieSignedWith(t *testing.T, uid string, secret string) *http.Cookie {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return &http.Cookie{Name: "token", Value: signed}
}

// doRequest sends one request through the app. Pass cookies to act as a signed-in user.
func doRequest(t *testing.T, app *fiber.App, method string, target string, body string, cookies ...*http.Cookie) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	if body != "" {
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	// -1 disables app.Test's default 1s timeout; a handler that spawns background work
	// would otherwise make this flaky on a loaded machine.
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// multipartField is one form value; a non-empty filename turns it into a file part.
type multipartField struct {
	name     string
	filename string
	value    string
}

// doMultipart sends a multipart/form-data request, which is how the diary, pet
// registration and image-upload routes are actually called.
func doMultipart(t *testing.T, app *fiber.App, method string, target string, fields []multipartField, cookies ...*http.Cookie) *http.Response {
	t.Helper()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for _, field := range fields {
		var (
			part io.Writer
			err  error
		)
		if field.filename != "" {
			part, err = writer.CreateFormFile(field.name, field.filename)
		} else {
			part, err = writer.CreateFormField(field.name)
		}
		if err != nil {
			t.Fatalf("build multipart field %q: %v", field.name, err)
		}
		if _, err := part.Write([]byte(field.value)); err != nil {
			t.Fatalf("write multipart field %q: %v", field.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(method, target, &buffer)
	request.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// expectStatus fails with the response body attached, which is what you actually need
// when a 400 shows up where a 200 was expected.
func expectStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()

	if response.StatusCode != want {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, want, body)
	}
}

// decodeJSON reads the response envelope into a generic map.
func decodeJSON(t *testing.T, response *http.Response) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

// ---------------------------------------------------------------------------
// Fakes. Each embeds its port so the compiler is satisfied without stubbing out
// methods a given test never reaches; the embedded interface is nil, so an
// unexpected call panics loudly instead of silently returning a zero value.
// ---------------------------------------------------------------------------

// petSearchCall records exactly what the repository was asked for, which is where the
// query-string binding and the service's defaulting become observable.
type petSearchCall struct {
	uid         string
	page        int
	pageSize    int
	petAgeGroup string
	petGender   string
	petType     string
	petBreed    string
	petColor    string
	petLocation string
	keyword     string
	userLat     float64
	userLong    float64
}

type adoptCall struct {
	uid     string
	pid     int
	q1_3    string
	q2_1    string
	note    string
	answers []domain.CustomAnswerInput
}

type fakePetSQLRepo struct {
	port.PetSQLRepository

	searchResult     []domain.PetsInfo
	searchTotalCount int
	searchErr        error
	searchCalls      []petSearchCall

	petInfo    domain.PetInfo
	petInfoErr error
	petInfoIDs []int

	suggestions    []domain.PetsInfo
	suggestionsErr error

	ownerPets    []domain.PetsInfo
	ownerPetsErr error
	ownerPetUIDs []string

	adoptionStatus     string
	adoptionStatusErr  error
	adoptionStatusPid  int
	adoptionStatusUID  string
	adoptionStatusCall int

	adoptRid   int
	adoptErr   error
	adoptCalls []adoptCall

	deleteImages []string
	deleteErr    error
	deleteUID    string
	deletePid    int

	cancelErr   error
	cancelUID   string
	cancelRid   int
	cancelCalls int

	myRequests    []domain.MyAdoptionRequest
	myRequestsErr error
	myRequestUIDs []string

	summaryUpdates []string

	screeningQuestions    []domain.CustomScreeningQuestion
	screeningLocked       bool
	screeningErr          error
	screeningQuestionPids []int

	saveQuestionsErr   error
	saveQuestionsCalls int
	saveQuestionsUID   string
	saveQuestionsPid   int
	savedQuestions     []domain.ScreeningQuestionInput

	diaryPets    []domain.DiaryPet
	diaryPetsErr error
	diaryPetUIDs []string

	diaryRole      string
	diarySince     string
	diaryAccessErr error
	diaryAccessPid int
	diaryAccessUID string

	diaryEntries    []domain.DiaryEntry
	diaryEntriesErr error
	diaryRangeFrom  string
	diaryRangeTo    string

	existingEntry    domain.DiaryEntry
	existingEntryErr error

	upsertEntry         domain.DiaryEntry
	upsertReplacedImage string
	upsertErr           error
	upsertCalls         int
	upsertDate          string
	upsertImage         string
	upsertCaption       string

	deleteEntryImage string
	deleteEntryErr   error
	deleteEntryCalls int
	deleteEntryDate  string
}

func (r *fakePetSQLRepo) GetPetsInfo(uid string, page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, petLocation string, keyword string, userLat float64, userLong float64) ([]domain.PetsInfo, int, error) {
	r.searchCalls = append(r.searchCalls, petSearchCall{uid, page, pageSize, petAgeGroup, petGender, petType, petBreed, petColor, petLocation, keyword, userLat, userLong})
	return r.searchResult, r.searchTotalCount, r.searchErr
}

func (r *fakePetSQLRepo) GetPetInfo(pid int) (domain.PetInfo, error) {
	r.petInfoIDs = append(r.petInfoIDs, pid)
	return r.petInfo, r.petInfoErr
}

func (r *fakePetSQLRepo) GetPetsSuggestion() ([]domain.PetsInfo, error) {
	return r.suggestions, r.suggestionsErr
}

func (r *fakePetSQLRepo) GetPetsByOwner(uid string) ([]domain.PetsInfo, error) {
	r.ownerPetUIDs = append(r.ownerPetUIDs, uid)
	return r.ownerPets, r.ownerPetsErr
}

func (r *fakePetSQLRepo) GetMyAdoptionStatus(pid int, uid string) (string, error) {
	r.adoptionStatusCall++
	r.adoptionStatusPid, r.adoptionStatusUID = pid, uid
	return r.adoptionStatus, r.adoptionStatusErr
}

func (r *fakePetSQLRepo) PostPetAdopt(uid string, pid int, _ bool, _ bool, q1_3 string, q2_1 string, _ bool, _ bool, _ int8, _ bool, _ string, _ int8, _ int8, _ int8, _ int8, note string, answers []domain.CustomAnswerInput) (int, error) {
	r.adoptCalls = append(r.adoptCalls, adoptCall{uid: uid, pid: pid, q1_3: q1_3, q2_1: q2_1, note: note, answers: answers})
	return r.adoptRid, r.adoptErr
}

func (r *fakePetSQLRepo) DeletePet(uid string, pid int) ([]string, error) {
	r.deleteUID, r.deletePid = uid, pid
	return r.deleteImages, r.deleteErr
}

func (r *fakePetSQLRepo) CancelAdoptionRequest(uid string, rid int) error {
	r.cancelCalls++
	r.cancelUID, r.cancelRid = uid, rid
	return r.cancelErr
}

func (r *fakePetSQLRepo) GetMyAdoptionRequests(uid string) ([]domain.MyAdoptionRequest, error) {
	r.myRequestUIDs = append(r.myRequestUIDs, uid)
	return r.myRequests, r.myRequestsErr
}

// UpdateAdoptionSummary is reached from the detached goroutine AdoptPet starts. It
// records nothing a test asserts on directly; it exists so that goroutine cannot panic
// on a nil embedded interface while another test is running.
func (r *fakePetSQLRepo) UpdateAdoptionSummary(_ int, _ string, status string) error {
	r.summaryUpdates = append(r.summaryUpdates, status)
	return nil
}

func (r *fakePetSQLRepo) GetScreeningAnswer(_ int, _ string) (domain.ScreeningAnswer, error) {
	return domain.ScreeningAnswer{}, nil
}

func (r *fakePetSQLRepo) GetScreeningQuestions(pid int) ([]domain.CustomScreeningQuestion, bool, error) {
	r.screeningQuestionPids = append(r.screeningQuestionPids, pid)
	return r.screeningQuestions, r.screeningLocked, r.screeningErr
}

func (r *fakePetSQLRepo) SaveScreeningQuestions(uid string, pid int, questions []domain.ScreeningQuestionInput) error {
	r.saveQuestionsCalls++
	r.saveQuestionsUID, r.saveQuestionsPid = uid, pid
	r.savedQuestions = questions
	return r.saveQuestionsErr
}

func (r *fakePetSQLRepo) GetMyDiaryPets(uid string) ([]domain.DiaryPet, error) {
	r.diaryPetUIDs = append(r.diaryPetUIDs, uid)
	return r.diaryPets, r.diaryPetsErr
}

func (r *fakePetSQLRepo) GetDiaryAccess(pid int, uid string) (string, string, error) {
	r.diaryAccessPid, r.diaryAccessUID = pid, uid
	return r.diaryRole, r.diarySince, r.diaryAccessErr
}

func (r *fakePetSQLRepo) GetDiaryEntries(_ int, from string, to string) ([]domain.DiaryEntry, error) {
	r.diaryRangeFrom, r.diaryRangeTo = from, to
	return r.diaryEntries, r.diaryEntriesErr
}

func (r *fakePetSQLRepo) GetDiaryEntry(_ int, _ string) (domain.DiaryEntry, error) {
	return r.existingEntry, r.existingEntryErr
}

func (r *fakePetSQLRepo) UpsertDiaryEntry(_ int, _ string, date string, imageAddress string, caption string) (domain.DiaryEntry, string, error) {
	r.upsertCalls++
	r.upsertDate, r.upsertImage, r.upsertCaption = date, imageAddress, caption
	return r.upsertEntry, r.upsertReplacedImage, r.upsertErr
}

func (r *fakePetSQLRepo) DeleteDiaryEntry(_ int, date string) (string, error) {
	r.deleteEntryCalls++
	r.deleteEntryDate = date
	return r.deleteEntryImage, r.deleteEntryErr
}

type fakePetMongoRepo struct {
	port.PetMongoRepository

	breeds    []string
	breedsErr error
	breedType string

	behavior    string
	behaviorErr error
}

func (r *fakePetMongoRepo) GetBreeds(petType string) ([]string, error) {
	r.breedType = petType
	return r.breeds, r.breedsErr
}

func (r *fakePetMongoRepo) GetBreedBehavior(_ string, _ string) (string, error) {
	return r.behavior, r.behaviorErr
}

type fakeUserRepo struct {
	port.UserMySQLRepository

	userInfo    *domain.UserInfo
	userInfoErr error
	userInfoIDs []string

	createErr   error
	createEmail string
	createCalls int

	newUserStatus     bool
	newUserStatusErr  error
	newUserStatusUID  string
	updatedStatus     bool
	updateStatusCalls int

	updateInfoErr     error
	updateInfoCalls   int
	updateInfoUID     string
	updateInfoName    string
	updateInfoPhone   string
	updateInfoAddress string
	updateInfoPrefs   []domain.UserPreference
}

func (r *fakeUserRepo) GetUserInfo(uid string) (*domain.UserInfo, error) {
	r.userInfoIDs = append(r.userInfoIDs, uid)
	return r.userInfo, r.userInfoErr
}

func (r *fakeUserRepo) CreateUser(email string, _ string, _ string, _ string) error {
	r.createCalls++
	r.createEmail = email
	return r.createErr
}

func (r *fakeUserRepo) UpdateUserInfo(uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, preferences []domain.UserPreference) error {
	r.updateInfoCalls++
	r.updateInfoUID = uid
	r.updateInfoName = firstName + " " + lastName
	r.updateInfoPhone = phoneNumber
	r.updateInfoAddress = address
	r.updateInfoPrefs = preferences
	return r.updateInfoErr
}

func (r *fakeUserRepo) GetNewUserStatus(uid string) (bool, error) {
	r.newUserStatusUID = uid
	return r.newUserStatus, r.newUserStatusErr
}

func (r *fakeUserRepo) UpdateNewUserStatus(_ string) (bool, error) {
	r.updateStatusCalls++
	return r.updatedStatus, nil
}

type fakeNotificationRepo struct {
	notifications []domain.Notification
	unreadCount   int
	listErr       error
	listUIDs      []string

	markReadErr   error
	markReadUID   string
	markReadID    int
	markReadCalls int

	markAllErr   error
	markAllUID   string
	markAllCalls int
}

func (r *fakeNotificationRepo) GetNotifications(uid string) ([]domain.Notification, int, error) {
	r.listUIDs = append(r.listUIDs, uid)
	return r.notifications, r.unreadCount, r.listErr
}

func (r *fakeNotificationRepo) MarkNotificationRead(uid string, notificationID int) error {
	r.markReadCalls++
	r.markReadUID, r.markReadID = uid, notificationID
	return r.markReadErr
}

func (r *fakeNotificationRepo) MarkAllNotificationsRead(uid string) error {
	r.markAllCalls++
	r.markAllUID = uid
	return r.markAllErr
}

// fakeUploader stands in for Cloudinary. DeletePet fans out to it per image, so the
// deletions it records are how a test sees that cleanup really happened.
type fakeUploader struct {
	uploaded  []string
	uploadErr error

	deleted   []string
	deleteErr error
}

func (u *fakeUploader) UploadImages(files []*multipart.FileHeader, folder string) ([]string, error) {
	if u.uploadErr != nil {
		return nil, u.uploadErr
	}
	urls := make([]string, 0, len(files))
	for range files {
		urls = append(urls, "https://cdn.example.test/"+folder+"/image.jpg")
	}
	u.uploaded = append(u.uploaded, urls...)
	return urls, nil
}

func (u *fakeUploader) DeleteImage(imageURL string) error {
	u.deleted = append(u.deleted, imageURL)
	return u.deleteErr
}

// timeNow exists so a cookie-expiry assertion reads as intent rather than as an
// inline time.Now() buried in a comparison.
func timeNow() time.Time { return time.Now() }

// errNoRows is the sentinel the diary service checks for when deciding whether a
// caption-only edit is allowed, so the fake has to return exactly this value.
var errNoRows = sql.ErrNoRows

// errRepository is the generic storage failure these tests inject to check the error
// branches of handlers that do not care which error it was.
var errRepository = errors.New("repository unavailable")
