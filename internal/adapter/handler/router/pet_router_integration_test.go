package router_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"

	pethttp "github.com/S-nudhana/stray2stay/internal/adapter/handler/http/pet"
	"github.com/S-nudhana/stray2stay/internal/adapter/handler/router"
	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/service"
)

// newPetRouteIntegrationApp wires PetRouter over the real handler and the real
// PetService, leaving only the two repositories and the uploader faked.
func newPetRouteIntegrationApp(t *testing.T, sqlRepo *fakePetSQLRepo, mongoRepo *fakePetMongoRepo, userRepo *fakeUserRepo, uploader *fakeUploader) *fiber.App {
	t.Helper()

	setIntegrationEnv(t)

	// A nil summarizer is the supported "no AI key" deployment, and it keeps the
	// detached summary goroutine out of these tests entirely.
	petService := service.NewPetService(sqlRepo, mongoRepo, uploader, userRepo, nil)
	petHandler := pethttp.NewHttpPetHandler(petService)
	app := fiber.New()
	router.PetRouter(app, petHandler)
	return app
}

func TestPetRouterSearchBindsEveryQueryParameterThroughToTheRepository(t *testing.T) { // IT-BE-PET-01
	sqlRepo := &fakePetSQLRepo{
		searchResult:     []domain.PetsInfo{{Pid: 7, PetName: "Mochi", PetBreed: "Shiba Inu"}},
		searchTotalCount: 41,
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet,
		"/api/pets?page=3&pageSize=16&petType=dog&petGender=MALE&petAgeGroup=BABY&petBreed=Shiba+Inu&petColor=Cream&petLocation=Bangkok&keyword=friendly&userLat=13.5&userLong=100.5",
		"")
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.searchCalls) != 1 {
		t.Fatalf("repository search calls = %d, want 1", len(sqlRepo.searchCalls))
	}
	got := sqlRepo.searchCalls[0]
	want := petSearchCall{
		uid: "", page: 3, pageSize: 16,
		petAgeGroup: "BABY", petGender: "MALE", petType: "dog",
		petBreed: "Shiba Inu", petColor: "Cream", petLocation: "Bangkok",
		keyword: "friendly", userLat: 13.5, userLong: 100.5,
	}
	if got != want {
		t.Fatalf("repository received %+v, want %+v", got, want)
	}

	body := decodeJSON(t, response)
	if body["totalCount"] != float64(41) {
		t.Fatalf("totalCount = %v, want 41", body["totalCount"])
	}
	// 41 rows at 16 per page is three pages — the handler computes this, not the repo.
	if body["totalPages"] != float64(3) {
		t.Fatalf("totalPages = %v, want 3", body["totalPages"])
	}
}

func TestPetRouterSearchFallsBackToTenPerPageWhenPageSizeIsMissing(t *testing.T) { // IT-BE-PET-02
	sqlRepo := &fakePetSQLRepo{searchTotalCount: 25}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets", "")
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	// The repository still sees pageSize 0; only the page count assumes 10.
	if sqlRepo.searchCalls[0].pageSize != 0 {
		t.Fatalf("repository pageSize = %d, want 0", sqlRepo.searchCalls[0].pageSize)
	}
	if body["totalPages"] != float64(3) {
		t.Fatalf("totalPages = %v, want 3 (25 rows / 10 per page)", body["totalPages"])
	}
}

func TestPetRouterSearchReportsOnePageWhenNothingMatches(t *testing.T) { // IT-BE-PET-02
	sqlRepo := &fakePetSQLRepo{searchResult: nil, searchTotalCount: 0}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets?page=1&pageSize=16", "")
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	if body["totalPages"] != float64(1) {
		t.Fatalf("totalPages = %v, want 1; an empty result must not paginate to zero pages", body["totalPages"])
	}
}

func TestPetRouterSearchUsesTheSignedInUsersSavedCoordinates(t *testing.T) { // IT-BE-PET-03
	sqlRepo := &fakePetSQLRepo{}
	userRepo := &fakeUserRepo{userInfo: &domain.UserInfo{AddressLat: 13.65, AddressLong: 100.49}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, userRepo, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets?page=1&pageSize=16", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	got := sqlRepo.searchCalls[0]
	// OptionalAuth put the uid in locals, so the service filled the distance origin in
	// from the profile rather than sorting from (0, 0) off the coast of Africa.
	if got.uid != "user-123" || got.userLat != 13.65 || got.userLong != 100.49 {
		t.Fatalf("repository received uid=%q lat=%v long=%v, want user-123 / 13.65 / 100.49", got.uid, got.userLat, got.userLong)
	}
	if len(userRepo.userInfoIDs) != 1 || userRepo.userInfoIDs[0] != "user-123" {
		t.Fatalf("user profile lookups = %v, want one for user-123", userRepo.userInfoIDs)
	}
}

func TestPetRouterSearchKeepsExplicitCoordinatesOverTheSavedProfile(t *testing.T) { // IT-BE-PET-03
	sqlRepo := &fakePetSQLRepo{}
	userRepo := &fakeUserRepo{userInfo: &domain.UserInfo{AddressLat: 13.65, AddressLong: 100.49}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, userRepo, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets?userLat=18.78&userLong=98.99", "", authCookie(t, "user-123"))
	expectStatus(t, response, http.StatusOK)

	got := sqlRepo.searchCalls[0]
	if got.userLat != 18.78 || got.userLong != 98.99 {
		t.Fatalf("repository received lat=%v long=%v, want the request's 18.78 / 98.99", got.userLat, got.userLong)
	}
	if len(userRepo.userInfoIDs) != 0 {
		t.Fatalf("profile was looked up %d times; an explicit origin should skip it", len(userRepo.userInfoIDs))
	}
}

func TestPetRouterSearchReturnsInternalErrorWhenTheRepositoryFails(t *testing.T) { // IT-BE-PET-02
	sqlRepo := &fakePetSQLRepo{searchErr: errRepository}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets", "")
	expectStatus(t, response, http.StatusInternalServerError)
}

func TestPetRouterPetInfoMergesBreedBehaviourFromMongo(t *testing.T) { // IT-BE-PET-04
	sqlRepo := &fakePetSQLRepo{petInfo: domain.PetInfo{
		Pid: 42, PetName: "Mochi", PetType: "DOG", PetBreed: "Shiba Inu", PetOwnerID: "owner-1",
	}}
	mongoRepo := &fakePetMongoRepo{behavior: "Alert and independent."}
	app := newPetRouteIntegrationApp(t, sqlRepo, mongoRepo, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "")
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.petInfoIDs) != 1 || sqlRepo.petInfoIDs[0] != 42 {
		t.Fatalf("repository received pids %v, want [42]", sqlRepo.petInfoIDs)
	}
	body := decodeJSON(t, response)
	pet, ok := body["petsInfo"].(map[string]any)
	if !ok {
		t.Fatalf("petsInfo missing from %v", body)
	}
	// petDetail is not a MySQL column; the service stitches it in from Mongo.
	if pet["petDetail"] != "Alert and independent." {
		t.Fatalf("petDetail = %v, want the Mongo behaviour text", pet["petDetail"])
	}
}

func TestPetRouterPetInfoTreatsAnAnonymousVisitorAsNeitherOwnerNorApplicant(t *testing.T) { // IT-BE-PET-05
	sqlRepo := &fakePetSQLRepo{
		petInfo:        domain.PetInfo{Pid: 42, PetOwnerID: "owner-1"},
		adoptionStatus: "PENDING",
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "")
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	if body["isOwner"] != false || body["adoptionStatus"] != "" {
		t.Fatalf("isOwner=%v adoptionStatus=%v, want false and empty", body["isOwner"], body["adoptionStatus"])
	}
	// Nobody is signed in, so there is no one to have an adoption status for.
	if sqlRepo.adoptionStatusCall != 0 {
		t.Fatalf("adoption status was queried %d times for an anonymous visitor", sqlRepo.adoptionStatusCall)
	}
}

func TestPetRouterPetInfoFlagsTheOwnerAndSkipsTheAdoptionLookup(t *testing.T) { // IT-BE-PET-05
	sqlRepo := &fakePetSQLRepo{
		petInfo:        domain.PetInfo{Pid: 42, PetOwnerID: "owner-1"},
		adoptionStatus: "PENDING",
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "", authCookie(t, "owner-1"))
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	if body["isOwner"] != true {
		t.Fatalf("isOwner = %v, want true", body["isOwner"])
	}
	// An owner cannot apply to their own pet, so the status query is pointless work.
	if sqlRepo.adoptionStatusCall != 0 {
		t.Fatalf("adoption status was queried %d times for the owner", sqlRepo.adoptionStatusCall)
	}
}

func TestPetRouterPetInfoMatchesTheOwnerAcrossAProviderSuffixedID(t *testing.T) { // IT-BE-PET-06
	sqlRepo := &fakePetSQLRepo{petInfo: domain.PetInfo{Pid: 42, PetOwnerID: "owner-1"}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	// The same account arriving through OAuth carries a "<uid>:<provider>" token.
	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "", authCookie(t, "owner-1:google"))
	expectStatus(t, response, http.StatusOK)

	if decodeJSON(t, response)["isOwner"] != true {
		t.Fatal("isOwner = false; the provider suffix must not hide ownership")
	}
}

func TestPetRouterPetInfoReportsTheCallersOwnAdoptionStatus(t *testing.T) { // IT-BE-PET-06
	sqlRepo := &fakePetSQLRepo{
		petInfo:        domain.PetInfo{Pid: 42, PetOwnerID: "owner-1"},
		adoptionStatus: "PENDING",
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "", authCookie(t, "applicant-9"))
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	if body["isOwner"] != false || body["adoptionStatus"] != "PENDING" {
		t.Fatalf("isOwner=%v adoptionStatus=%v, want false and PENDING", body["isOwner"], body["adoptionStatus"])
	}
	if sqlRepo.adoptionStatusPid != 42 || sqlRepo.adoptionStatusUID != "applicant-9" {
		t.Fatalf("adoption status queried for pid=%d uid=%q, want 42 / applicant-9", sqlRepo.adoptionStatusPid, sqlRepo.adoptionStatusUID)
	}
}

func TestPetRouterPetInfoRejectsANonNumericPetID(t *testing.T) { // IT-BE-PET-07
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/not-a-number", "")
	expectStatus(t, response, http.StatusBadRequest)

	if len(sqlRepo.petInfoIDs) != 0 {
		t.Fatalf("repository was queried with %v for an unparseable id", sqlRepo.petInfoIDs)
	}
}

// Route order is load-bearing: "/mine" and its children are declared before "/:pid", so
// a regression that reorders them would turn every one of these into a pet lookup for
// pet "mine". The 401s below are the guard proving the right handler was reached.
func TestPetRouterKeepsMineRoutesAheadOfThePetIDWildcard(t *testing.T) { // IT-BE-PET-07
	for _, target := range []string{
		"/api/pets/mine",
		"/api/pets/mine/adoptions",
		"/api/pets/mine/adoptors",
		"/api/pets/mine/diaries",
	} {
		t.Run(target, func(t *testing.T) {
			sqlRepo := &fakePetSQLRepo{}
			app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

			response := doRequest(t, app, http.MethodGet, target, "")

			expectStatus(t, response, http.StatusUnauthorized)
			if len(sqlRepo.petInfoIDs) != 0 {
				t.Fatalf("%s was handled as a pet lookup (pids %v)", target, sqlRepo.petInfoIDs)
			}
		})
	}
}

func TestPetRouterMyPetsRequiresASessionAndPassesItsUID(t *testing.T) { // IT-BE-PET-08
	sqlRepo := &fakePetSQLRepo{ownerPets: []domain.PetsInfo{{Pid: 1, PetName: "Mochi"}}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	anonymous := doRequest(t, app, http.MethodGet, "/api/pets/mine", "")
	expectStatus(t, anonymous, http.StatusUnauthorized)
	if len(sqlRepo.ownerPetUIDs) != 0 {
		t.Fatalf("repository was reached without a session: %v", sqlRepo.ownerPetUIDs)
	}

	authorized := doRequest(t, app, http.MethodGet, "/api/pets/mine", "", authCookie(t, "owner-1"))
	expectStatus(t, authorized, http.StatusOK)
	if len(sqlRepo.ownerPetUIDs) != 1 || sqlRepo.ownerPetUIDs[0] != "owner-1" {
		t.Fatalf("repository received %v, want one call for owner-1", sqlRepo.ownerPetUIDs)
	}
}

func TestPetRouterRejectsACookieSignedWithAnotherSecret(t *testing.T) { // IT-BE-PET-09
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	forged := cookieSignedWith(t, "owner-1", "not-the-server-secret")
	response := doRequest(t, app, http.MethodGet, "/api/pets/mine", "", forged)

	expectStatus(t, response, http.StatusUnauthorized)
	if len(sqlRepo.ownerPetUIDs) != 0 {
		t.Fatalf("a forged token reached the repository: %v", sqlRepo.ownerPetUIDs)
	}
}

func TestPetRouterIgnoresAForgedCookieOnAnOptionallyAuthenticatedRoute(t *testing.T) { // IT-BE-PET-09
	sqlRepo := &fakePetSQLRepo{petInfo: domain.PetInfo{Pid: 42, PetOwnerID: "owner-1"}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	forged := cookieSignedWith(t, "owner-1", "not-the-server-secret")
	response := doRequest(t, app, http.MethodGet, "/api/pets/42", "", forged)

	// OptionalAuth lets the request through, but must not trust the claim: the page
	// renders, and the forger does not get the owner's view of the pet.
	expectStatus(t, response, http.StatusOK)
	if decodeJSON(t, response)["isOwner"] != false {
		t.Fatal("a forged token was accepted as proof of ownership")
	}
}

func TestPetRouterAdoptForwardsTheFormAndReturnsTheRequestID(t *testing.T) { // IT-BE-PET-10
	sqlRepo := &fakePetSQLRepo{adoptRid: 77}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	body := `{"pid":42,"q1_1":true,"q1_2":false,"q1_3":"House with a yard","q2_1":"Family of three",` +
		`"q2_2":true,"q2_3":true,"q3_1":2,"q3_2":true,"q3_3":"Daily walks","q4_1":1,"q5_1":2,"q6_1":3,"q6_2":4,` +
		`"note":"We are ready","answers":[{"questionId":5,"value":"\"yes\""}]}`
	response := doRequest(t, app, http.MethodPost, "/api/pets/42/adopt", body, authCookie(t, "applicant-9"))
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.adoptCalls) != 1 {
		t.Fatalf("repository adopt calls = %d, want 1", len(sqlRepo.adoptCalls))
	}
	got := sqlRepo.adoptCalls[0]
	if got.uid != "applicant-9" || got.pid != 42 || got.q1_3 != "House with a yard" || got.note != "We are ready" {
		t.Fatalf("repository received %+v, want the posted form under applicant-9", got)
	}
	if len(got.answers) != 1 || got.answers[0].QuestionID != 5 {
		t.Fatalf("custom answers = %+v, want one answer for question 5", got.answers)
	}
	if decodeJSON(t, response)["rid"] != float64(77) {
		t.Fatal("response did not carry the new request id")
	}
}

func TestPetRouterAdoptRejectsAFormMissingARequiredAnswer(t *testing.T) { // IT-BE-PET-11
	sqlRepo := &fakePetSQLRepo{adoptRid: 77}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	// q1_3 and q3_3 are required by the domain's validate tags.
	body := `{"pid":42,"q2_1":"Family of three","q3_1":2,"q4_1":1,"q5_1":2,"q6_1":3,"q6_2":4}`
	response := doRequest(t, app, http.MethodPost, "/api/pets/42/adopt", body, authCookie(t, "applicant-9"))

	expectStatus(t, response, http.StatusBadRequest)
	if len(sqlRepo.adoptCalls) != 0 {
		t.Fatalf("an invalid form reached the repository: %+v", sqlRepo.adoptCalls)
	}
}

func TestPetRouterAdoptRejectsAnOutOfRangeScaleAnswer(t *testing.T) { // IT-BE-PET-11
	sqlRepo := &fakePetSQLRepo{adoptRid: 77}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	// q4_1 is capped at 2 by the validate tag.
	body := `{"pid":42,"q1_3":"Yard","q2_1":"Three","q3_1":2,"q3_3":"Walks","q4_1":9,"q5_1":2,"q6_1":3,"q6_2":4}`
	response := doRequest(t, app, http.MethodPost, "/api/pets/42/adopt", body, authCookie(t, "applicant-9"))

	expectStatus(t, response, http.StatusBadRequest)
	if len(sqlRepo.adoptCalls) != 0 {
		t.Fatalf("an out-of-range answer reached the repository: %+v", sqlRepo.adoptCalls)
	}
}

func TestPetRouterAdoptRequiresASession(t *testing.T) { // IT-BE-PET-08
	sqlRepo := &fakePetSQLRepo{adoptRid: 77}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	body := `{"pid":42,"q1_3":"Yard","q2_1":"Three","q3_1":2,"q3_3":"Walks","q4_1":1,"q5_1":2,"q6_1":3,"q6_2":4}`
	response := doRequest(t, app, http.MethodPost, "/api/pets/42/adopt", body)

	expectStatus(t, response, http.StatusUnauthorized)
	if len(sqlRepo.adoptCalls) != 0 {
		t.Fatalf("an anonymous adoption reached the repository: %+v", sqlRepo.adoptCalls)
	}
}

func TestPetRouterDeletePetCleansUpEveryStoredImage(t *testing.T) { // IT-BE-PET-12
	sqlRepo := &fakePetSQLRepo{deleteImages: []string{
		"https://cdn.example.test/pets/a.jpg",
		"https://cdn.example.test/pets/b.jpg",
	}}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doRequest(t, app, http.MethodDelete, "/api/pets/42", "", authCookie(t, "owner-1"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.deleteUID != "owner-1" || sqlRepo.deletePid != 42 {
		t.Fatalf("repository delete received uid=%q pid=%d, want owner-1 / 42", sqlRepo.deleteUID, sqlRepo.deletePid)
	}
	// The row and the images live in different systems; losing the second deletion
	// would leak storage silently.
	if len(uploader.deleted) != 2 {
		t.Fatalf("uploader deleted %v, want both image URLs", uploader.deleted)
	}
}

func TestPetRouterDeletePetDoesNotTouchStorageWhenTheRowRefusesToDelete(t *testing.T) { // IT-BE-PET-12
	sqlRepo := &fakePetSQLRepo{
		deleteImages: []string{"https://cdn.example.test/pets/a.jpg"},
		deleteErr:    errRepository,
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doRequest(t, app, http.MethodDelete, "/api/pets/42", "", authCookie(t, "not-the-owner"))

	expectStatus(t, response, http.StatusInternalServerError)
	// Ownership is enforced in SQL; if that rejects, the images must survive.
	if len(uploader.deleted) != 0 {
		t.Fatalf("uploader deleted %v after a failed row delete", uploader.deleted)
	}
}

func TestPetRouterCancelAdoptionRequestBindsTheRequestID(t *testing.T) { // IT-BE-PET-13
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodDelete, "/api/pets/mine/adoptions/19", "", authCookie(t, "applicant-9"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.cancelCalls != 1 || sqlRepo.cancelUID != "applicant-9" || sqlRepo.cancelRid != 19 {
		t.Fatalf("repository received calls=%d uid=%q rid=%d, want one cancel for applicant-9 / 19", sqlRepo.cancelCalls, sqlRepo.cancelUID, sqlRepo.cancelRid)
	}
}

func TestPetRouterCancelAdoptionRequestRejectsANonNumericID(t *testing.T) { // IT-BE-PET-07
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodDelete, "/api/pets/mine/adoptions/abc", "", authCookie(t, "applicant-9"))

	expectStatus(t, response, http.StatusBadRequest)
	if sqlRepo.cancelCalls != 0 {
		t.Fatal("an unparseable request id reached the repository")
	}
}

func TestPetRouterRandomIsOpenToAnonymousVisitors(t *testing.T) { // IT-BE-PET-14
	sqlRepo := &fakePetSQLRepo{suggestions: []domain.PetsInfo{{Pid: 1, PetName: "Mochi"}, {Pid: 2, PetName: "Kuro"}}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/random", "")
	expectStatus(t, response, http.StatusOK)

	pets, ok := decodeJSON(t, response)["petsInfo"].([]any)
	if !ok || len(pets) != 2 {
		t.Fatalf("petsInfo = %v, want the two suggested pets", pets)
	}
}

func TestPetRouterBreedsValidatesTheSpeciesBeforeReachingMongo(t *testing.T) { // IT-BE-PET-14
	for _, tc := range []struct {
		name       string
		query      string
		wantStatus int
		wantType   string
	}{
		{"dog", "?petType=dog", http.StatusOK, "dog"},
		{"cat", "?petType=cat", http.StatusOK, "cat"},
		{"unsupported species", "?petType=hamster", http.StatusBadRequest, ""},
		{"missing species", "", http.StatusBadRequest, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mongoRepo := &fakePetMongoRepo{breeds: []string{"Shiba Inu"}}
			app := newPetRouteIntegrationApp(t, &fakePetSQLRepo{}, mongoRepo, &fakeUserRepo{}, &fakeUploader{})

			response := doRequest(t, app, http.MethodGet, "/api/pets/breeds"+tc.query, "")

			expectStatus(t, response, tc.wantStatus)
			if mongoRepo.breedType != tc.wantType {
				t.Fatalf("mongo received petType=%q, want %q", mongoRepo.breedType, tc.wantType)
			}
		})
	}
}

func TestPetRouterBreedsReturnsTheEnvelopeTheFrontendReads(t *testing.T) { // IT-BE-PET-14
	mongoRepo := &fakePetMongoRepo{breeds: []string{"Shiba Inu", "Corgi"}}
	app := newPetRouteIntegrationApp(t, &fakePetSQLRepo{}, mongoRepo, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/breeds?petType=dog", "")
	expectStatus(t, response, http.StatusOK)

	// petBreedsAPI in the frontend reads res.data.breedData — renaming this key breaks
	// the breed dropdown without breaking any backend test but this one.
	var body struct {
		BreedData []string `json:"breedData"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode breeds response: %v", err)
	}
	if len(body.BreedData) != 2 || body.BreedData[0] != "Shiba Inu" {
		t.Fatalf("breedData = %v, want the two stubbed breeds", body.BreedData)
	}
}

func TestPetRouterMyAdoptionRequestsUsesTheSessionUID(t *testing.T) { // IT-BE-PET-13
	sqlRepo := &fakePetSQLRepo{myRequests: []domain.MyAdoptionRequest{{Rid: 19, Pid: 42, PetName: "Mochi"}}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/mine/adoptions", "", authCookie(t, "applicant-9"))
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.myRequestUIDs) != 1 || sqlRepo.myRequestUIDs[0] != "applicant-9" {
		t.Fatalf("repository received %v, want one call for applicant-9", sqlRepo.myRequestUIDs)
	}
	requests, ok := decodeJSON(t, response)["adoptionRequests"].([]any)
	if !ok || len(requests) != 1 {
		t.Fatalf("adoptionRequests = %v, want the one stubbed request", requests)
	}
}
