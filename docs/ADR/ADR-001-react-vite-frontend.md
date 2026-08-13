# ADR-001: Use a static React/Vite frontend

## Status

Accepted — 2026-08-13.

## Context

Lyra now needs a browser interface for the public identification flow. The Go modular monolith remains the only backend. Adding a Node/NestJS backend would duplicate API, authentication, and deployment responsibilities without improving the acoustic matching path.

## Decision

Use React, TypeScript, and Vite in `web/` as a static frontend. It calls the public Go `/v1/identify` endpoint directly. Browser code never includes `LYRA_ADMIN_API_KEY`; protected catalog administration remains API/Postman-only until proper browser authentication is designed.

CORS is limited to `LYRA_ALLOWED_ORIGIN`, defaulting to the local Vite development origin. Production must set it to the deployed frontend origin.

## Consequences

The frontend has a separate Node build step but no second server-side application. Vite 5 is pinned because the current development environment runs Node 18; upgrade Vite after moving to Node 20.19+.
