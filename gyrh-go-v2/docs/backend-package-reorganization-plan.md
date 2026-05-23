# Backend Package Reorganization Plan

## Goal

Reduce backend package coupling by moving business use cases out of HTTP handlers and keeping `internal/platform/app` as the composition root.

## Target Boundaries

- `internal/api`: route registration only.
- `internal/api/handler`: HTTP request parsing, response writing, and middleware adapters only.
- `internal/application/<feature>`: business use cases and workflow orchestration.
- `internal/domain/<feature>`: stable domain types that do not depend on HTTP, SQLite, OSS, or model providers.
- `internal/infrastructure/<feature>`: repository and provider implementations.
- `internal/platform/app`: dependency wiring only.

## Phase 1: Authentication Boundary

Completed first because it is isolated and low risk.

- Business package: `internal/application/frontendauth`
- HTTP adapter: `internal/api/handler/frontend_auth.go`
- Responsibility split:
  - `frontendauth.Service`: JWT sign/verify, hot reload `.env.local`, password validation integration, session detection.
  - `FrontendAuthHandler`: reads `HOME1/HOME2`, writes HttpOnly cookie, returns JSON, exposes route middleware.

## Phase 2: Rewrite Task Boundary

Move async rewrite task state out of `internal/api/handler/rewrite_task.go`.

- Create `internal/application/rewrite`
- Move task store/status/done-channel logic into the application package.
- Keep SSE formatting in `api/handler`, because it is HTTP-specific.
- Keep DB persistence behind a small interface used by `application/rewrite`.

## Phase 3: Image Workflow Boundary

Expand `internal/application/image` until `ImageHandler` becomes a thin adapter.

Move use cases in this order:

- image listing and access target resolution
- upload and delete workflows
- rewrite input preparation
- compose/persist rewrite result
- async rewrite orchestration after Phase 2

## Phase 4: Background Prompt Boundary

Create `internal/application/backgroundprompt`.

Move remote sync/import, image download, dimension parsing, prompt suggestion orchestration, and category binding use cases out of `api/handler/background_prompt.go`.

## Phase 5: Smaller Feature Boundaries

- `internal/application/reference`
- `internal/application/skill`
- `internal/application/styleprompt`
- `internal/application/prompttemplate`

Each handler should only parse HTTP and delegate to its application service.

## Phase 6: Infrastructure Split

After handlers no longer depend directly on DB structs, split `internal/db` into infrastructure packages:

- `internal/infrastructure/sqlite`
- `internal/infrastructure/image`
- `internal/infrastructure/backgroundprompt`
- `internal/infrastructure/rewrite`

Do not move SQL first; stabilize application interfaces before moving storage implementation.

## Verification Requirements

For every phase:

- Run targeted Go tests for moved packages.
- Run `go test -race` for affected backend packages.
- Run frontend tests/build if route or response behavior changes.
- Restart `go run ./cmd/server` and verify critical routes when HTTP behavior changes.
