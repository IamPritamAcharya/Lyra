# ADR-001: Use a static React/Vite frontend

## Status

Accepted — 2026-08-13.

## Context

Lyra now needs a browser interface for the public identification flow. The Go modular monolith remains the only backend. Adding a Node/NestJS backend would duplicate API, authentication, and deployment responsibilities without improving the acoustic matching path.

## Decision

Use React, TypeScript, and Vite in `web/` as a static frontend. It calls the public Go `/v1/identify` endpoint directly. Browser code never includes an admin secret. The protected catalog workflow uses one configured admin account, bcrypt password verification, server-side opaque sessions, HttpOnly cookies, CSRF tokens, expiry, and login rate limiting.

CORS is limited to `LYRA_ALLOWED_ORIGIN`, defaulting to the local Vite development origin. Production must set it to the deployed frontend origin.

## Consequences

The frontend has a separate Node build step but no second server-side application. PostgreSQL stores only SHA-256 hashes of random session and CSRF tokens, not the session values themselves. Vite 5 is pinned because the current development environment runs Node 18; upgrade Vite after moving to Node 20.19+.
