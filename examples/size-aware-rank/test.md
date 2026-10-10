### Units

| unit | T | T date | end date | window commits replayed | sweep scope | functions at T | calibration | union | unresolved |
| --- | --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: |
| cobra o2 | `de2d9c4ec` | 2017-12-04 | 2021-11-16 | 139 | commit | 221 | threshold 0.45 struct-min 0.50 | 998 | 0 |
| cobra o3 | `fae133554` | 2013-11-05 | 2017-12-04 | 311 | commit | 48 | declined (only 406 eligible shape null pairs (need 1000)); static defaults | 133 | 0 |
| gin o3 | `094b3fdb3` | 2020-02-26 | 2022-02-14 | 126 | commit | 372 | threshold 0.42 struct-min 0.53 | 1473 | 0 |
| prometheus o2 | `2d9b3f6e9` | 2022-08-18 | 2024-08-16 | 1257 | commit | 2945 | threshold 0.35 struct-min 0.34 | 15115 | 0 |
| prometheus o3 | `edae2efe6` | 2020-08-18 | 2022-08-18 | 733 | commit | 2245 | threshold 0.35 struct-min 0.35 | 12268 | 0 |
| hugo o2 | `0e0fb1b64` | 2022-08-12 | 2024-08-12 | 775 | commit | 3985 | threshold 0.34 struct-min 0.32 | 22427 | 0 |
| hugo o3 | `5f4259014` | 2020-08-12 | 2022-08-12 | 652 | commit | 3479 | threshold 0.35 struct-min 0.32 | 19072 | 0 |
| moby o2 | `c09789c11` | 2021-11-03 | 2023-11-04 | 2300 | commit | 7683 | threshold 0.36 struct-min 0.30 | 36656 | 0 |
| moby o3 | `65523469c` | 2019-11-07 | 2021-11-03 | 2291 | commit | 5662 | threshold 0.36 struct-min 0.30 | 29145 | 0 |

### M1@100 per unit

Each cell is M1@100 over the list's own top 100 (n in brackets when shorter), the pairs with any event, the distinct commits behind them, and the median smaller side in lines.

| method | cobra o2 | cobra o3 | gin o3 | prometheus o2 | prometheus o3 | hugo o2 | hugo o3 | moby o2 | moby o3 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0.035 · 4 · 2 · 17 | 0.029 · 7 · 6 · 9 | 0.035 · 11 · 3 · 6 | 0.146 · 20 · 27 · 18 | 0.147 · 18 · 20 · 15 | 0.015 · 5 · 6 · 13 | 0.032 · 6 · 7 · 25 | 0.030 · 3 · 3 · 23 | 0.045 · 5 · 3 · 12 |
| dupl (t=100, default) | 0.000 (3) · 0 · 0 · 24 | — | 0.000 (16) · 0 · 0 · 3 | 0.183 (15) · 3 · 5 · 21 | 0.133 (15) · 2 · 2 · 20 | 0.250 (4) · 1 · 1 · 26 | 0.500 (3) · 2 · 3 · 26 | 0.032 · 4 · 5 · 8 | 0.000 (59) · 0 · 0 · 15 |
| dupl (t=50) | 0.000 (9) · 0 · 0 · 10 | 1.000 (1) · 1 · 1 · 10 | 0.000 (17) · 0 · 0 · 3 | 0.118 (74) · 9 · 12 · 17 | 0.095 (50) · 5 · 6 · 14 | 0.022 (92) · 2 · 4 · 10 | 0.020 (76) · 2 · 3 · 10 | 0.057 · 7 · 8 · 17 | 0.012 · 2 · 3 · 16 |
| token clones (≥100 nodes) | 0.000 · 0 · 0 · 24 | 0.000 (10) · 0 · 0 · 24 | 0.000 · 0 · 0 · 27 | 0.149 · 21 · 27 · 27 | 0.134 · 18 · 20 · 26 | 0.000 · 3 · 3 · 32 | 0.015 · 3 · 5 · 32 | 0.010 · 1 · 1 · 23 | 0.035 · 4 · 2 · 20 |
| token clones (≥50 nodes) | 0.030 · 3 · 1 · 18 | 0.082 (55) · 7 · 2 · 13 | 0.005 · 1 · 1 · 13 | 0.128 · 15 · 20 · 18 | 0.100 · 14 · 16 · 18 | 0.010 · 4 · 4 · 31 | 0.015 · 3 · 5 · 31 | 0.040 · 4 · 4 · 15 | 0.040 · 4 · 2 · 16 |
| token clones (no floor) | 0.010 · 1 · 1 · 6 | 0.029 · 6 · 6 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 | 0.000 · 0 · 0 · 3 |
| code-shape (≥50 nodes) | 0.030 · 3 · 1 · 16 | 0.082 (55) · 7 · 2 · 13 | 0.005 · 1 · 1 · 14 | 0.120 · 13 · 18 · 18 | 0.122 · 16 · 17 · 16 | 0.010 · 3 · 4 · 25 | 0.032 · 5 · 5 · 23 | 0.040 · 4 · 4 · 16 | 0.030 · 3 · 1 · 16 |
| variant: K-size | 0.035 · 4 · 2 · 20 | 0.029 · 7 · 6 · 9 | 0.035 · 11 · 3 · 6 | 0.171 · 25 · 39 · 27 | 0.168 · 20 · 21 · 21 | 0.015 · 5 · 6 · 31 | 0.032 · 7 · 9 · 31 | 0.065 · 8 · 8 · 27 | 0.048 · 7 · 5 · 22 |
| variant: K-subtree | 0.035 · 4 · 2 · 19 | 0.029 · 7 · 6 · 9 | 0.035 · 11 · 3 · 6 | 0.159 · 22 · 28 · 21 | 0.168 · 20 · 21 · 21 | 0.015 · 5 · 6 · 29 | 0.032 · 7 · 9 · 31 | 0.055 · 6 · 6 · 26 | 0.045 · 6 · 4 · 19 |
| variant: K-floor | 0.035 · 4 · 2 · 19 | 0.029 · 7 · 6 · 9 | 0.035 · 11 · 3 · 6 | 0.166 · 22 · 29 · 18 | 0.147 · 18 · 20 · 18 | 0.015 · 5 · 6 · 19 | 0.032 · 6 · 7 · 29 | 0.030 · 3 · 3 · 23 | 0.045 · 5 · 3 · 15 |

### Primary: each variant against doppel, M1@100

Per unit: the difference at matched depth (each list's top min(nX, nY, 100)), its paired-clustered 95% CI (2000 replicates), ✓/✗ when the CI excludes 0, and in brackets the independent event clusters in the frame; · marks a unit below the floor of 3, which is not pooled; @n a depth below 100. Pooled D is the mean over corpora of the mean over each corpus's counting units, with the CI read off the replicate-wise pooled replicates.

| comparison | cobra o2 | cobra o3 | gin o3 | prometheus o2 | prometheus o3 | hugo o2 | hugo o3 | moby o2 | moby o3 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | --- | --- |
| variant: K-size − doppel | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.000 [+0.000, +0.000] (2 ·) | +0.024 [-0.010, +0.060] (20) | +0.020 [+0.001, +0.049] ✓ (12) | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.035 [+0.010, +0.070] ✓ (7) | +0.003 [+0.000, +0.010] (4) | 6 · 4 | +0.0103 [+0.0038, +0.0175] | variant: K-size beats doppel |
| variant: K-subtree − doppel | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.000 [+0.000, +0.000] (2 ·) | +0.012 [+0.001, +0.033] ✓ (17) | +0.020 [+0.001, +0.048] ✓ (12) | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.025 [+0.000, +0.055] (6) | +0.000 [+0.000, +0.000] (3) | 6 · 4 | +0.0072 [+0.0025, +0.0126] | variant: K-subtree beats doppel |
| variant: K-floor − doppel | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.000 [+0.000, +0.000] (2 ·) | +0.020 [+0.000, +0.050] (18) | +0.000 [+0.000, +0.000] (11) | +0.000 [+0.000, +0.000] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.000 [+0.000, +0.000] (3) | +0.000 [+0.000, +0.000] (3) | 6 · 4 | +0.0025 [+0.0000, +0.0062] | not distinguishable |

### The gap to the clone detectors

Every method against each detector, at matched depth, under the same rule.

| comparison | cobra o2 | cobra o3 | gin o3 | prometheus o2 | prometheus o3 | hugo o2 | hugo o3 | moby o2 | moby o3 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | --- | --- |
| doppel − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | — | +0.000 [+0.000, +0.000] (0 ·) @16 | +0.083 [-0.133, +0.333] (6) @15 | +0.100 [-0.124, +0.300] (4) @15 | -0.250 [-0.750, +0.000] (1 ·) @4 | -0.333 [-0.667, +0.000] (2 ·) @3 | -0.002 [-0.038, +0.040] (6) | +0.076 [+0.000, +0.183] (3) @59 | 4 · 2 | +0.0645 [-0.0152, +0.1462] | not distinguishable |
| variant: K-size − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | — | +0.000 [+0.000, +0.000] (0 ·) @16 | +0.100 [-0.133, +0.333] (7) @15 | +0.117 [-0.098, +0.304] (5) @15 | -0.250 [-0.750, +0.000] (1 ·) @4 | -0.500 [-1.000, +0.000] (2 ·) @3 | +0.033 [-0.003, +0.080] (8) | +0.082 [+0.006, +0.193] ✓ (4) @59 | 4 · 2 | +0.0830 [+0.0028, +0.1628] | variant: K-size beats dupl (t=100, default) |
| variant: K-subtree − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | — | +0.000 [+0.000, +0.000] (0 ·) @16 | +0.117 [-0.050, +0.300] (6) @15 | +0.000 [-0.167, +0.117] (3) @15 | -0.250 [-0.750, +0.000] (1 ·) @4 | -0.500 [-1.000, +0.000] (2 ·) @3 | +0.023 [-0.007, +0.063] (7) | +0.076 [+0.000, +0.183] (3) @59 | 4 · 2 | +0.0541 [-0.0092, +0.1164] | not distinguishable |
| variant: K-floor − dupl (t=100, default) | +0.000 [+0.000, +0.000] (0 ·) @3 | — | +0.000 [+0.000, +0.000] (0 ·) @16 | +0.083 [-0.167, +0.333] (6) @15 | +0.100 [-0.100, +0.300] (4) @15 | -0.250 [-0.750, +0.000] (1 ·) @4 | -0.333 [-0.667, +0.000] (2 ·) @3 | -0.002 [-0.038, +0.038] (6) | +0.076 [+0.000, +0.185] (3) @59 | 4 · 2 | +0.0645 [-0.0222, +0.1455] | not distinguishable |
| doppel − dupl (t=50) | +0.111 [+0.000, +0.333] (1 ·) @9 | +0.000 [+0.000, +0.000] (1 ·) @1 | +0.000 [+0.000, +0.000] (0 ·) @17 | +0.066 [+0.007, +0.129] ✓ (17) @74 | +0.120 [-0.003, +0.248] (10) @50 | -0.005 [-0.033, +0.016] (3) @92 | +0.022 [+0.000, +0.053] (3) @76 | -0.027 [-0.078, +0.028] (10) | +0.033 [-0.013, +0.099] (4) | 6 · 3 | +0.0349 [+0.0070, +0.0617] | doppel beats dupl (t=50) |
| variant: K-size − dupl (t=50) | +0.111 [+0.000, +0.333] (1 ·) @9 | +0.000 [+0.000, +0.000] (1 ·) @1 | +0.000 [+0.000, +0.000] (0 ·) @17 | +0.068 [+0.010, +0.129] ✓ (17) @74 | +0.160 [+0.019, +0.293] ✓ (11) @50 | -0.005 [-0.033, +0.016] (3) @92 | +0.022 [+0.000, +0.053] (3) @76 | +0.008 [-0.050, +0.070] (12) | +0.037 [-0.010, +0.105] (5) | 6 · 3 | +0.0483 [+0.0181, +0.0771] | variant: K-size beats dupl (t=50) |
| variant: K-subtree − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @9 | +0.000 [+0.000, +0.000] (1 ·) @1 | +0.000 [+0.000, +0.000] (0 ·) @17 | +0.059 [+0.017, +0.109] ✓ (15) @74 | +0.160 [+0.026, +0.294] ✓ (11) @50 | -0.005 [-0.033, +0.016] (3) @92 | +0.022 [+0.000, +0.053] (3) @76 | -0.002 [-0.055, +0.050] (11) | +0.033 [-0.013, +0.105] (4) | 6 · 3 | +0.0446 [+0.0169, +0.0720] | variant: K-subtree beats dupl (t=50) |
| variant: K-floor − dupl (t=50) | +0.000 [+0.000, +0.000] (0 ·) @9 | +0.000 [+0.000, +0.000] (1 ·) @1 | +0.000 [+0.000, +0.000] (0 ·) @17 | +0.053 [-0.003, +0.112] (16) @74 | +0.100 [-0.026, +0.218] (9) @50 | -0.005 [-0.033, +0.016] (3) @92 | +0.022 [+0.000, +0.053] (3) @76 | -0.027 [-0.082, +0.027] (10) | +0.033 [-0.013, +0.105] (4) | 6 · 3 | +0.0293 [+0.0023, +0.0544] | variant: K-floor beats dupl (t=50) |
| doppel − token clones (≥100 nodes) | +0.035 [+0.000, +0.096] (2 ·) | +0.139 [+0.000, +0.339] (3) @10 | +0.035 [+0.000, +0.083] (2 ·) | -0.002 [-0.062, +0.055] (21) | +0.013 [-0.067, +0.095] (13) | +0.015 [+0.000, +0.040] (2 ·) | +0.017 [+0.000, +0.041] (3) | +0.020 [+0.000, +0.050] (3) | +0.010 [+0.000, +0.031] (3) | 6 · 4 | +0.0441 [+0.0048, +0.0970] | doppel beats token clones (≥100 nodes) |
| variant: K-size − token clones (≥100 nodes) | +0.035 [+0.000, +0.096] (2 ·) | +0.264 [+0.000, +0.546] (2 ·) @10 | +0.035 [+0.000, +0.083] (2 ·) | +0.022 [-0.023, +0.065] (21) | +0.033 [-0.047, +0.114] (13) | +0.015 [+0.000, +0.040] (2 ·) | +0.017 [+0.000, +0.041] (3) | +0.055 [+0.015, +0.100] ✓ (7) | +0.013 [+0.000, +0.037] (4) | 5 · 3 | +0.0261 [+0.0068, +0.0448] | variant: K-size beats token clones (≥100 nodes) |
| variant: K-subtree − token clones (≥100 nodes) | +0.035 [+0.000, +0.096] (2 ·) | +0.275 [+0.000, +0.554] (2 ·) @10 | +0.035 [+0.000, +0.081] (2 ·) | +0.010 [-0.046, +0.063] (21) | +0.033 [-0.043, +0.117] (13) | +0.015 [+0.000, +0.040] (2 ·) | +0.017 [+0.000, +0.041] (3) | +0.045 [+0.010, +0.090] ✓ (6) | +0.010 [+0.000, +0.031] (3) | 5 · 3 | +0.0219 [+0.0035, +0.0418] | variant: K-subtree beats token clones (≥100 nodes) |
| variant: K-floor − token clones (≥100 nodes) | +0.035 [+0.000, +0.096] (2 ·) | +0.275 [+0.000, +0.554] (2 ·) @10 | +0.035 [+0.000, +0.083] (2 ·) | +0.018 [-0.030, +0.067] (21) | +0.013 [-0.066, +0.097] (13) | +0.015 [+0.000, +0.040] (2 ·) | +0.017 [+0.000, +0.041] (3) | +0.020 [+0.000, +0.050] (3) | +0.010 [+0.000, +0.031] (3) | 5 · 3 | +0.0157 [-0.0015, +0.0343] | not distinguishable |
| doppel − token clones (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.006] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.019 [-0.061, +0.082] (19) | +0.047 [+0.009, +0.101] ✓ (11) | +0.005 [-0.000, +0.015] (2 ·) | +0.017 [+0.000, +0.040] (3) | -0.010 [-0.050, +0.030] (6) | +0.005 [-0.025, +0.035] (4) | 6 · 4 | +0.0045 [-0.0128, +0.0223] | not distinguishable |
| variant: K-size − token clones (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.005] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.043 [-0.033, +0.104] (21) | +0.068 [+0.013, +0.135] ✓ (12) | +0.005 [-0.000, +0.015] (2 ·) | +0.017 [+0.000, +0.041] (3) | +0.025 [-0.030, +0.080] (10) | +0.008 [-0.022, +0.039] (5) | 6 · 4 | +0.0149 [-0.0027, +0.0335] | not distinguishable |
| variant: K-subtree − token clones (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.007] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.031 [-0.050, +0.099] (20) | +0.068 [+0.013, +0.131] ✓ (12) | +0.005 [-0.000, +0.015] (2 ·) | +0.017 [+0.000, +0.042] (3) | +0.015 [-0.035, +0.065] (9) | +0.005 [-0.026, +0.034] (4) | 6 · 4 | +0.0117 [-0.0066, +0.0304] | not distinguishable |
| variant: K-floor − token clones (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.007] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.039 [-0.035, +0.100] (20) | +0.047 [+0.009, +0.100] ✓ (11) | +0.005 [-0.000, +0.015] (2 ·) | +0.017 [+0.000, +0.042] (3) | -0.010 [-0.050, +0.030] (6) | +0.005 [-0.026, +0.035] (4) | 6 · 4 | +0.0070 [-0.0097, +0.0250] | not distinguishable |
| doppel − code-shape (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.007] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.026 [-0.013, +0.064] (17) | +0.025 [+0.002, +0.053] ✓ (11) | +0.005 [+0.000, +0.015] (2 ·) | +0.000 [+0.000, +0.000] (3) | -0.010 [-0.050, +0.030] (6) | +0.015 [+0.000, +0.041] (3) | 6 · 4 | -0.0003 [-0.0133, +0.0138] | not distinguishable |
| variant: K-size − code-shape (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.006] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.051 [+0.009, +0.095] ✓ (20) | +0.045 [+0.007, +0.091] ✓ (12) | +0.005 [+0.000, +0.015] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.025 [-0.025, +0.080] (10) | +0.018 [+0.000, +0.043] (4) | 6 · 4 | +0.0101 [-0.0039, +0.0252] | not distinguishable |
| variant: K-subtree − code-shape (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.006] (3) @55 | +0.030 [+0.000, +0.066] (2 ·) | +0.039 [+0.000, +0.079] ✓ (18) | +0.045 [+0.007, +0.090] ✓ (12) | +0.005 [+0.000, +0.015] (2 ·) | +0.000 [+0.000, +0.000] (3) | +0.015 [-0.035, +0.065] (9) | +0.015 [+0.000, +0.040] (3) | 6 · 4 | +0.0069 [-0.0076, +0.0235] | not distinguishable |
| variant: K-floor − code-shape (≥50 nodes) | +0.005 [+0.000, +0.015] (2 ·) | -0.029 [-0.067, +0.007] (3) @55 | +0.030 [+0.000, +0.063] (2 ·) | +0.046 [+0.004, +0.091] ✓ (19) | +0.025 [+0.002, +0.053] ✓ (11) | +0.005 [+0.000, +0.015] (2 ·) | +0.000 [+0.000, +0.000] (3) | -0.010 [-0.050, +0.030] (6) | +0.015 [+0.000, +0.040] (3) | 6 · 4 | +0.0022 [-0.0106, +0.0166] | not distinguishable |

### Gap change per detector

M1(variant) − M1(doppel) at the detector's matched depth, pooled over every unit where the detector lists a pair (no cluster floor): how far the variant moves doppel's difference with that detector. Then the two verdicts against the detector.

| variant | detector | gap change [95% CI] | doppel vs detector | variant vs detector |
| --- | --- | --- | --- | --- |
| variant: K-size | dupl (t=100, default) | -0.0093 [-0.0357, +0.0163] | not distinguishable | variant: K-size beats dupl (t=100, default) |
| variant: K-size | dupl (t=50) | +0.0081 [-0.0014, +0.0176] | doppel beats dupl (t=50) | variant: K-size beats dupl (t=50) |
| variant: K-size | token clones (≥100 nodes) | +0.0208 [+0.0012, +0.0318] | doppel beats token clones (≥100 nodes) | variant: K-size beats token clones (≥100 nodes) |
| variant: K-size | token clones (≥50 nodes) | +0.0083 [+0.0030, +0.0140] | not distinguishable | not distinguishable |
| variant: K-size | code-shape (≥50 nodes) | +0.0083 [+0.0030, +0.0140] | not distinguishable | not distinguishable |
| variant: K-subtree | dupl (t=100, default) | -0.0208 [-0.0402, -0.0005] | not distinguishable | not distinguishable |
| variant: K-subtree | dupl (t=50) | -0.0053 [-0.0139, +0.0031] | doppel beats dupl (t=50) | variant: K-subtree beats dupl (t=50) |
| variant: K-subtree | token clones (≥100 nodes) | +0.0193 [+0.0014, +0.0296] | doppel beats token clones (≥100 nodes) | variant: K-subtree beats token clones (≥100 nodes) |
| variant: K-subtree | token clones (≥50 nodes) | +0.0057 [+0.0020, +0.0101] | not distinguishable | not distinguishable |
| variant: K-subtree | code-shape (≥50 nodes) | +0.0057 [+0.0020, +0.0101] | not distinguishable | not distinguishable |
| variant: K-floor | dupl (t=100, default) | +0.0000 [+0.0000, +0.0000] | not distinguishable | not distinguishable |
| variant: K-floor | dupl (t=50) | -0.0145 [-0.0180, -0.0111] | doppel beats dupl (t=50) | variant: K-floor beats dupl (t=50) |
| variant: K-floor | token clones (≥100 nodes) | +0.0156 [-0.0014, +0.0247] | doppel beats token clones (≥100 nodes) | not distinguishable |
| variant: K-floor | token clones (≥50 nodes) | +0.0020 [+0.0000, +0.0050] | not distinguishable | not distinguishable |
| variant: K-floor | code-shape (≥50 nodes) | +0.0020 [+0.0000, +0.0050] | not distinguishable | not distinguishable |

### Criteria (a) and (b)

(a) the variant beats doppel: the pooled lower bound above 0. (b) the gap closes: the gap change is above 0 for at least 4 of the 5 detectors, and against no detector is the variant's verdict worse than doppel's.

| variant | (a) | gap change > 0 | verdicts worse | (b) |
| --- | --- | ---: | ---: | --- |
| variant: K-size | **yes** | 4 of 5 | 0 | **yes** |
| variant: K-subtree | **yes** | 3 of 5 | 0 | no |
| variant: K-floor | no | 3 of 5 | 1 | no |

### Small-family cost

Pairs in doppel's top 100 whose smaller side has fewer than 23 lines at T (below a clone detector's floor), and how many of them each variant's top 100 drops; with an event in brackets.

| variant | cobra o2 | cobra o3 | gin o3 | prometheus o2 | prometheus o3 | hugo o2 | hugo o3 | moby o2 | moby o3 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| doppel: small pairs in top 100 | 54 (4) | 93 (5) | 96 (7) | 67 (7) | 72 (6) | 60 (1) | 49 (0) | 47 (1) | 79 (0) |
| variant: K-size: dropped | 2 (0) | 0 (0) | 6 (0) | 28 (1) | 21 (0) | 36 (0) | 30 (0) | 24 (0) | 48 (0) |
| variant: K-subtree: dropped | 5 (0) | 0 (0) | 3 (0) | 18 (0) | 18 (0) | 28 (0) | 24 (0) | 17 (0) | 31 (0) |
| variant: K-floor: dropped | 5 (0) | 0 (0) | 3 (0) | 10 (0) | 14 (0) | 15 (0) | 17 (0) | 1 (0) | 5 (0) |

### Overlap of each top 100 with doppel's (descriptive)

| method | cobra o2 | cobra o3 | gin o3 | prometheus o2 | prometheus o3 | hugo o2 | hugo o3 | moby o2 | moby o3 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| dupl (t=100, default) | 3 | 0 | 1 | 12 | 11 | 4 | 3 | 18 | 15 |
| dupl (t=50) | 7 | 1 | 2 | 46 | 29 | 30 | 30 | 23 | 26 |
| token clones (≥100 nodes) | 15 | 6 | 4 | 38 | 38 | 27 | 39 | 46 | 27 |
| token clones (≥50 nodes) | 40 | 20 | 7 | 53 | 67 | 37 | 43 | 40 | 37 |
| token clones (no floor) | 51 | 27 | 25 | 1 | 0 | 1 | 1 | 0 | 0 |
| code-shape (≥50 nodes) | 40 | 20 | 8 | 59 | 69 | 50 | 58 | 45 | 41 |
| variant: K-size | 98 | 100 | 94 | 72 | 79 | 64 | 70 | 76 | 52 |
| variant: K-subtree | 95 | 100 | 96 | 80 | 82 | 72 | 76 | 82 | 67 |
| variant: K-floor | 95 | 100 | 96 | 90 | 86 | 85 | 83 | 98 | 93 |
