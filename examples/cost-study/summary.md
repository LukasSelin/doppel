### Operating points

| corpus | T | T date | pin date | window commits | functions | threshold | struct-min | reported | top50 | tail | short of controls |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| cobra | `9e1d6f1c2` | 2021-11-16 | 2025-12-03 | 113 | 259 | 0.46 | 0.52 | 229 | 50 | 179 | 0 |
| gin | `ecdbbbe94` | 2024-02-19 | 2026-02-28 | 132 | 417 | 0.42 | 0.50 | 490 | 50 | 440 | 0 |
| prometheus | `e86e4ed87` | 2024-08-16 | 2026-08-17 | 1402 | 4086 | 0.33 | 0.35 | 8910 | 50 | 450 | 0 |
| hugo | `2192cf7ec` | 2024-08-12 | 2026-08-12 | 779 | 4631 | 0.34 | 0.31 | 10745 | 50 | 446 | 0 |
| moby | `02011af7b` | 2023-11-04 | 2025-11-05 | 1885 | 7299 | 0.36 | 0.31 | 16595 | 50 | 450 | 0 |

### Decision (pre-registered: top50, M1 or M2 ratio >= 2.0 with CI lower bound > 1, on >= 4 corpora)

| corpus | M1 ratio [95% CI] | M2 ratio [95% CI] | shows cost | top50 pairs with an M1/M2 event | distinct commits behind them |
| --- | --- | --- | --- | ---: | ---: |
| cobra | 15.00 [0.00, ∞] (266/2000 undefined) | — [—, —] (2000/2000 undefined) | no | 1 | 1 |
| gin | ∞ [∞, ∞] (1/2000 undefined) | 0.00 [0.00, 0.00] (273/2000 undefined) | **yes** | 6 | 1 |
| prometheus | ∞ [∞, ∞] | 6.00 [0.00, ∞] (95/2000 undefined) | **yes** | 17 | 28 |
| hugo | ∞ [∞, ∞] (10/2000 undefined) | ∞ [∞, ∞] (279/2000 undefined) | **yes** | 7 | 6 |
| moby | ∞ [∞, ∞] (96/2000 undefined) | ∞ [∞, ∞] (703/2000 undefined) | **yes** | 4 | 3 |

4 of 5 corpora show cost: **evidence of cost**.

### cobra (50 + 179 treated pairs, 3 controls each)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.020 | 0.001 | 15.00 [0.00, ∞] (266/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| **M2** lagged or unpropagated (share of pairs) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.006 | 0.000 | ∞ [∞, ∞] (733/2000 undefined) |
| any co-change | 0.020 | 0.007 | 3.00 [0.00, ∞] (269/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| synced (>= 2 co-changes) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| lagged | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| unpropagated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.006 | 0.000 | ∞ [∞, ∞] (718/2000 undefined) |
| extracted or consolidated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| both sides edited | 0.080 | 0.420 | 0.19 [0.04, 0.41] | 0.184 | 0.121 | 1.52 [1.14, 2.05] |
| mean edits per side | 0.350 | 3.427 | 0.10 [0.02, 0.24] | 0.905 | 0.970 | 0.93 [0.79, 1.10] |
| a side gone by the pin | 0.560 | 0.153 | 3.65 [2.40, 6.19] | 0.067 | 0.099 | 0.68 [0.32, 1.21] |

### gin (50 + 440 treated pairs, 3 controls each)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.110 | 0.000 | ∞ [∞, ∞] (1/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| **M2** lagged or unpropagated (share of pairs) | 0.000 | 0.013 | 0.00 [0.00, 0.00] (273/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| any co-change | 0.120 | 0.000 | ∞ [∞, ∞] (2/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| synced (>= 2 co-changes) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| lagged | 0.000 | 0.007 | 0.00 [0.00, 0.00] (697/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| unpropagated | 0.000 | 0.007 | 0.00 [0.00, 0.00] (697/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| extracted or consolidated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| both sides edited | 0.220 | 0.040 | 5.50 [2.62, 33.00] | 0.023 | 0.018 | 1.25 [0.53, 2.50] |
| mean edits per side | 0.390 | 0.233 | 1.67 [0.90, 2.84] | 0.102 | 0.093 | 1.10 [0.82, 1.43] |
| a side gone by the pin | 0.000 | 0.047 | 0.00 [0.00, 0.00] | 0.000 | 0.027 | 0.00 [0.00, 0.00] |

### prometheus (50 + 450 treated pairs, 3 controls each)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.256 | 0.000 | ∞ [∞, ∞] | 0.069 | 0.000 | ∞ [∞, ∞] |
| **M2** lagged or unpropagated (share of pairs) | 0.040 | 0.007 | 6.00 [0.00, ∞] (95/2000 undefined) | 0.022 | 0.002 | 10.00 [3.00, ∞] |
| any co-change | 0.320 | 0.000 | ∞ [∞, ∞] | 0.087 | 0.000 | ∞ [∞, ∞] |
| synced (>= 2 co-changes) | 0.120 | 0.000 | ∞ [∞, ∞] (2/2000 undefined) | 0.027 | 0.000 | ∞ [∞, ∞] |
| lagged | 0.020 | 0.000 | ∞ [∞, ∞] (732/2000 undefined) | 0.009 | 0.000 | ∞ [∞, ∞] (41/2000 undefined) |
| unpropagated | 0.020 | 0.007 | 3.00 [0.00, ∞] (254/2000 undefined) | 0.013 | 0.002 | 6.00 [1.29, ∞] |
| extracted or consolidated | 0.120 | 0.000 | ∞ [∞, ∞] (2/2000 undefined) | 0.011 | 0.000 | ∞ [∞, ∞] (14/2000 undefined) |
| both sides edited | 0.460 | 0.253 | 1.82 [1.19, 2.88] | 0.171 | 0.105 | 1.63 [1.28, 2.06] |
| mean edits per side | 1.280 | 1.177 | 1.09 [0.70, 1.62] | 0.539 | 0.669 | 0.81 [0.63, 1.03] |
| a side gone by the pin | 0.180 | 0.133 | 1.35 [0.60, 2.79] | 0.102 | 0.153 | 0.67 [0.48, 0.89] |

### hugo (50 + 446 treated pairs, 3 controls each)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.080 | 0.000 | ∞ [∞, ∞] (10/2000 undefined) | 0.012 | 0.000 | ∞ [∞, ∞] (5/2000 undefined) |
| **M2** lagged or unpropagated (share of pairs) | 0.040 | 0.000 | ∞ [∞, ∞] (279/2000 undefined) | 0.002 | 0.000 | ∞ [∞, ∞] (747/2000 undefined) |
| any co-change | 0.100 | 0.000 | ∞ [∞, ∞] (9/2000 undefined) | 0.013 | 0.000 | ∞ [∞, ∞] (5/2000 undefined) |
| synced (>= 2 co-changes) | 0.060 | 0.000 | ∞ [∞, ∞] (108/2000 undefined) | 0.002 | 0.000 | ∞ [∞, ∞] (775/2000 undefined) |
| lagged | 0.020 | 0.000 | ∞ [∞, ∞] (709/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| unpropagated | 0.020 | 0.000 | ∞ [∞, ∞] (731/2000 undefined) | 0.002 | 0.000 | ∞ [∞, ∞] (698/2000 undefined) |
| extracted or consolidated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| both sides edited | 0.180 | 0.093 | 1.93 [0.83, 3.86] | 0.047 | 0.045 | 1.05 [0.64, 1.61] |
| mean edits per side | 0.460 | 0.453 | 1.01 [0.55, 1.72] | 0.173 | 0.216 | 0.80 [0.59, 1.05] |
| a side gone by the pin | 0.140 | 0.193 | 0.72 [0.27, 1.32] | 0.108 | 0.235 | 0.46 [0.33, 0.61] |

### moby (50 + 450 treated pairs, 3 controls each)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.060 | 0.000 | ∞ [∞, ∞] (96/2000 undefined) | 0.035 | 0.000 | ∞ [∞, ∞] |
| **M2** lagged or unpropagated (share of pairs) | 0.020 | 0.000 | ∞ [∞, ∞] (703/2000 undefined) | 0.007 | 0.000 | ∞ [∞, ∞] (115/2000 undefined) |
| any co-change | 0.060 | 0.000 | ∞ [∞, ∞] (84/2000 undefined) | 0.038 | 0.000 | ∞ [∞, ∞] |
| synced (>= 2 co-changes) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.004 | 0.000 | ∞ [∞, ∞] (271/2000 undefined) |
| lagged | 0.020 | 0.000 | ∞ [∞, ∞] (733/2000 undefined) | 0.007 | 0.000 | ∞ [∞, ∞] (100/2000 undefined) |
| unpropagated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| extracted or consolidated | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) | 0.000 | 0.000 | — [—, —] (2000/2000 undefined) |
| both sides edited | 0.200 | 0.127 | 1.58 [0.71, 3.25] | 0.069 | 0.084 | 0.82 [0.54, 1.15] |
| mean edits per side | 0.260 | 0.580 | 0.45 [0.22, 0.80] | 0.184 | 0.402 | 0.46 [0.29, 0.68] |
| a side gone by the pin | 0.220 | 0.140 | 1.57 [0.72, 3.00] | 0.127 | 0.146 | 0.87 [0.65, 1.12] |

### Pooled (resampled within corpus)

| metric | top50 treated | control | ratio [95% CI] | tail treated | control | ratio [95% CI] |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| **M1** normalised co-change (cochanges / min edits) | 0.105 | 0.000 | 394.50 [114.00, ∞] | 0.027 | 0.000 | ∞ [∞, ∞] |
| **M2** lagged or unpropagated (share of pairs) | 0.020 | 0.004 | 5.00 [1.00, ∞] (1/2000 undefined) | 0.008 | 0.001 | 15.00 [4.71, ∞] |
| any co-change | 0.124 | 0.001 | 93.00 [25.00, ∞] | 0.032 | 0.000 | ∞ [∞, ∞] |
| synced (>= 2 co-changes) | 0.036 | 0.000 | ∞ [∞, ∞] | 0.008 | 0.000 | ∞ [∞, ∞] |
| lagged | 0.012 | 0.001 | 9.00 [0.00, ∞] (29/2000 undefined) | 0.004 | 0.000 | ∞ [∞, ∞] (3/2000 undefined) |
| unpropagated | 0.008 | 0.003 | 3.00 [0.00, ∞] (25/2000 undefined) | 0.004 | 0.001 | 8.00 [2.00, ∞] |
| extracted or consolidated | 0.024 | 0.000 | ∞ [∞, ∞] (4/2000 undefined) | 0.003 | 0.000 | ∞ [∞, ∞] (13/2000 undefined) |
| both sides edited | 0.228 | 0.187 | 1.22 [0.92, 1.60] | 0.088 | 0.069 | 1.28 [1.09, 1.49] |
| mean edits per side | 0.548 | 1.174 | 0.47 [0.35, 0.62] | 0.310 | 0.403 | 0.77 [0.67, 0.87] |
| a side gone by the pin | 0.220 | 0.133 | 1.65 [1.24, 2.18] | 0.083 | 0.137 | 0.61 [0.51, 0.71] |

### By pair kind (pooled, ranks 1-500)

| kind | pairs | M1 treated / control | M1 ratio [95% CI] | M2 treated / control | M2 ratio [95% CI] | any co-change ratio [95% CI] |
| --- | ---: | --- | --- | --- | --- | --- |
| different calls | 117 | 0.019 / 0.000 | ∞ [∞, ∞] (56/2000 undefined) | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | ∞ [∞, ∞] (50/2000 undefined) |
| diverged copy | 10 | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | — [—, —] (2000/2000 undefined) |
| interface implementations | 410 | 0.027 / 0.000 | ∞ [∞, ∞] | 0.029 / 0.002 | 18.00 [5.40, ∞] | ∞ [∞, ∞] |
| mirror operations | 24 | 0.056 / 0.000 | ∞ [∞, ∞] (258/2000 undefined) | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | ∞ [∞, ∞] (238/2000 undefined) |
| none | 1322 | 0.048 / 0.000 | 961.14 [286.75, ∞] | 0.005 / 0.001 | 5.25 [1.33, ∞] | 228.00 [69.00, ∞] |
| subsystem copies | 8 | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | — [—, —] (2000/2000 undefined) |
| thin wrappers | 324 | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | 0.003 / 0.000 | ∞ [∞, ∞] (744/2000 undefined) | — [—, —] (2000/2000 undefined) |

### By locality (pooled, ranks 1-500)

| locality | pairs | M1 treated / control | M1 ratio [95% CI] | M2 treated / control | M2 ratio [95% CI] | any co-change ratio [95% CI] |
| --- | ---: | --- | --- | --- | --- | --- |
| cross | 57 | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | 0.000 / 0.000 | — [—, —] (2000/2000 undefined) | — [—, —] (2000/2000 undefined) |
| file | 1315 | 0.036 / 0.000 | ∞ [∞, ∞] | 0.002 / 0.001 | 2.00 [0.00, ∞] (12/2000 undefined) | ∞ [∞, ∞] |
| package | 485 | 0.053 / 0.000 | 386.25 [111.37, ∞] | 0.025 / 0.002 | 12.00 [3.50, ∞] | 90.00 [27.00, ∞] |
| top | 358 | 0.015 / 0.000 | ∞ [∞, ∞] (1/2000 undefined) | 0.017 / 0.000 | ∞ [∞, ∞] (5/2000 undefined) | ∞ [∞, ∞] (2/2000 undefined) |
