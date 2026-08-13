# Benchmarks

## Discipline

Benchmark results are environment-specific measurements, not capacity claims. Re-run the exact command after a meaningful change and record the corpus/index characteristics with the result.

## Synthetic matcher baseline

Command run on 2026-08-13:

```bash
./lyra benchmark --synthetic-tracks=1000
```

Result:

```json
{"SyntheticTracks":1000,"QueryFingerprints":200,"PostingCount":4200,"DurationMS":2,"Matched":true}
```

This is a deterministic in-memory synthetic matcher baseline only. It does not measure PostgreSQL lookup latency, database index size, network overhead, or real audio accuracy.
