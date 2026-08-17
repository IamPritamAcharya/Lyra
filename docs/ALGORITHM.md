# landmark-v1 algorithm

`landmark-v1` is immutable and currently maps to numeric version `1`. It uses mono signed-16-bit PCM at 11025 Hz, 512-point Hann-window STFTs, and a 256-sample hop. It retains deterministic local spectral peaks, applies frame and global density limits, pairs each anchor with up to three later peaks in a constrained target zone, and stores `(hash, anchor_frame)`.

Hash payload layout is 20 bits: `f1` (8 bits) at bits 12–19, signed `df` (6-bit two's complement) at bits 6–11, and `dt` (6 bits) at bits 0–5. `df` is limited to -31..31 and `dt` to 2..63. Higher bits remain zero. Any change to these values, peak behavior, target zone, FFT settings, or match tolerance requires a new algorithm version.

Identification looks up deduplicated hashes in one batch, votes by `(track, database_frame-query_frame)`, and sums a ±2-frame offset neighbourhood. The winning offset's evidence is then calculated only from matches inside that neighbourhood. Hashes, query anchors, frequency bins, and alignment span outside the winning neighbourhood do not count toward acceptance.

The current deterministic matcher queries the index once per deduplicated hash set and ranks candidates by aligned hits, distinct hashes, alignment span, frequency diversity, then temporal concentration. Its seed acceptance floors are six aligned hits, three distinct hashes, three query anchors, two distinct anchor-frequency bins, and a four-frame alignment span. If two tracks tie on aligned-hit score, the matcher returns no match rather than choosing an arbitrary winner. These are not accuracy claims and must be calibrated by Phase 7 evaluation.

`alignment_coherence` is an internal temporal-concentration diagnostic: aligned hit pairings divided by all raw hit pairings for the candidate. Repeated sections and common hashes can reduce it even for a correct match, so it is not a probability of correctness and must not be presented as user-facing confidence. The API instead reports `match_strength=timing_aligned` when a candidate passes the evidence gates.

For scalability, indexing refreshes `fingerprint_hash_stats` transactionally. The PostgreSQL index filters query hashes whose posting counts exceed the matcher’s default threshold of 10,000 before looking up postings. This is stop-word suppression: it preserves discriminative hashes while bounding work from common patterns.

## Evaluation requirements

Before changing landmark extraction, target-zone values, hash packing, or the seed acceptance floors, evaluate against a legal reproducible corpus with separate development and holdout partitions. The manifest must label query condition and expected outcome. At minimum, report clean excerpts; codec/resampling/EQ/noise variants; real microphone captures; and unrelated music, speech, silence, and noise. `lyra eval` reports correct matches, wrong-track matches, false negatives, false positives, latency percentiles, and per-condition counts. Use these measured distributions to calibrate evidence gates; do not tune them from an individual clip.
