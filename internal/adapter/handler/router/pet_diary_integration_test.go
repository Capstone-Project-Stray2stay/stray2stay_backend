package router_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// The diary routes are where the backend holds a permission model the UI also
// implements: only the adopter writes, only between the adoption date and today. The
// client's copy of those rules is not trustworthy, so these drive the real service and
// check that each refusal arrives with the status the frontend branches on.

func today() string { return time.Now().Format("2006-01-02") }

func tomorrow() string { return time.Now().AddDate(0, 0, 1).Format("2006-01-02") }

func TestPetRouterDiaryRoutesRequireASession(t *testing.T) { // IT-BE-DIARY-01
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/api/pets/mine/diaries"},
		{http.MethodGet, "/api/pets/42/diary?from=2026-01-01&to=2026-01-31"},
		{http.MethodPut, "/api/pets/42/diary/2026-01-05"},
		{http.MethodDelete, "/api/pets/42/diary/2026-01-05"},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			sqlRepo := &fakePetSQLRepo{}
			app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

			response := doRequest(t, app, tc.method, tc.target, "")

			expectStatus(t, response, http.StatusUnauthorized)
			if sqlRepo.diaryAccessUID != "" {
				t.Fatalf("access was checked for an unauthenticated caller (uid=%q)", sqlRepo.diaryAccessUID)
			}
		})
	}
}

func TestPetRouterDiaryEntriesReturnsTheRangeAndTheCallersRole(t *testing.T) { // IT-BE-DIARY-02
	sqlRepo := &fakePetSQLRepo{
		diaryRole:  domain.DiaryRoleAdopter,
		diarySince: "2026-01-01",
		diaryEntries: []domain.DiaryEntry{
			{Did: 1, Pid: 42, Date: "2026-01-05", Caption: "First walk"},
		},
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42/diary?from=2026-01-01&to=2026-01-31", "", authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.diaryAccessPid != 42 || sqlRepo.diaryAccessUID != "adopter-1" {
		t.Fatalf("access checked for pid=%d uid=%q, want 42 / adopter-1", sqlRepo.diaryAccessPid, sqlRepo.diaryAccessUID)
	}
	if sqlRepo.diaryRangeFrom != "2026-01-01" || sqlRepo.diaryRangeTo != "2026-01-31" {
		t.Fatalf("range passed through as %q..%q", sqlRepo.diaryRangeFrom, sqlRepo.diaryRangeTo)
	}

	body := decodeJSON(t, response)
	if body["role"] != domain.DiaryRoleAdopter || body["since"] != "2026-01-01" {
		t.Fatalf("role=%v since=%v, want ADOPTER / 2026-01-01", body["role"], body["since"])
	}
	// canWrite is derived here, not stored — the diary page enables its editor on it.
	if body["canWrite"] != true {
		t.Fatal("canWrite = false for the adopter")
	}
}

func TestPetRouterDiaryEntriesMarksTheFinderReadOnly(t *testing.T) { // IT-BE-DIARY-02
	sqlRepo := &fakePetSQLRepo{diaryRole: domain.DiaryRoleFinder, diarySince: "2026-01-01"}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42/diary?from=2026-01-01&to=2026-01-31", "", authCookie(t, "finder-1"))
	expectStatus(t, response, http.StatusOK)

	body := decodeJSON(t, response)
	// The person who rehomed the pet may read the diary but never write to it.
	if body["role"] != domain.DiaryRoleFinder || body["canWrite"] != false {
		t.Fatalf("role=%v canWrite=%v, want FINDER / false", body["role"], body["canWrite"])
	}
}

func TestPetRouterDiaryEntriesForbidsAStranger(t *testing.T) { // IT-BE-DIARY-01
	sqlRepo := &fakePetSQLRepo{diaryAccessErr: errors.New("no diary access for this pet")}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/42/diary?from=2026-01-01&to=2026-01-31", "", authCookie(t, "stranger-1"))

	// 403, not 500: the caller is authenticated, just unrelated to this pet.
	expectStatus(t, response, http.StatusForbidden)
	if sqlRepo.diaryRangeFrom != "" {
		t.Fatal("entries were read despite the access check failing")
	}
}

func TestPetRouterDiaryEntriesRejectsAMalformedOrBackwardsRange(t *testing.T) { // IT-BE-DIARY-03
	for _, tc := range []struct{ name, query string }{
		{"missing from", "?to=2026-01-31"},
		{"missing to", "?from=2026-01-01"},
		{"unparseable from", "?from=01-01-2026&to=2026-01-31"},
		{"unparseable to", "?from=2026-01-01&to=31st"},
		{"ends before it starts", "?from=2026-01-31&to=2026-01-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlRepo := &fakePetSQLRepo{diaryRole: domain.DiaryRoleAdopter, diarySince: "2026-01-01"}
			app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

			response := doRequest(t, app, http.MethodGet, "/api/pets/42/diary"+tc.query, "", authCookie(t, "adopter-1"))

			expectStatus(t, response, http.StatusBadRequest)
			if sqlRepo.diaryRangeFrom != "" {
				t.Fatalf("a bad range reached the entries query: %q..%q", sqlRepo.diaryRangeFrom, sqlRepo.diaryRangeTo)
			}
		})
	}
}

func TestPetRouterSaveDiaryEntryUploadsThePhotoAndStoresTheDay(t *testing.T) { // IT-BE-DIARY-04
	sqlRepo := &fakePetSQLRepo{
		diaryRole:   domain.DiaryRoleAdopter,
		diarySince:  "2020-01-01",
		upsertEntry: domain.DiaryEntry{Did: 9, Pid: 42, Date: today(), Caption: "Good day"},
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "image", filename: "day.jpg", value: "fake-jpeg-bytes"},
		{name: "caption", value: "Good day"},
	}, authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	if len(uploader.uploaded) != 1 {
		t.Fatalf("uploader received %v, want one image", uploader.uploaded)
	}
	if sqlRepo.upsertCalls != 1 || sqlRepo.upsertCaption != "Good day" {
		t.Fatalf("repository received calls=%d caption=%q", sqlRepo.upsertCalls, sqlRepo.upsertCaption)
	}
	// The row must point at the uploaded URL, not at the raw form value.
	if sqlRepo.upsertImage != uploader.uploaded[0] {
		t.Fatalf("stored image = %q, want the uploaded URL %q", sqlRepo.upsertImage, uploader.uploaded[0])
	}
}

func TestPetRouterSaveDiaryEntryRemovesTheReplacedPhoto(t *testing.T) { // IT-BE-DIARY-04
	sqlRepo := &fakePetSQLRepo{
		diaryRole:           domain.DiaryRoleAdopter,
		diarySince:          "2020-01-01",
		upsertReplacedImage: "https://cdn.example.test/diary/old.jpg",
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "image", filename: "day.jpg", value: "fake-jpeg-bytes"},
	}, authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	// Swapping a day's photo must not leave the previous file orphaned in storage.
	if len(uploader.deleted) != 1 || uploader.deleted[0] != "https://cdn.example.test/diary/old.jpg" {
		t.Fatalf("uploader deleted %v, want the replaced image", uploader.deleted)
	}
}

func TestPetRouterSaveDiaryEntryDropsTheUploadWhenTheRowFails(t *testing.T) { // IT-BE-DIARY-05
	sqlRepo := &fakePetSQLRepo{
		diaryRole:  domain.DiaryRoleAdopter,
		diarySince: "2020-01-01",
		upsertErr:  errRepository,
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "image", filename: "day.jpg", value: "fake-jpeg-bytes"},
	}, authCookie(t, "adopter-1"))

	expectStatus(t, response, http.StatusInternalServerError)
	// The upload already happened, so the orphan has to be cleaned up.
	if len(uploader.deleted) != 1 || uploader.deleted[0] != uploader.uploaded[0] {
		t.Fatalf("uploader deleted %v, want the just-uploaded orphan %v", uploader.deleted, uploader.uploaded)
	}
}

func TestPetRouterSaveDiaryEntryRefusesTheFinder(t *testing.T) { // IT-BE-DIARY-06
	sqlRepo := &fakePetSQLRepo{diaryRole: domain.DiaryRoleFinder, diarySince: "2020-01-01"}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "image", filename: "day.jpg", value: "fake-jpeg-bytes"},
		{name: "caption", value: "Not mine to write"},
	}, authCookie(t, "finder-1"))

	expectStatus(t, response, http.StatusForbidden)
	if sqlRepo.upsertCalls != 0 || len(uploader.uploaded) != 0 {
		t.Fatal("a read-only caller reached storage or the database")
	}
}

func TestPetRouterSaveDiaryEntryRejectsDaysOutsideTheWritableWindow(t *testing.T) { // IT-BE-DIARY-07
	for _, tc := range []struct {
		name  string
		date  string
		since string
	}{
		{"future day", tomorrow(), "2020-01-01"},
		{"before the adoption", "2019-01-01", "2020-01-01"},
		{"unparseable day", "05-01-2026", "2020-01-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlRepo := &fakePetSQLRepo{diaryRole: domain.DiaryRoleAdopter, diarySince: tc.since}
			uploader := &fakeUploader{}
			app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

			response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+tc.date, []multipartField{
				{name: "image", filename: "day.jpg", value: "fake-jpeg-bytes"},
			}, authCookie(t, "adopter-1"))

			expectStatus(t, response, http.StatusBadRequest)
			if sqlRepo.upsertCalls != 0 || len(uploader.uploaded) != 0 {
				t.Fatal("an out-of-window day reached storage or the database")
			}
		})
	}
}

func TestPetRouterSaveDiaryEntryAllowsACaptionOnlyEditOfAnExistingDay(t *testing.T) { // IT-BE-DIARY-08
	sqlRepo := &fakePetSQLRepo{
		diaryRole:     domain.DiaryRoleAdopter,
		diarySince:    "2020-01-01",
		existingEntry: domain.DiaryEntry{Did: 9, Pid: 42, Date: today(), ImageAddress: "https://cdn.example.test/diary/old.jpg"},
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "caption", value: "Fixed a typo"},
	}, authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.upsertCalls != 1 || sqlRepo.upsertCaption != "Fixed a typo" {
		t.Fatalf("repository received calls=%d caption=%q", sqlRepo.upsertCalls, sqlRepo.upsertCaption)
	}
	// An empty image address is the repository's signal to keep the existing photo.
	if sqlRepo.upsertImage != "" {
		t.Fatalf("image address = %q, want empty so the stored photo survives", sqlRepo.upsertImage)
	}
	if len(uploader.uploaded) != 0 {
		t.Fatal("a caption-only edit re-uploaded a photo")
	}
}

func TestPetRouterSaveDiaryEntryRequiresAPhotoOnABrandNewDay(t *testing.T) { // IT-BE-DIARY-08
	sqlRepo := &fakePetSQLRepo{
		diaryRole:        domain.DiaryRoleAdopter,
		diarySince:       "2020-01-01",
		existingEntryErr: errNoRows,
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doMultipart(t, app, http.MethodPut, "/api/pets/42/diary/"+today(), []multipartField{
		{name: "caption", value: "No photo"},
	}, authCookie(t, "adopter-1"))

	expectStatus(t, response, http.StatusBadRequest)
	if sqlRepo.upsertCalls != 0 {
		t.Fatal("an entry with neither a new nor an existing photo was written")
	}
}

func TestPetRouterDeleteDiaryEntryRemovesTheRowAndItsPhoto(t *testing.T) { // IT-BE-DIARY-09
	sqlRepo := &fakePetSQLRepo{
		diaryRole:        domain.DiaryRoleAdopter,
		diarySince:       "2020-01-01",
		deleteEntryImage: "https://cdn.example.test/diary/day.jpg",
	}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doRequest(t, app, http.MethodDelete, "/api/pets/42/diary/"+today(), "", authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.deleteEntryCalls != 1 || sqlRepo.deleteEntryDate != today() {
		t.Fatalf("repository received calls=%d date=%q", sqlRepo.deleteEntryCalls, sqlRepo.deleteEntryDate)
	}
	if len(uploader.deleted) != 1 {
		t.Fatalf("uploader deleted %v, want the entry's photo", uploader.deleted)
	}
}

func TestPetRouterDeleteDiaryEntryRefusesTheFinder(t *testing.T) { // IT-BE-DIARY-06
	sqlRepo := &fakePetSQLRepo{diaryRole: domain.DiaryRoleFinder, diarySince: "2020-01-01"}
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doRequest(t, app, http.MethodDelete, "/api/pets/42/diary/"+today(), "", authCookie(t, "finder-1"))

	expectStatus(t, response, http.StatusForbidden)
	if sqlRepo.deleteEntryCalls != 0 || len(uploader.deleted) != 0 {
		t.Fatal("a read-only caller deleted a diary entry")
	}
}

func TestPetRouterMyDiaryPetsUsesTheSessionUID(t *testing.T) { // IT-BE-DIARY-01
	sqlRepo := &fakePetSQLRepo{diaryPets: []domain.DiaryPet{{Pid: 42, PetName: "Mochi", Role: domain.DiaryRoleAdopter, CanWrite: true}}}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	response := doRequest(t, app, http.MethodGet, "/api/pets/mine/diaries", "", authCookie(t, "adopter-1"))
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.diaryPetUIDs) != 1 || sqlRepo.diaryPetUIDs[0] != "adopter-1" {
		t.Fatalf("repository received %v, want one call for adopter-1", sqlRepo.diaryPetUIDs)
	}
	pets, ok := decodeJSON(t, response)["diaryPets"].([]any)
	if !ok || len(pets) != 1 {
		t.Fatalf("diaryPets = %v, want the one stubbed pet", pets)
	}
}

func TestPetRouterGetScreeningQuestionsIsReadableWithoutASession(t *testing.T) { // IT-BE-SCREEN-01
	sqlRepo := &fakePetSQLRepo{
		screeningQuestions: []domain.CustomScreeningQuestion{
			{QuestionID: 1, QuestionType: "ESSAY", QuestionText: "Why this pet?", QuestionRequired: true},
		},
		screeningLocked: true,
	}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	// Adopters read the form before deciding to apply, so this one is OptionalAuth.
	response := doRequest(t, app, http.MethodGet, "/api/pets/42/screening-questions", "")
	expectStatus(t, response, http.StatusOK)

	if len(sqlRepo.screeningQuestionPids) != 1 || sqlRepo.screeningQuestionPids[0] != 42 {
		t.Fatalf("repository received %v, want [42]", sqlRepo.screeningQuestionPids)
	}
	body := decodeJSON(t, response)
	// locked tells the owner's editor the form can no longer change.
	if body["locked"] != true {
		t.Fatalf("locked = %v, want true", body["locked"])
	}
	questions, ok := body["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %v, want one entry", body["questions"])
	}
}

func TestPetRouterSaveScreeningQuestionsStoresTheWholeSet(t *testing.T) { // IT-BE-SCREEN-01
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	body := `{"questions":[` +
		`{"questionType":"ESSAY","questionText":"Why this pet?","questionRequired":true,"questionOrder":1},` +
		`{"questionType":"MULTIPLE_CHOICE","questionText":"Housing?","questionOptions":["House","Condo"],"questionOrder":2}` +
		`]}`
	response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))
	expectStatus(t, response, http.StatusOK)

	if sqlRepo.saveQuestionsCalls != 1 || sqlRepo.saveQuestionsUID != "owner-1" || sqlRepo.saveQuestionsPid != 42 {
		t.Fatalf("repository received calls=%d uid=%q pid=%d", sqlRepo.saveQuestionsCalls, sqlRepo.saveQuestionsUID, sqlRepo.saveQuestionsPid)
	}
	if len(sqlRepo.savedQuestions) != 2 || sqlRepo.savedQuestions[1].QuestionType != "MULTIPLE_CHOICE" {
		t.Fatalf("saved questions = %+v, want both in order", sqlRepo.savedQuestions)
	}
}

func TestPetRouterSaveScreeningQuestionsRejectsAnUnknownQuestionType(t *testing.T) { // IT-BE-SCREEN-02
	sqlRepo := &fakePetSQLRepo{}
	app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

	body := `{"questions":[{"questionType":"VIDEO","questionText":"Record yourself","questionOrder":1}]}`
	response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))

	expectStatus(t, response, http.StatusBadRequest)
	if sqlRepo.saveQuestionsCalls != 0 {
		t.Fatal("an unsupported question type reached the repository")
	}
}

func TestPetRouterSaveScreeningQuestionsRequiresTextExceptForImageQuestions(t *testing.T) { // IT-BE-SCREEN-02
	t.Run("text question without text", func(t *testing.T) {
		sqlRepo := &fakePetSQLRepo{}
		app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

		body := `{"questions":[{"questionType":"ESSAY","questionOrder":1}]}`
		response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))

		expectStatus(t, response, http.StatusBadRequest)
		if sqlRepo.saveQuestionsCalls != 0 {
			t.Fatal("a question with no prompt reached the repository")
		}
	})

	t.Run("image question without text", func(t *testing.T) {
		sqlRepo := &fakePetSQLRepo{}
		app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

		// IMAGE questions carry no prompt by design, hence required_unless.
		body := `{"questions":[{"questionType":"IMAGE","questionOrder":1}]}`
		response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))

		expectStatus(t, response, http.StatusOK)
		if sqlRepo.saveQuestionsCalls != 1 {
			t.Fatal("a valid image question was rejected")
		}
	})
}

func TestPetRouterSaveScreeningQuestionsKeepsIllustrationImages(t *testing.T) { // IT-BE-SCREEN-04
	t.Run("a choice question with photos keeps both", func(t *testing.T) {
		sqlRepo := &fakePetSQLRepo{}
		app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

		body := `{"questions":[{"questionType":"MULTIPLE_CHOICE","questionText":"Which bed?",` +
			`"questionOptions":["Basket","Crate"],"questionImages":["https://img/a.jpg","https://img/b.jpg"],"questionOrder":1}]}`
		response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))

		expectStatus(t, response, http.StatusOK)
		saved := sqlRepo.savedQuestions
		if len(saved) != 1 || saved[0].QuestionType != "MULTIPLE_CHOICE" || len(saved[0].QuestionOptions) != 2 || len(saved[0].QuestionImages) != 2 {
			t.Fatalf("saved = %+v, want the type, both options and both photos", saved)
		}
	})

	t.Run("more than four photos is rejected", func(t *testing.T) {
		sqlRepo := &fakePetSQLRepo{}
		app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

		body := `{"questions":[{"questionType":"ESSAY","questionText":"Why?",` +
			`"questionImages":["a","b","c","d","e"],"questionOrder":1}]}`
		response := doRequest(t, app, http.MethodPut, "/api/pets/42/screening-questions", body, authCookie(t, "owner-1"))

		expectStatus(t, response, http.StatusBadRequest)
		if sqlRepo.saveQuestionsCalls != 0 {
			t.Fatal("five illustration photos reached the repository")
		}
	})
}

func TestPetRouterUploadScreeningAnswerImageRequiresTheFilePart(t *testing.T) { // IT-BE-SCREEN-03
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, &fakePetSQLRepo{}, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPost, "/api/pets/42/screening-answer-image", []multipartField{
		{name: "caption", value: "no file here"},
	}, authCookie(t, "applicant-9"))

	expectStatus(t, response, http.StatusBadRequest)
	if len(uploader.uploaded) != 0 {
		t.Fatal("storage was called without a file")
	}
}

func TestPetRouterUploadScreeningAnswerImageReturnsTheStoredURL(t *testing.T) { // IT-BE-SCREEN-03
	uploader := &fakeUploader{}
	app := newPetRouteIntegrationApp(t, &fakePetSQLRepo{}, &fakePetMongoRepo{}, &fakeUserRepo{}, uploader)

	response := doMultipart(t, app, http.MethodPost, "/api/pets/42/screening-answer-image", []multipartField{
		{name: "image", filename: "proof.jpg", value: "fake-jpeg-bytes"},
	}, authCookie(t, "applicant-9"))
	expectStatus(t, response, http.StatusOK)

	if len(uploader.uploaded) != 1 {
		t.Fatalf("uploader received %v, want one image", uploader.uploaded)
	}
	if decodeJSON(t, response)["imageUrl"] != uploader.uploaded[0] {
		t.Fatal("response did not carry the uploaded URL")
	}
}
