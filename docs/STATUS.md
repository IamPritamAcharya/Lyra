# Lyra Status

Current phase: Brand and product-experience phase — native identity system and UI refresh (in progress).

Completed:
- Persistent project context and repository skeleton created.
- Canonical PCM16 mono WAV reader and deterministic landmark-v1 extractor implemented.
- Hash packing boundary tests, deterministic extraction test, and silence rejection test implemented.
- A synthetic deterministic landmark-v1 golden vector (244 packed fingerprints, SHA-256 asserted in tests) is committed.
- Minimal `serve` command, graceful shutdown, health endpoints, and metrics shell implemented.
- Docker/Compose and baseline GitHub Actions workflow created.
- PostgreSQL migration pair, sqlc query definitions, and explicit track lifecycle state machine created.
- pgx/PostgreSQL catalog repository added, including locked transactional lifecycle transitions.
- Admin catalog create/list/get HTTP routes with timing-safe key comparison and contract skeleton added.
- `lyra migrate` now applies golang-migrate migrations; it was executed against local Compose PostgreSQL and created all five expected tables.
- Tagged PostgreSQL catalog lifecycle integration test passed against Compose PostgreSQL.
- Phase 3 completed: safe FFmpeg/ffprobe boundary validates source audio and produces canonical mono 11025 Hz PCM16; DSP now uses Gonum Fourier.
- Deterministic matcher with one batched inverted-index lookup, ±2-frame offset voting, ranking, and no-match/insufficient-signal decisions implemented and unit-tested.
- Phase 4 worker execution mode and Phase 5 matcher are implemented; Phase 6 multipart identify handler is wired to the matcher.
- Phase 6 completed with ephemeral multipart handling, public metadata responses, request IDs, request limits, and per-process identify rate limiting.
- Synthetic matcher benchmark command implemented. Initial in-memory 1,000-track baseline completed; see `docs/BENCHMARKS.md`.
- Prometheus-compatible HTTP and identification counters/latency summaries replace the metrics placeholder.
- Transactionally refreshed hash posting statistics and hot-hash suppression are implemented before PostgreSQL fingerprint lookup.
- Serving now fails fast when PostgreSQL, object storage, or required security configuration is absent instead of exposing an in-memory fallback.
- Panic recovery, restrictive security headers, and PostgreSQL backup/restore guidance are implemented.
- Render web/worker blueprint, deployment environment documentation, and functional `make docker-build` target are implemented.
- Static React/TypeScript/Vite public identification frontend is scaffolded and production-build verified.
- Single-admin browser authentication is implemented: bcrypt password verification, opaque server-side PostgreSQL sessions, HttpOnly cookies, CSRF checks for catalog writes, 12-hour expiry, and per-IP login rate limiting.
- Structured logging is implemented with colorized local text or production JSON, configurable severity, secret-token redaction, request status/request IDs, lifecycle, upload/indexing, matching, and authentication events.
- `make dev` now gracefully stops API, worker, frontend, and Docker infrastructure on `Ctrl-C`; named database/object-storage volumes preserve data.
- MIT licensing and contributor guidelines are published.
- Local Compose includes Adminer at `http://localhost:8081` for PostgreSQL inspection; it is development-only and excluded from deployment.
- `make db-reset CONFIRM=RESET_LYRA_DB` safely gates a destructive local PostgreSQL schema reset and reapplies migrations; it intentionally does not delete MinIO objects.
- Race tests, `go vet`, and golangci-lint have passed in the current environment after unchecked-error fixes.
- Matcher evidence accounting now derives distinct hashes, query anchors, frequency diversity, and alignment span only from the selected temporal-offset neighbourhood. It rejects tied top-track evidence and single-frequency chains instead of selecting an arbitrary candidate. `landmark-v1` hash generation is unchanged.
- The Identify UI now shows the non-probabilistic status `Timing aligned` rather than treating temporal concentration as a percentage confidence. The API retains `confidence` as a documented internal diagnostic and adds `match_strength=timing_aligned` for accepted matches.
- Evaluation reports now separate wrong-track matches from false negatives and false positives, and include sorted per-condition counts. The checked-in manifest is an example only; a legal corpus and active manifest remain required.
- The public Identify view now supports a bounded progressive live microphone finder: browser-memory capture through `getUserMedia`/`MediaRecorder` is checked against the existing identify endpoint at 5, 8, and 11 seconds. An accepted match stops capture early; otherwise capture ends at 15 seconds and receives one final check. It does not create a new backend, object-store query audio, or retain microphone recordings.

Latest evaluation: not yet available; no recognition claims have been measured.

Latest brand verification:
- `web: npm run lint && npm run build` passed after the native SVG identity, responsive UI refresh, and README update on 2026-08-13.
- Brand refinement removes external fonts, repeated backdrop blurs, and glow-heavy effects after local UI responsiveness feedback; the SVG mark was simplified to a minimal lyre/spectral form.

Known issues:
- Docker socket access is unavailable to this coding session, so the Docker image and local full-workflow smoke test cannot be rerun here. `golangci-lint` is not installed in this coding environment, so `make lint` cannot run. `govulncheck`/npm audit are blocked here because they cannot fetch remote vulnerability databases. A legal evaluation corpus remains unimplemented.

Next:
- Build and index a legal reproducible evaluation corpus, run `make eval`, and calibrate matcher evidence gates using its development/holdout results.
- Complete the branded UI/browser smoke test with indexed local references, including the revised `Timing aligned` match presentation, and validate the refreshed README rendering on GitHub.
- Perform a browser microphone permission/capture smoke test on `localhost` and a deployed HTTPS origin, confirming progressive early-match stop, final no-match after 15 seconds, manual stop, rejected permission, and query-audio deletion behavior.

Manual local test data:
- `testdata/audio/side-to-side.mp3` (237.9 seconds) and `testdata/audio/kalyani.mp3` (287.3 seconds) are full local references.
- `testdata/queries/side-to-side-clip.mp3` (24.2 seconds) and `testdata/queries/kalyani-clip.mp3` (14.8 seconds) are their local query clips.
- These manually downloaded files are Git-ignored and must not be committed. The Docker database inspected on 2026-08-13 had no tracks/fingerprints, so the references need uploading again before matching can be evaluated.

Last verification:
- `go test ./...`
- `go vet ./...`
- `make build`
- local `GET /health/ready` smoke test
- `DATABASE_URL=... go test -tags=integration ./...` against local Compose PostgreSQL
- `make benchmark` → 2 ms, 200 query fingerprints, 4,200 in-memory synthetic postings; matched=true.
- `go test -race ./...`, `go vet ./...`, and `golangci-lint run` passed on 2026-08-13.
- `go test ./...`, `go vet ./...`, and `web: npm run lint && npm run build` passed after the admin-auth implementation on 2026-08-13.
- `make dev` startup readiness wait was added after Docker returned before PostgreSQL accepted its first migration connection.
- `make dev` launches Vite using `localhost` to match the deliberately restricted local CORS origin.
- `go test ./...`, `go vet ./...`, and `golangci-lint run` passed after logging and development shutdown changes on 2026-08-13.
- On 2026-08-17, `make verify`, `make test-race`, `web: npm run lint`, and `web: npm run build` passed after matcher-evidence, evaluation-reporting, and Identify UI changes. `make lint` could not run because `golangci-lint` is absent from the environment. `make benchmark` returned `1 ms`, `200` query fingerprints, `4,200` synthetic postings, and `matched=true`; it is still an in-memory synthetic result only. Go emitted a non-fatal module-stat-cache warning for the read-only global cache during that benchmark build.
- On 2026-08-17, `make verify`, `web: npm run lint`, and `web: npm run build` passed after the bounded browser microphone-capture flow was added. A real browser microphone smoke test remains pending because the coding session cannot bind to or inspect the user's local API process.
- On 2026-08-17, `make verify`, `web: npm run lint`, and `web: npm run build` passed after the live finder became progressive: it checks in-memory capture at 5, 8, and 11 seconds, stops on an early accepted match, and otherwise reports no match only after the 15-second limit. End-to-end testing found that a redundant 14-second check caused the final 15-second request to hit the per-IP rate limiter; that check was removed.
- On 2026-08-17, a temporary reference was uploaded, indexed to `READY` in 3.841 seconds, and deleted through the local API. With the corrected 5/8/11/15-second request cadence, an 8-second excerpt returned an accepted match in 151 ms; generated white-noise requests returned no match at every checkpoint, including the final 15-second request (96 ms), without a rate-limit response. `make verify`, `web: npm run lint`, and `web: npm run build` passed. A physical microphone/browser-permission smoke test remains pending because no graphical browser is available in this coding environment.
- Live identify checkpoints now add the safe `live_capture_ms` field to the existing `identification_completed` structured log event. The client sends the elapsed capture duration in a bounded CORS-permitted header; the API accepts only 1–15,000 ms. This provides visible 5/8/11/15-second progression together with `matched`, `reason`, and `duration_ms`, without logging query audio. `make verify`, `web: npm run lint`, and `web: npm run build` passed.
- Development reverse-proxy/tunnel requests now preserve per-client rate limiting: the limiter accepts a validated `CF-Connecting-IP` only when the direct peer is loopback, and ignores that header from network peers. This prevents all phone clients behind a local Cloudflare Tunnel from sharing one login/identify limit. `make verify`, `web: npm run lint`, and `web: npm run build` passed.
- CORS now accepts an explicit comma-separated `LYRA_ALLOWED_ORIGIN` allowlist. This permits local `http://localhost:5173` development and a temporary HTTPS phone-testing origin concurrently; every accepted origin is still reflected individually, and unlisted origins remain rejected. `make verify`, `web: npm run lint`, and `web: npm run build` passed.
- Real-phone testing exposed invalid audio from incremental fragmented `MediaRecorder` containers. The live finder now captures PCM through Web Audio and encodes complete WAV files for every checkpoint, so each identify request has a valid duration-bearing audio container. A subsequent real-phone test successfully identified indexed audio. `make verify`, `web: npm run lint`, and `web: npm run build` passed.
- The matcher now rejects weak or ambiguous candidates with a 2% usable-query-anchor-coverage floor and a three-aligned-hit lead over the runner-up. Rejected best-candidate aggregate evidence is logged safely with the existing identification event, so `no_match` decisions retain aligned-hit, diversity, span, coverage, runner-up, and coherence diagnostics without retaining query audio. The local unrelated-music regression query `testdata/queries/kamariya.mp3` changed from a false Kalyani match (42 aligned hits, 0.96% coverage, runner-up 40) to a no-match in 156 ms. `make eval` using a temporary one-query local manifest reported one correct no-match at 190 ms; `make verify` and `make benchmark` passed. The current saved positive clips return no match against the existing catalog and must be re-indexed before they can serve as positive controls; this single negative is not a calibrated accuracy claim.
