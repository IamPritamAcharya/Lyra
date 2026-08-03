# landmark-v1 algorithm

`landmark-v1` is immutable and currently maps to numeric version `1`. It uses mono signed-16-bit PCM at 11025 Hz, 512-point Hann-window STFTs, and a 256-sample hop. It retains deterministic local spectral peaks, applies frame and global density limits, pairs each anchor with up to three later peaks in a constrained target zone, and stores `(hash, anchor_frame)`.

Hash payload layout is 20 bits: `f1` (8 bits) at bits 12–19, signed `df` (6-bit two's complement) at bits 6–11, and `dt` (6 bits) at bits 0–5. `df` is limited to -31..31 and `dt` to 2..63. Higher bits remain zero. Any change to these values, peak behavior, target zone, FFT settings, or match tolerance requires a new algorithm version.

Identification will lookup deduplicated hashes in batches, vote by `(track, database_frame-query_frame)`, sum a ±2-frame offset neighbourhood, and reject candidates lacking sufficient aligned distinct evidence. These initial thresholds are deliberately documented as seeds until a legal evaluation corpus measures them.

The current deterministic matcher queries the index once per deduplicated hash set and ranks candidates by aligned hits, distinct hashes, alignment span, then coherence. Its seed acceptance floors are six aligned hits, three distinct hashes, and three query anchors. These are not accuracy claims and must be calibrated by Phase 7 evaluation.
