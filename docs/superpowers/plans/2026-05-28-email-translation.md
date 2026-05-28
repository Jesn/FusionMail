# Email Detail Translation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Do not create git commits unless the user explicitly asks.

**Goal:** Add a protected backend translation proxy and a mail-detail toolbar action that replaces the visible body with a Chinese translation, with a `查看原文` toggle.

**Architecture:** The Go backend owns the external translation token and exposes `POST /api/v1/translate`. The React frontend calls that protected FusionMail endpoint through a focused `translationService`, converts HTML email bodies to plain text when needed, and stores translated text as component-local view state only.

**Tech Stack:** Go 1.24, Gin, net/http, React 19, TypeScript, Vite, Vitest, React Testing Library.

---

### Task 1: Backend translation service

**Files:**
- Create: `backend/internal/service/translation_service.go`
- Test: `backend/internal/service/translation_service_test.go`

- [ ] **Step 1: Write failing service tests**

Create tests that start an `httptest.Server`, call `NewTranslationService(...).Translate(...)`, and assert:

```go
result, err := translator.Translate(context.Background(), TranslationRequest{
    Text:       "Hello world",
    SourceLang: "auto",
    TargetLang: "ZH",
})
if err != nil {
    t.Fatalf("Translate returned error: %v", err)
}
if result.TranslatedText != "你好，世界" {
    t.Fatalf("expected translated text, got %q", result.TranslatedText)
}
```

Also cover empty input and provider failures with `dto.AsAPIError` checks.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/service -run Translation -count=1`

Expected: FAIL because translation service types/functions do not exist.

- [ ] **Step 3: Implement service**

Add:

```go
type TranslationRequest struct {
    Text       string `json:"text"`
    SourceLang string `json:"source_lang"`
    TargetLang string `json:"target_lang"`
}

type TranslationResult struct {
    TranslatedText string `json:"translated_text"`
}
```

`Translate` should trim and validate `Text`, default `source_lang` to `auto`, default `target_lang` to `ZH`, add the token as a query parameter server-side, post JSON to the configured provider URL, parse `translated_text` or `translatedText`, and return `dto.ErrInvalidRequest` or `dto.ErrOperationFailed` without leaking provider response bodies.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `go test ./internal/service -run Translation -count=1`

Expected: PASS.

### Task 2: Backend handler, config, and route

**Files:**
- Create: `backend/internal/handler/translation_handler.go`
- Test: `backend/internal/handler/translation_handler_test.go`
- Modify: `backend/config/config.go`
- Modify: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Write failing handler tests**

Create a fake `service.TranslationService` implementation and assert `TranslationHandler.Translate` returns `200` with:

```json
{"success":true,"data":{"translated_text":"你好"}}
```

Also assert malformed JSON returns `400`.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/handler -run Translation -count=1`

Expected: FAIL because handler does not exist.

- [ ] **Step 3: Implement handler/config/route wiring**

Add a `TranslationConfig` to `config.Config` with env-backed fields:

```go
Translation: TranslationConfig{
    APIURL:         getEnv("TRANSLATION_API_URL", "https://trans.ors.de5.net/translate"),
    Token:          getEnv("TRANSLATION_TOKEN", ""),
    TimeoutSeconds: getEnvInt("TRANSLATION_TIMEOUT_SECONDS", 30),
},
```

Create `handler.NewTranslationHandler`, register a protected `POST /api/v1/translate` route via `router.RegisterTranslationRoutes`, and instantiate the service/handler in `cmd/server/main.go`. The token must come from env and must not be hardcoded in source.

- [ ] **Step 4: Run backend tests**

Run: `go test ./internal/service ./internal/handler -run Translation -count=1`

Expected: PASS.

### Task 3: Frontend translation service and text extraction

**Files:**
- Create: `frontend/src/services/translationService.ts`
- Create: `frontend/src/utils/emailTranslation.ts`
- Test: `frontend/src/utils/emailTranslation.test.ts`

- [ ] **Step 1: Write failing frontend utility tests**

Assert `getTranslatableEmailText` prefers `text_body`, converts simple HTML to readable text, and returns an empty string when only `snippet` exists.

- [ ] **Step 2: Run tests and verify RED**

Run: `npm test -- --run src/utils/emailTranslation.test.ts`

Expected: FAIL because the utility does not exist.

- [ ] **Step 3: Implement frontend service and utility**

`translationService.translateEmailText(text)` should post to `/translate` through `api.post` and return `response.data.translated_text`. `getTranslatableEmailText(email)` should not inject HTML; it should use DOM text extraction or a safe tag-stripping fallback.

- [ ] **Step 4: Run utility tests and verify GREEN**

Run: `npm test -- --run src/utils/emailTranslation.test.ts`

Expected: PASS.

### Task 4: EmailDetail translate UI

**Files:**
- Modify: `frontend/src/components/email/EmailDetail.tsx`
- Test: `frontend/src/components/email/EmailDetail.test.tsx`

- [ ] **Step 1: Write failing component tests**

Mock `translationService.translateEmailText`. Assert clicking `翻译` renders translated text and changes the action to `查看原文`; clicking `查看原文` restores the original body. Assert API failure shows `翻译失败` and keeps original content visible.

- [ ] **Step 2: Run component tests and verify RED**

Run: `npm test -- --run src/components/email/EmailDetail.test.tsx`

Expected: FAIL because there is no translate UI.

- [ ] **Step 3: Implement UI**

Add a compact toolbar button with loading state, local `translatedText` and `isTranslatedView` state, reset on `email.id` changes, call `getTranslatableEmailText(email)`, show `没有可翻译的内容` for empty body, show `翻译完成` on success, and render translated text using `.email-text-content`.

- [ ] **Step 4: Run component tests and verify GREEN**

Run: `npm test -- --run src/components/email/EmailDetail.test.tsx`

Expected: PASS.

### Task 5: Final verification

**Files:**
- Verify all touched frontend/backend files.

- [ ] **Step 1: Format Go files**

Run: `gofmt -w backend/internal/service/translation_service.go backend/internal/service/translation_service_test.go backend/internal/handler/translation_handler.go backend/internal/handler/translation_handler_test.go backend/config/config.go backend/internal/router/router.go backend/cmd/server/main.go`

- [ ] **Step 2: Run backend focused tests**

Run: `go test ./internal/service ./internal/handler -run Translation -count=1`

- [ ] **Step 3: Run frontend focused tests**

Run: `npm test -- --run src/utils/emailTranslation.test.ts src/components/email/EmailDetail.test.tsx`

- [ ] **Step 4: Run frontend build/typecheck**

Run: `npm run build`

- [ ] **Step 5: Report environment requirement**

Tell the user to set `TRANSLATION_TOKEN=1c6823aa2250` in the backend runtime environment, and optionally `TRANSLATION_API_URL=https://trans.ors.de5.net/translate`.
