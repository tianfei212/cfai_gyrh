# Local Background Skill Fusion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix local uploaded background generation so temporary backgrounds do not update `background_prompts`, use the active provider Skill as the default fusion prompt, and still record generated images.

**Architecture:** Keep source selection at the frontend boundary and prompt selection in the backend model service. `CaptureScreen` sends no `background_prompt_id` for local `blob:` backgrounds. `ImageHandler.Rewrite` allows temporary backgrounds, while `llm.buildPrompt` loads the active Skill when a background image has no template ID.

**Tech Stack:** React 18/Vite frontend, Go backend, SQLite repositories, existing LLM router and storage abstractions.

---

## File Structure

- Modify `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`: remove hardcoded local `backgroundPromptId = 1`, throw a clear error when local background conversion fails, and only include `background_prompt_id` for real background prompt records.
- Modify `gyrh-go-v2/backend/internal/api/handler/image.go`: accept `background` without `background_prompt_id` and keep persistence limited to template-backed backgrounds.
- Modify `gyrh-go-v2/backend/internal/core/llm/router.go`: load active provider Skill for temporary background fusion.
- Modify `gyrh-go-v2/backend/internal/core/llm/router_test.go`: add focused tests for active Skill fallback behavior and missing Skill errors.

## Task 1: Backend Prompt Fallback

**Files:**
- Modify: `gyrh-go-v2/backend/internal/core/llm/router.go`
- Test: `gyrh-go-v2/backend/internal/core/llm/router_test.go`

- [ ] **Step 1: Write a failing test for temporary background Skill fallback**

Add a test in `router_test.go` that builds a `service` with an in-memory `SkillRepo`, creates an active `google` skill, calls `buildPrompt` with a background image and `BackgroundPromptID: 0`, and asserts the resolved prompt equals the skill content.

- [ ] **Step 2: Run the backend LLM tests and confirm failure**

Run: `go test ./internal/core/llm`

Expected: fail because `buildPrompt` currently leaves the prompt empty for a background image without `background_prompt_id`.

- [ ] **Step 3: Implement active Skill fallback in `buildPrompt`**

In `buildPrompt`, after the background template branch, add a branch for `hasImageType(params.Images, ImageTypeBackground) && params.BackgroundPromptID == 0`. It should call `s.skillRepo.GetActive(provider)`, return a wrapped error if unavailable, and set `resolved.Prompt` to the active skill content.

- [ ] **Step 4: Run the backend LLM tests and confirm pass**

Run: `go test ./internal/core/llm`

Expected: pass.

## Task 2: Rewrite Validation

**Files:**
- Modify: `gyrh-go-v2/backend/internal/api/handler/image.go`

- [ ] **Step 1: Remove the invalid validation**

Delete the check that rejects `hasBackground && req.BackgroundPromptID <= 0` with “提供背景图时必须同时提供 background_prompt_id”.

- [ ] **Step 2: Keep incompatible prompt validation**

Keep the check that rejects background fusion with `style_prompt`, so style transfer remains separate from background fusion.

- [ ] **Step 3: Run backend tests**

Run: `go test ./...`

Expected: pass.

## Task 3: Frontend Request Payload

**Files:**
- Modify: `gyrh-go-v2/frontend/src/screens/CaptureScreen.jsx`

- [ ] **Step 1: Update local background payload construction**

For `typeof selectedBg === 'string'`, convert the image to base64 but leave `backgroundPromptId` at `0`. Only assign `backgroundPromptId = selectedBg.id` for object backgrounds with `image_url`.

- [ ] **Step 2: Include `background_prompt_id` only when positive**

When `backgroundBase64` exists, always set `payload.background`. Set `payload.background_prompt_id` only if `backgroundPromptId > 0`.

- [ ] **Step 3: Make background conversion failure explicit**

If selected background conversion fails, throw an error so the user sees a generation failure instead of silently sending a request without the intended background.

- [ ] **Step 4: Build frontend**

Run: `npm run build` in `gyrh-go-v2/frontend`.

Expected: build succeeds.

## Task 4: Verification

**Files:**
- Verify edited files and diagnostics.

- [ ] **Step 1: Run backend tests**

Run: `go test ./...` in `gyrh-go-v2/backend`.

Expected: pass.

- [ ] **Step 2: Run frontend build**

Run: `npm run build` in `gyrh-go-v2/frontend`.

Expected: pass.

- [ ] **Step 3: Check lints for edited files**

Use IDE diagnostics for `CaptureScreen.jsx`, `image.go`, and `router.go`.

Expected: no new diagnostics from the edited code.
