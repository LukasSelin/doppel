### Units

| unit | T | T date | end date | window commits replayed | sweep scope | functions at T | calibration | union | unresolved |
| --- | --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: |
| cobra o1 | `9e1d6f1c2` | 2021-11-16 | 2025-12-03 | 113 | commit | 259 | threshold 0.46 struct-min 0.52 | 1337 | 0 |
| gin o1 | `ecdbbbe94` | 2024-02-19 | 2026-02-28 | 133 | commit | 417 | threshold 0.42 struct-min 0.50 | 1916 | 0 |
| prometheus o1 | `e86e4ed87` | 2024-08-16 | 2026-08-17 | 1454 | commit | 4086 | threshold 0.33 struct-min 0.35 | 22072 | 0 |
| hugo o1 | `2192cf7ec` | 2024-08-12 | 2026-08-12 | 807 | commit | 4631 | threshold 0.34 struct-min 0.31 | 25469 | 0 |
| moby o1 | `02011af7b` | 2023-11-04 | 2025-11-05 | 2361 | commit | 7299 | threshold 0.36 struct-min 0.31 | 36846 | 0 |

### M1@100 per unit

Each cell is M1@100 over the list's own top 100 (n in brackets when shorter), the pairs with any event, the distinct commits behind them, and the median smaller side in lines.

| method | cobra o1 | gin o1 | prometheus o1 | hugo o1 | moby o1 |
| --- | ---: | ---: | ---: | ---: | ---: |
| doppel | 0.010 · 1 · 1 · 12 | 0.065 · 7 · 2 · 6 | 0.216 · 33 · 47 · 27 | 0.050 · 8 · 7 · 11 | 0.040 · 6 · 10 · 23 |
| dupl (t=100, default) | 0.000 (3) · 0 · 0 · 24 | 0.000 (16) · 0 · 0 · 3 | 0.154 (66) · 17 · 28 · 19 | 0.000 (1) · 0 · 0 · 26 | 0.090 · 10 · 6 · 11 |
| dupl (t=50) | 0.000 (7) · 0 · 0 · 10 | 0.000 (21) · 0 · 0 · 3 | 0.209 · 29 · 46 · 21 | 0.000 (88) · 0 · 0 · 10 | 0.105 · 12 · 9 · 17 |
| token clones (≥100 nodes) | 0.002 · 1 · 1 · 26 | 0.000 · 2 · 1 · 26 | 0.184 · 29 · 44 · 31 | 0.000 · 1 · 2 · 32 | 0.000 · 4 · 10 · 25 |
| token clones (≥50 nodes) | 0.012 · 2 · 2 · 13 | 0.000 · 2 · 1 · 15 | 0.159 · 21 · 23 · 19 | 0.000 · 1 · 2 · 31 | 0.090 · 10 · 4 · 15 |
| token clones (no floor) | 0.030 · 3 · 1 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 |
| code-shape (≥50 nodes) | 0.012 · 2 · 2 · 14 | 0.000 · 2 · 1 · 16 | 0.179 · 27 · 38 · 19 | 0.013 · 3 · 4 · 23 | 0.090 · 9 · 2 · 15 |
| variant: K-size | 0.012 · 2 · 2 · 17 | 0.065 · 7 · 2 · 6 | 0.268 · 44 · 59 · 35 | 0.065 · 11 · 11 · 31 | 0.057 · 10 · 18 · 31 |
| variant: K-subtree | 0.010 · 1 · 1 · 12 | 0.065 · 7 · 2 · 6 | 0.251 · 39 · 53 · 32 | 0.065 · 11 · 11 · 29 | 0.113 · 15 · 16 · 26 |
| variant: K-floor | 0.010 · 1 · 1 · 12 | 0.065 · 7 · 2 · 6 | 0.215 · 33 · 44 · 29 | 0.055 · 9 · 8 · 17 | 0.050 · 7 · 11 · 23 |

### Primary: each variant against doppel, M1@100

Per unit: the difference at matched depth (each list's top min(nX, nY, 100)), its paired-clustered 95% CI (2000 replicates), ✓/✗ when the CI excludes 0, and in brackets the independent event clusters in the frame; · marks a unit below the floor of 3, which is not pooled; @n a depth below 100. Pooled D is the mean over corpora of the mean over each corpus's counting units, with the CI read off the replicate-wise pooled replicates.

| comparison | cobra o1 | gin o1 | prometheus o1 | hugo o1 | moby o1 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | --- | --- | --- | --- | ---: | --- | --- |
| variant: K-size − doppel | +0.002 [+0.000, +0.006] (2 ·) | +0.000 [+0.000, +0.000] (2 ·) | +0.051 [+0.010, +0.094] ✓ (31) | +0.015 [+0.000, +0.040] (5) | +0.017 [+0.001, +0.042] ✓ (5) | 3 · 3 | +0.0279 [+0.0114, +0.0452] | variant: K-size beats doppel |
| variant: K-subtree − doppel | +0.000 [+0.000, +0.000] (1 ·) | +0.000 [+0.000, +0.000] (2 ·) | +0.035 [+0.008, +0.067] ✓ (28) | +0.015 [+0.000, +0.039] (5) | +0.073 [+0.003, +0.178] ✓ (5) | 3 · 3 | +0.0410 [+0.0118, +0.0784] | variant: K-subtree beats doppel |
| variant: K-floor − doppel | +0.000 [+0.000, +0.000] (1 ·) | +0.000 [+0.000, +0.000] (2 ·) | -0.001 [-0.008, +0.003] (26) | +0.005 [+0.000, +0.015] (4) | +0.010 [+0.000, +0.030] (3) | 3 · 3 | +0.0047 [-0.0013, +0.0127] | not distinguishable |

### The gap to the clone detectors

Every method against each detector, at matched depth, under the same rule.

| comparison | cobra o1 | gin o1 | prometheus o1 | hugo o1 | moby o1 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | --- | --- | --- | --- | ---: | --- | --- |
| doppel − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | +0.156 [+0.000, +0.375] (1 ·) @16 | +0.113 [+0.028, +0.208] ✓ (23) @66 | +0.000 [+0.000, +0.000] (0 ·) @1 | -0.050 [-0.161, +0.024] (3) | 2 · 2 | +0.0316 [-0.0410, +0.0966] | not distinguishable |
| variant: K-size − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | +0.344 [+0.000, +0.635] (1 ·) @16 | +0.152 [+0.073, +0.243] ✓ (26) @66 | +0.000 [+0.000, +0.000] (0 ·) @1 | -0.033 [-0.153, +0.050] (6) | 2 · 2 | +0.0597 [-0.0131, +0.1275] | not distinguishable |
| variant: K-subtree − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | +0.219 [+0.000, +0.477] (1 ·) @16 | +0.166 [+0.079, +0.250] ✓ (26) @66 | +0.000 [+0.000, +0.000] (0 ·) @1 | +0.023 [+0.002, +0.054] ✓ (5) | 2 · 2 | +0.0945 [+0.0502, +0.1396] | variant: K-subtree beats dupl (t=100, default) |
| variant: K-floor − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | +0.219 [+0.000, +0.477] (1 ·) @16 | +0.108 [+0.023, +0.200] ✓ (23) @66 | +0.000 [+0.000, +0.000] (0 ·) @1 | -0.040 [-0.156, +0.040] (4) | 2 · 2 | +0.0340 [-0.0404, +0.0996] | not distinguishable |
| doppel − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @7 | +0.119 [+0.000, +0.300] (1 ·) @21 | +0.007 [-0.034, +0.047] (27) | +0.045 [+0.000, +0.100] (3) @88 | -0.065 [-0.179, +0.018] (5) | 3 · 3 | -0.0041 [-0.0487, +0.0337] | not distinguishable |
| variant: K-size − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @7 | +0.262 [+0.000, +0.532] (1 ·) @21 | +0.059 [+0.005, +0.111] ✓ (32) | +0.074 [+0.012, +0.149] ✓ (5) @88 | -0.048 [-0.165, +0.037] (8) | 3 · 3 | +0.0282 [-0.0211, +0.0727] | not distinguishable |
| variant: K-subtree − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @7 | +0.167 [+0.000, +0.389] (1 ·) @21 | +0.042 [-0.005, +0.090] (29) | +0.074 [+0.012, +0.149] ✓ (5) @88 | +0.008 [-0.027, +0.045] (7) | 3 · 3 | +0.0413 [+0.0139, +0.0750] | variant: K-subtree beats dupl (t=50) |
| variant: K-floor − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @7 | +0.167 [+0.000, +0.389] (1 ·) @21 | +0.006 [-0.034, +0.046] (27) | +0.057 [+0.000, +0.134] (3) @88 | -0.055 [-0.178, +0.030] (6) | 3 · 3 | +0.0027 [-0.0450, +0.0442] | not distinguishable |
| doppel − token clones (≥100 nodes) | +0.008 [-0.006, +0.030] (2 ·) | +0.065 [+0.000, +0.177] (2 ·) | +0.032 [-0.016, +0.083] (28) | +0.050 [+0.000, +0.113] (3) | +0.040 [+0.000, +0.107] (2 ·) | 2 · 2 | +0.0412 [+0.0059, +0.0834] | doppel beats token clones (≥100 nodes) |
| variant: K-size − token clones (≥100 nodes) | +0.010 [+0.000, +0.030] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.084 [+0.044, +0.128] ✓ (30) | +0.065 [+0.010, +0.132] ✓ (5) | +0.057 [+0.007, +0.126] ✓ (5) | 3 · 3 | +0.0686 [+0.0387, +0.1039] | variant: K-size beats token clones (≥100 nodes) |
| variant: K-subtree − token clones (≥100 nodes) | +0.008 [-0.006, +0.030] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.067 [+0.016, +0.119] ✓ (29) | +0.065 [+0.010, +0.132] ✓ (5) | +0.113 [+0.011, +0.235] ✓ (5) | 3 · 3 | +0.0818 [+0.0361, +0.1305] | variant: K-subtree beats token clones (≥100 nodes) |
| variant: K-floor − token clones (≥100 nodes) | +0.008 [-0.006, +0.030] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.032 [-0.016, +0.083] (28) | +0.055 [+0.005, +0.123] ✓ (4) | +0.050 [+0.000, +0.115] (3) | 3 · 3 | +0.0455 [+0.0139, +0.0797] | variant: K-floor beats token clones (≥100 nodes) |
| doppel − token clones (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.057 [+0.005, +0.112] ✓ (26) | +0.050 [+0.000, +0.119] (3) | -0.050 [-0.161, +0.021] (3) | 3 · 3 | +0.0190 [-0.0259, +0.0600] | not distinguishable |
| variant: K-size − token clones (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.108 [+0.040, +0.175] ✓ (31) | +0.065 [+0.010, +0.132] ✓ (5) | -0.033 [-0.150, +0.048] (6) | 3 · 3 | +0.0469 [-0.0005, +0.0912] | not distinguishable |
| variant: K-subtree − token clones (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.092 [+0.033, +0.151] ✓ (27) | +0.065 [+0.010, +0.132] ✓ (5) | +0.023 [+0.002, +0.055] ✓ (5) | 3 · 3 | +0.0600 [+0.0308, +0.0907] | variant: K-subtree beats token clones (≥50 nodes) |
| variant: K-floor − token clones (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.056 [+0.001, +0.108] ✓ (25) | +0.055 [+0.005, +0.119] ✓ (4) | -0.040 [-0.159, +0.040] (4) | 3 · 3 | +0.0237 [-0.0233, +0.0642] | not distinguishable |
| doppel − code-shape (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.038 [-0.005, +0.082] (27) | +0.037 [+0.000, +0.080] (3) | -0.050 [-0.167, +0.024] (3) | 3 · 3 | +0.0082 [-0.0377, +0.0438] | not distinguishable |
| variant: K-size − code-shape (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.089 [+0.027, +0.151] ✓ (32) | +0.052 [+0.010, +0.095] ✓ (5) | -0.033 [-0.149, +0.048] (6) | 3 · 3 | +0.0360 [-0.0106, +0.0749] | not distinguishable |
| variant: K-subtree − code-shape (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.072 [+0.015, +0.126] ✓ (29) | +0.052 [+0.010, +0.098] ✓ (5) | +0.023 [+0.002, +0.056] ✓ (5) | 3 · 3 | +0.0492 [+0.0251, +0.0749] | variant: K-subtree beats code-shape (≥50 nodes) |
| variant: K-floor − code-shape (≥50 nodes) | -0.002 [-0.006, +0.000] (2 ·) | +0.065 [+0.000, +0.168] (2 ·) | +0.037 [-0.007, +0.082] (27) | +0.042 [+0.005, +0.086] ✓ (4) | -0.040 [-0.156, +0.041] (4) | 3 · 3 | +0.0129 [-0.0301, +0.0500] | not distinguishable |

### Gap change per detector

M1(variant) − M1(doppel) at the detector's matched depth, pooled over every unit where the detector lists a pair (no cluster floor): how far the variant moves doppel's difference with that detector. Then the two verdicts against the detector.

| variant | detector | gap change [95% CI] | doppel vs detector | variant vs detector |
| --- | --- | --- | --- | --- |
| variant: K-size | dupl (t=100, default) | +0.0488 [+0.0020, +0.0702] | not distinguishable | not distinguishable |
| variant: K-size | dupl (t=50) | +0.0480 [+0.0111, +0.0695] | not distinguishable | not distinguishable |
| variant: K-size | token clones (≥100 nodes) | +0.0171 [+0.0074, +0.0277] | doppel beats token clones (≥100 nodes) | variant: K-size beats token clones (≥100 nodes) |
| variant: K-size | token clones (≥50 nodes) | +0.0171 [+0.0074, +0.0277] | not distinguishable | not distinguishable |
| variant: K-size | code-shape (≥50 nodes) | +0.0171 [+0.0074, +0.0277] | not distinguishable | not distinguishable |
| variant: K-subtree | dupl (t=100, default) | +0.0377 [+0.0087, +0.0623] | not distinguishable | variant: K-subtree beats dupl (t=100, default) |
| variant: K-subtree | dupl (t=50) | +0.0368 [+0.0114, +0.0609] | not distinguishable | variant: K-subtree beats dupl (t=50) |
| variant: K-subtree | token clones (≥100 nodes) | +0.0246 [+0.0071, +0.0470] | doppel beats token clones (≥100 nodes) | variant: K-subtree beats token clones (≥100 nodes) |
| variant: K-subtree | token clones (≥50 nodes) | +0.0246 [+0.0071, +0.0470] | not distinguishable | variant: K-subtree beats token clones (≥50 nodes) |
| variant: K-subtree | code-shape (≥50 nodes) | +0.0246 [+0.0071, +0.0470] | not distinguishable | variant: K-subtree beats code-shape (≥50 nodes) |
| variant: K-floor | dupl (t=100, default) | +0.0135 [-0.0019, +0.0231] | not distinguishable | not distinguishable |
| variant: K-floor | dupl (t=50) | +0.0136 [-0.0001, +0.0240] | not distinguishable | not distinguishable |
| variant: K-floor | token clones (≥100 nodes) | +0.0028 [-0.0008, +0.0076] | doppel beats token clones (≥100 nodes) | variant: K-floor beats token clones (≥100 nodes) |
| variant: K-floor | token clones (≥50 nodes) | +0.0028 [-0.0008, +0.0076] | not distinguishable | not distinguishable |
| variant: K-floor | code-shape (≥50 nodes) | +0.0028 [-0.0008, +0.0076] | not distinguishable | not distinguishable |

### Criteria (a) and (b)

(a) the variant beats doppel: the pooled lower bound above 0. (b) the gap closes: the gap change is above 0 for at least 4 of the 5 detectors, and against no detector is the variant's verdict worse than doppel's.

| variant | (a) | gap change > 0 | verdicts worse | (b) |
| --- | --- | ---: | ---: | --- |
| variant: K-size | **yes** | 5 of 5 | 0 | **yes** |
| variant: K-subtree | **yes** | 5 of 5 | 0 | **yes** |
| variant: K-floor | no | 5 of 5 | 0 | **yes** |

### Small-family cost

Pairs in doppel's top 100 whose smaller side has fewer than 23 lines at T (below a clone detector's floor), and how many of them each variant's top 100 drops; with an event in brackets.

| variant | cobra o1 | gin o1 | prometheus o1 | hugo o1 | moby o1 |
| --- | --- | --- | --- | --- | --- |
| doppel: small pairs in top 100 | 56 (1) | 99 (7) | 42 (6) | 61 (1) | 50 (3) |
| variant: K-size: dropped | 5 (0) | 3 (0) | 23 (1) | 40 (0) | 28 (0) |
| variant: K-subtree: dropped | 3 (0) | 2 (0) | 17 (1) | 30 (0) | 21 (0) |
| variant: K-floor: dropped | 3 (0) | 2 (0) | 6 (1) | 15 (0) | 7 (0) |

### Overlap of each top 100 with doppel's (descriptive)

| method | cobra o1 | gin o1 | prometheus o1 | hugo o1 | moby o1 |
| --- | ---: | ---: | ---: | ---: | ---: |
| dupl (t=100, default) | 3 | 1 | 30 | 1 | 22 |
| dupl (t=50) | 5 | 3 | 52 | 32 | 25 |
| token clones (≥100 nodes) | 14 | 1 | 60 | 25 | 43 |
| token clones (≥50 nodes) | 29 | 4 | 59 | 27 | 44 |
| token clones (no floor) | 28 | 21 | 1 | 0 | 0 |
| code-shape (≥50 nodes) | 29 | 4 | 66 | 47 | 50 |
| variant: K-size | 95 | 97 | 77 | 60 | 72 |
| variant: K-subtree | 95 | 98 | 80 | 70 | 79 |
| variant: K-floor | 95 | 98 | 93 | 85 | 93 |
