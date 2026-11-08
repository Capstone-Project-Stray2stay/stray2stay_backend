# Backend tests

```bash
go test ./...                                  # everything
go test ./internal/adapter/handler/router/     # the integration layer
go test ./... -race                            # the adoption summary path spawns goroutines
```

Two layers, split by package.

| Layer | Lives in | What it proves |
| --- | --- | --- |
| Unit | `internal/core/service/`, `internal/infrastructure/ai/`, `internal/adapter/handler/http/user/` | One component in isolation: prompt building, retry/backoff, summary status transitions, handler-level validation |
| Integration | `internal/adapter/handler/router/` | Router + middleware + handler + service wired together, driven over HTTP |

## The integration layer

Each test builds the real `fiber.App` through the production `*Router` function, over the
real handler and the real service. The seam is the **port boundary**: only the
repositories (`PetSQLRepository`, `PetMongoRepository`, `UserMySQLRepository`,
`NotificationRepository`), the `ImageUploader` and the `AdoptionSummarizer` are
substituted. Everything a request touches above those interfaces is production code, so
route ordering, middleware, body/query/param binding, validator tags, status codes and
response envelopes are all exercised for real — no database, no Cloudinary, no Gemini.

Shared scaffolding is in `integration_support_test.go`:

- `setIntegrationEnv(t)` — the routers call `config.Load()` at construction time, and it
  `log.Fatalf`s on a missing variable. This fills them all in.
- `authCookie(t, uid)` / `cookieSignedWith(t, uid, secret)` — mint a valid session, or a
  forged one, so guard tests can tell "has a cookie" from "has a *valid* cookie".
- `doRequest` / `doMultipart` / `expectStatus` / `decodeJSON` — request plumbing.
- The fakes. Each embeds its port interface, so a method no test reaches does not have to
  be stubbed — but calling it panics loudly rather than silently returning a zero value.

```go
sqlRepo := &fakePetSQLRepo{searchResult: pets, searchTotalCount: 41}
app := newPetRouteIntegrationApp(t, sqlRepo, &fakePetMongoRepo{}, &fakeUserRepo{}, &fakeUploader{})

response := doRequest(t, app, http.MethodGet, "/api/pets?page=3&pageSize=16", "", authCookie(t, "user-123"))
expectStatus(t, response, http.StatusOK)
// What the repository was asked for is the assertion — that is the query binding.
```

## Things worth knowing before you edit a router

- **Route order is load-bearing.** `/mine`, `/mine/adoptions`, `/mine/adoptors` and
  `/mine/diaries` are registered before `/:pid`; move them below it and every one becomes
  a pet lookup for a pet named "mine".
  `TestPetRouterKeepsMineRoutesAheadOfThePetIDWildcard` pins this.
- **`pet.Group("", middleware.AuthRequired)` guards by position, not by receiver.** It
  registers the middleware against the `/api/pets` prefix, so *every* route declared after
  that line is authenticated — including ones added via the parent `pet` group. Moving a
  route above that line silently unguards it.
- **Handlers read the session with an unchecked assertion** (`c.Locals("uid").(string)`).
  With the guard in place that is always a string; without it the handler panics instead
  of returning 401. Keep new private routes below the `authPet` line.

## Traceability to the test case register

Every test here carries the identifier of the documented case it belongs to, as a trailing
comment on its signature:

```go
func TestPetRouterSearchBindsEveryQueryParameterThroughToTheRepository(t *testing.T) { // IT-BE-PET-01
```

Identifiers read `<level>-<component>-<module>-<nn>`: `UT` unit, `IT` integration, `ST`
system; `FE` frontend, `BE` Go backend. A case may be established by more than one test, so
the same identifier can appear on several functions — the register's Coverage column names
the file.

The cases themselves live in `Stray2Stay_Test_Cases.xlsx` and
`Stray2Stay_Test_Design_Specification.docx` in the repository root. When you add, remove or
rename a test, update the matching case there so the two stay in step.
