# Lyra v1 ExecPlan

## Purpose

Deliver a backend-only, deterministic acoustic-landmark music identification system as a Go modular monolith.

## Scope

Includes phases 0–11 from the bootstrap specification: DSP spike; production foundation; catalog/database; production fingerprinting; async ingestion; matcher; HTTP identify API; evaluation; measured scalability safeguards; production hardening; Docker deployment; release verification. Excludes humming, embeddings, ML, frontend, and distributed fingerprint storage.

## Progress

- [completed] Phase 0: repository context and deterministic DSP spike
- [completed] Phase 1: foundation
- [completed] Phase 2: schema/catalog
- [completed] Phase 3: production landmark-v1
- [completed] Phase 4: ingestion
- [completed] Phase 5: matcher
- [completed] Phase 6: identification API
- [in progress] Phase 7: evaluation
- [completed] Phase 8: scalability hardening
- [completed] Phase 9: production hardening
- [in progress] Phase 10: deployment
- [pending] Phase 11: release verification

## Architecture/Context

Read `AGENTS.md`, `docs/ARCHITECTURE.md`, and `docs/ALGORITHM.md` before changes. One binary supports all modes. PostgreSQL is truth; Valkey is queue/cache; private S3 objects retain only references. Query audio is ephemeral.

## Implementation steps

1. Build deterministic PCM/STFT/peak/landmark/hash extraction and golden tests.
2. Add validated configuration, lifecycle, HTTP health/metrics, Docker/Compose, CI.
3. Add migrations, sqlc query source, Postgres adapters, catalog admin API.
4. Add safe FFmpeg normalization and version-frozen production DSP.
5. Add S3 adapter, Asynq worker, lifecycle, atomic idempotent indexing.
6. Add batched lookup matcher and conservative no-match logic.
7. Add synchronous multipart identification API and contract.
8. Add legal reproducible evaluation and actual reporting.
9. Add measured query/index safeguards and synthetic benchmark.
10. Harden dependencies, auth, rate limiting, traces, shutdown, backup/recovery.
11. Run release gates and clean-environment smoke tests.

## Acceptance criteria

Each phase meets its bootstrap acceptance checks. v1 only completes after all release gates and documented smoke tests pass with real measured evaluation data.

## Verification commands

`make fmt`, `make lint`, `make vet`, `make vuln`, `make test`, `make test-race`, `make test-integration`, `make docker-build`, `make eval`, `make benchmark`, and `make verify` as applicable.

## Surprises & Discoveries

- Go toolchain initially available is 1.22.2; the environment did not complete Gonum/toolchain downloads. Phase 0 therefore uses a tested radix-2 FFT implementation; Phase 3 must switch to Gonum Fourier when dependency resolution is available.

## Decision Log

- landmark-v1 uses the frozen values in `docs/ALGORITHM.md`.
- Match-strength percentages must not be presented as probabilities. `timing_aligned` indicates that an accepted candidate passed temporal-evidence gates; Phase 7 measured data is required before adding calibrated strength levels or changing thresholds.

## Outcomes & Retrospective

Phase 0 completed with deterministic extraction, hash boundary/silence tests, and a committed synthetic golden vector (244 fingerprints; SHA-256 asserted). It was verified with `go test ./...`, `go vet ./...`, `make build`, and a local readiness smoke test.

Phase 2 has its initial migration/schema and a pgx catalog repository. The API integration and migration test remain open.

Phase 2 completed: migrations ran successfully against Compose PostgreSQL and the tagged lifecycle integration test passed. Catalog admin create/list/get/delete endpoints are present. Delete is synchronous only until Phase 4 introduces the deletion worker.

Phase 3 completed with safe no-shell FFmpeg/ffprobe execution, input validation, canonical PCM conversion, and Gonum Fourier. Phase 5 completed with a consumer-owned index contract, one batched hash lookup, offset-neighbourhood voting, deterministic ranking, and conservative seed rejection floors. Phase 4 ingestion remains required before release.

Reference upload endpoint wiring is now present: create track, post multipart reference audio, persist private object and metadata, then enqueue the indexing worker. A full local run awaits MinIO/Valkey image availability in this environment.

Phase 8 began with bounded matcher work: maximum query fingerprints, postings, and candidates are enforced, and the query hash set is sorted for deterministic lookups. A reproducible synthetic benchmark exists. On 2026-08-13, 1,000 synthetic tracks / 4,200 postings / 200 query fingerprints completed in 2 ms in memory and returned a match. This is not a PostgreSQL or audio-accuracy claim.

Hot-hash suppression now uses transactionally refreshed `fingerprint_hash_stats`; the PostgreSQL adapter filters hashes exceeding the configured maximum posting count prior to lookup. Unit tests verify suppression.

Phase 9 began by removing the in-memory serving fallback. `serve` requires configured PostgreSQL, S3-compatible object storage, and the admin key; it verifies the reference bucket at startup. This prevents a process that cannot perform correct identification from reporting ready.

Phase 9 adds panic recovery, restrictive HTTP security headers, and `scripts/backup.sh` with restore guidance in `docs/DEVELOPMENT.md`.

Phase 10 adds `render.yaml`, `docs/DEPLOYMENT.md`, and a real `make docker-build` target. Docker execution has not been verified in this session because the Docker socket is inaccessible to the execution sandbox.

Release preflight ran on 2026-08-13: race tests, vet, and golangci-lint passed. `govulncheck` could not retrieve `vuln.go.dev` due session network restrictions; Docker checks remain blocked by Docker socket access.

Matcher-scoring milestone on 2026-08-17: candidate evidence is now recomputed from the selected ±2-frame offset neighbourhood only. The matcher requires evidence across at least two anchor-frequency bins and rejects tied top-track aligned-hit scores. It records runner-up, query-coverage, span, and frequency-diversity diagnostics in safe identification logs. This is a scoring correction; `landmark-v1` extraction and hash compatibility are unchanged. The Identify UI now labels accepted evidence as `Timing aligned` instead of presenting temporal concentration as percentage confidence. Evaluation reports now include wrong-track matches, false negatives, false positives, and per-condition counts. `make verify`, `make test-race`, `web: npm run lint`, and `web: npm run build` passed. `make lint` could not run because `golangci-lint` is absent from the environment. `make benchmark` reported 1 ms for 1,000 synthetic tracks, 4,200 postings, and 200 query fingerprints; it is not a PostgreSQL or accuracy measurement. A legal corpus and `testdata/manifests/eval.json` are still absent, so `make eval` cannot run.

Bounded microphone-finder milestone on 2026-08-17: the static React client now requests microphone permission only from the public Identify view, records through `MediaRecorder` in browser memory for at most 15 seconds, stops microphone tracks, and submits one temporary capture to the existing identify endpoint. It supports an early user stop and permission/unsupported-browser errors. No streaming backend, persistent query storage, or API change was introduced. `make verify`, `web: npm run lint`, and `web: npm run build` passed. A manual browser microphone smoke test remains pending because this coding session cannot access the user's local listener or browser.

Progressive microphone-finder update on 2026-08-17: the static client now checks the growing in-memory microphone capture at 5, 8, and 11 seconds through the existing synchronous identify endpoint. An accepted deterministic match stops the microphone early. Intermediate no-matches do not interrupt listening; at 15 seconds the client stops capture, makes a final check, and then presents a specific no-match result. This adopts the useful progressive interaction of music-search products without changing Lyra’s scope: it remains exact landmark matching rather than Google’s ML humming/melody embedding system. End-to-end request testing exposed a rate-limit collision from a redundant 14-second probe followed one second later by the final request; the 14-second probe was removed. A temporary full reference indexed to `READY` in 3.841 seconds; its matching 8-second excerpt was accepted in 151 ms. Generated white noise returned no match at 5, 8, 11, and 15 seconds, with the final request completing in 96 ms and no rate-limit response. The temporary track was deleted. `make verify`, `web: npm run lint`, and `web: npm run build` passed. A physical microphone/browser-permission check remains pending because this environment has no graphical browser.

Live-checkpoint observability update on 2026-08-17: browser live requests now carry only their elapsed capture duration in `X-Lyra-Live-Capture-Ms`. CORS explicitly permits the header; the API validates the 1–15,000 ms range and appends `live_capture_ms` to the existing safe `identification_completed` structured log event. This makes the 5/8/11/15-second progression visible with the established `matched`, `reason`, and `duration_ms` fields, without logging audio or other sensitive content. `make verify`, `web: npm run lint`, and `web: npm run build` passed.

Tunnel rate-limit update on 2026-08-17: phone testing through a local Cloudflare Tunnel revealed that the connector presents every request to the API from loopback, causing separate phone actions to share a single rate-limit bucket. The limiter now accepts a syntactically valid `CF-Connecting-IP` only from a loopback peer; network peers cannot spoof the header. Unit coverage verifies direct-header rejection, loopback forwarding, and invalid-header fallback. `make verify`, `web: npm run lint`, and `web: npm run build` passed.

Concurrent local/phone CORS update on 2026-08-17: `LYRA_ALLOWED_ORIGIN` now accepts an explicit comma-separated allowlist so local Vite development and one temporary HTTPS phone-testing origin can run together. CORS returns only the requesting origin when it is listed; unlisted origins remain rejected. Unit coverage verifies both configured origins. `make verify`, `web: npm run lint`, and `web: npm run build` passed.

Complete-WAV live-capture update on 2026-08-17: real-phone testing showed that incremental `MediaRecorder` WebM/MP4 fragments can lack duration metadata, causing FFmpeg to reject them as invalid audio. The live finder now uses Web Audio PCM capture in browser memory and encodes a complete WAV container for each 5/8/11/15-second check. It remains bounded, uses the existing ephemeral endpoint, and never persists query audio. A subsequent real-phone test successfully identified indexed audio. `make verify`, `web: npm run lint`, and `web: npm run build` passed.

False-positive rejection update on 2026-08-18: the supplied unrelated-music query `testdata/queries/kamariya.mp3` was initially accepted as Kalyani with 42 aligned hits, 33 anchors, 0.96% usable-query-anchor coverage, and a runner-up at 40 hits. The matcher now requires at least 2% usable-query-anchor coverage and a three-hit lead over the runner-up. Its safe identification log also records the best rejected candidate's existing aggregate evidence fields, allowing no-match diagnostics without retaining query audio. The query then returned no-match in 156 ms; a temporary one-query `make eval` manifest recorded one correct unrelated-music no-match in 190 ms. `make verify` and `make benchmark` passed (benchmark: 1 ms, 200 query fingerprints, 4,200 synthetic postings, matched=true). Existing saved positive clips currently do not match their catalog entries, so they were not used to make an accuracy claim and must be re-indexed before positive regression evaluation.

Local edge-case evaluation update on 2026-08-18: reset and rebuilt the local PostgreSQL catalog from the Kalyani and Side by Side source MP3s, restoring both tracks to `READY`. This revealed the 5,000-query-fingerprint safeguard was returning no-match for 15–24 second clips; it now deterministically samples across the full query while retaining original anchor frames. The eval adapter now treats `insufficient audio signal` as a valid no-match so silence can be included. A temporary local ten-query smoke manifest produced 7/7 correct matches (clean, short 5-second, quiet, noise, and resampled/low-bitrate phone-like) and 3/3 correct no-matches (Kamariya, silence, white noise), with zero false positives, false negatives, or wrong matches; p50 was 162 ms and p95/p99 182 ms. `make verify` and `make benchmark` passed (1 ms synthetic benchmark). Generated test audio is Git-ignored and the temporary manifest was removed. This materially covers local regressions but is not a legal reproducible development/holdout corpus; covers, real speech, and real microphone captures remain unmeasured.

Live-listening UI update on 2026-08-18: redesigned the public Identify landing screen around a centered listening orb inspired by modern voice interfaces. The orb is a code-native CSS visual, not a generated asset: it receives measured RMS level from the existing Web Audio PCM capture callback, scales its core and animates concentric ripples while capture is active, presents elapsed 15-second progress, and shows a checking state during identification. The same button stops a live capture, the existing upload flow remains below it, and the catalog/admin experience is untouched. `web: npm run lint && npm run build` passed.

Landing-page refinement on 2026-08-18: extended the listening orb into a complete public experience with a centered editorial composition, restrained grid-and-gradient background, glass-like file-upload fallback, simplified catalog navigation, and three concise explanatory steps. The rebuild uses only the existing React/CSS system and local SVG mark; no external images, fonts, API, catalog, or microphone-flow changes were introduced. `web: npm run lint && npm run build` passed.

Fluid orb interaction update on 2026-08-18: removed all default side cards and the waveform. The orb now carries idle, microphone-listening, API-checking/upload-processing, accepted-match, and no-match states; layered CSS gradients, changing border radii, lighting, and rings give it a smooth liquid motion without a rendering dependency. An accepted match displays its title inside the orb and is the sole condition that creates an outward animated line and one real catalog-match card. No song candidates are invented before the API confirms a match; all nonessential motion is disabled for `prefers-reduced-motion`. `web: npm run lint && npm run build` passed.

The user authorized a new frontend phase after the backend work. `web/` is a static React/TypeScript/Vite identification UI. ADR-001 records that Go remains the only backend. The public UI build passed with Node 18/Vite 5.

The frontend now includes the one permitted administrator workflow. `LYRA_ADMIN_USERNAME` and a bcrypt `LYRA_ADMIN_PASSWORD_HASH` configure the account. Successful login generates random opaque session and CSRF tokens; only SHA-256 token hashes are stored in PostgreSQL. The cookie is HttpOnly/SameSite=Lax (and configurable Secure), catalog writes require the CSRF header, sessions expire after 12 hours, and login has a per-IP limit. The static client receives no admin secret and keeps its CSRF token only in memory. `make dev` now starts local infrastructure, applies migrations, and runs API, worker, and Vite together. Go unit/vet and TypeScript lint/production-build checks passed; a real browser login smoke test remains pending.

Development observability now uses a colorized structured `slog` terminal handler; production can set `LYRA_LOG_FORMAT=json`. `LYRA_LOG_LEVEL` controls debug/info/warn/error filtering. A redaction handler protects fields with sensitive names, while service lifecycle, HTTP status/request IDs, authentication results, uploads, indexing jobs, fingerprint counts, and identification results are logged with safe fields. `make dev` starts processes in isolated process groups and tears down API, worker, Vite, and Compose infrastructure on Ctrl-C while preserving named Docker volumes. A Ctrl-C is reported as a successful exit. Make targets build to ignored `bin/lyra`, avoiding generated root binaries.

Brand/product-experience scope: establish a native SVG mark and lockup, document palette/tone/usage in `docs/BRAND.md`, apply the system to the responsive React Identify and Catalog views, and refresh the root README as an organization-facing project page. The work deliberately uses lightweight SVG/CSS rather than a generated raster mark or UI framework; it does not alter fingerprinting, API contracts, or auth behavior.
