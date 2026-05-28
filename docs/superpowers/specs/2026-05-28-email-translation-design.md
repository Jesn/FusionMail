# Email Detail Translation Design

## Goal

Add a Translate action to the email detail page so a user can translate the currently viewed email body into Chinese, replace the visible body with the translated text, and switch back to the original body without modifying stored email data.

## Approved Approach

Use a backend translation proxy. The frontend calls the FusionMail API, and the backend forwards the request to `https://trans.ors.de5.net/translate?token=...`. This keeps the provider token out of browser bundles and avoids browser CORS failures.

## User Experience

- Add a compact toolbar button on `frontend/src/components/email/EmailDetail.tsx` near the existing reply/forward actions.
- Initial button label/title: `翻译`.
- While the request is in flight, disable the button and show a loading state.
- After a successful translation, replace the email body area with the translated Chinese text and change the action to `查看原文`.
- Clicking `查看原文` restores the existing original rendering and changes the action back to `翻译`.
- Translation is view-only state on the detail page. It is not persisted to the email record and should reset when a different email is opened.

## Content Selection

- Prefer `email.text_body` because it is already plain text.
- If `text_body` is unavailable and `html_body` exists, convert the HTML body to readable plain text before sending it to the backend proxy.
- If neither body exists, fall back to the snippet only for display and show a toast saying there is no translatable body.
- Render translated content as plain text using the existing text-body style rather than injecting translated HTML.

## Backend API

- Add `POST /api/v1/translate`.
- Request body:

```json
{
  "text": "Email body text",
  "source_lang": "auto",
  "target_lang": "ZH"
}
```

- Response body:

```json
{
  "success": true,
  "data": {
    "translated_text": "翻译后的中文正文"
  }
}
```

- The backend should reject empty `text` values with a client error.
- The backend should use a configurable token/URL when practical and may default to the provided endpoint for this implementation.
- External translation failures should return a clear API error so the frontend can show `翻译失败`.

## Frontend Data Flow

1. User opens `/email/:id`; `EmailDetailPage` loads the email as it does today.
2. `EmailDetail` receives the email and renders the existing original HTML/text body.
3. User clicks `翻译`.
4. `EmailDetail` derives plain text from the selected email body and calls a frontend service method.
5. The frontend service posts to `/translate` through the existing API client.
6. On success, `EmailDetail` stores `translatedText` and sets the view mode to translated.
7. On `查看原文`, `EmailDetail` clears only the translated view mode and displays the original body again.

## Error Handling

- Empty body: do not call the API; show a toast with `没有可翻译的内容`.
- Request failure or malformed provider response: show `翻译失败` and keep the original body visible.
- Repeated clicks while loading: disabled button prevents duplicate requests.

## Security and Privacy

- The translation token must not be embedded in frontend code.
- Translated content should be rendered as text, not unsanitized HTML.
- The feature sends email body text to the external translation service only when the user clicks `翻译`.

## Tests

- Backend service/handler tests should cover successful provider response parsing, empty input validation, and provider failure handling.
- Frontend component tests should cover clicking `翻译`, rendering translated text, toggling `查看原文`, and preserving the original body on failure.

## Spec Self-Review

- Placeholder scan: no `TBD`, `TODO`, or deferred requirements remain.
- Consistency check: UX, backend API, frontend flow, and tests all describe the backend proxy approach.
- Scope check: this is one focused feature touching one backend proxy path and one email detail UI path.
- Ambiguity check: display mode, content selection, error behavior, and non-persistence are explicit.
