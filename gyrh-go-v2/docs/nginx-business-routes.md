# Nginx Business Route Allowlist

Only the routes below should be forwarded to the Go service. All other paths should be blocked by nginx.

## Frontend Pages

- `GET /`
- `GET /login`
- `GET /admin_viewer`
- `GET /assets/*`

## Public API Routes

- `GET /api/v1/health`
- `GET /api/v1/skills/active`
- `GET /api/v1/style-prompts`
- `POST /api/v1/frontend-auth/login`
- `GET /api/v1/frontend-auth/session`
- `POST /api/v1/frontend-auth/logout`
- `GET /api/v1/images/thumbnail`
- `GET /api/v1/images/download`
- `GET /api/v1/images/view`

## Protected Business API Routes

- `GET /api/v1/images`
- `DELETE /api/v1/images`
- `POST /api/v1/images/upload`
- `POST /api/v1/images/rewrite`
- `GET /api/v1/images/rewrite/tasks/*`
- `GET /api/v1/images/rewrite/tasks/*/events`
- `GET /api/v1/references`
- `DELETE /api/v1/references`
- `GET /api/v1/references/view`
- `POST /api/v1/references/upload`
- `PUT /api/v1/references/*`
- `GET /api/v1/skills`
- `POST /api/v1/skills`
- `GET /api/v1/skills/*`
- `PUT /api/v1/skills/*`
- `DELETE /api/v1/skills/*`
- `GET /api/v1/llm-prompt-templates`
- `POST /api/v1/llm-prompt-templates`
- `GET /api/v1/llm-prompt-templates/*`
- `PUT /api/v1/llm-prompt-templates/*`
- `DELETE /api/v1/llm-prompt-templates/*`
- `GET /api/v1/background-prompts`
- `POST /api/v1/background-prompts`
- `GET /api/v1/background-prompts/*`
- `PUT /api/v1/background-prompts/*`
- `DELETE /api/v1/background-prompts/*`
- `POST /api/v1/background-prompts/import`
- `POST /api/v1/background-prompts/suggest-defaults`
- `POST /api/v1/background-prompts/sync-remote`
- `POST /api/v1/background-prompts/sync-english`
- `GET /api/v1/style-prompts/*`
- `POST /api/v1/style-prompts`
- `PUT /api/v1/style-prompts/*`
- `DELETE /api/v1/style-prompts/*`

## Suggested Nginx Default

```nginx
location / {
    return 403;
}
```

Use exact or prefix `location` blocks for the allowlisted routes above before the default block.
