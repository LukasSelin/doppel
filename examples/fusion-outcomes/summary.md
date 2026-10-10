### Validity: input lists against the clone-outcome study's

| corpus | method | pairs | identical, in order |
| --- | --- | ---: | --- |
| cobra | doppel | 500 | yes |
| cobra | dupl (t=100, default) | 3 | yes |
| cobra | dupl (t=50) | 7 | yes |
| cobra | token clones (≥50 nodes) | 500 | yes |
| gin | doppel | 500 | yes |
| gin | dupl (t=100, default) | 16 | yes |
| gin | dupl (t=50) | 21 | yes |
| gin | token clones (≥50 nodes) | 500 | yes |
| prometheus | doppel | 500 | yes |
| prometheus | dupl (t=100, default) | 66 | yes |
| prometheus | dupl (t=50) | 216 | yes |
| prometheus | token clones (≥50 nodes) | 500 | yes |
| hugo | doppel | 500 | yes |
| hugo | dupl (t=100, default) | 1 | yes |
| hugo | dupl (t=50) | 88 | yes |
| hugo | token clones (≥50 nodes) | 500 | yes |
| moby | doppel | 500 | yes |
| moby | dupl (t=100, default) | 500 | yes |
| moby | dupl (t=50) | 500 | yes |
| moby | token clones (≥50 nodes) | 500 | yes |

### Operating points

| corpus | T | T date | pin date | window commits | functions at T | calibration | union | listed units unresolved |
| --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: |
| cobra | `9e1d6f1c2` | 2021-11-16 | 2025-12-03 | 111 | 259 | threshold 0.46 struct-min 0.52 | 1337 | 0 |
| gin | `ecdbbbe94` | 2024-02-19 | 2026-02-28 | 131 | 417 | threshold 0.42 struct-min 0.50 | 1916 | 0 |
| prometheus | `e86e4ed87` | 2024-08-16 | 2026-08-17 | 1351 | 4086 | threshold 0.33 struct-min 0.35 | 22072 | 0 |
| hugo | `2192cf7ec` | 2024-08-12 | 2026-08-12 | 512 | 4631 | threshold 0.34 struct-min 0.31 | 25469 | 0 |
| moby | `02011af7b` | 2023-11-04 | 2025-11-05 | 1594 | 7299 | threshold 0.36 struct-min 0.31 | 36846 | 0 |

### Rates per method

M1 is mean co-changes per edit of the less-edited side; M2 the share with a lagged or unpropagated event; E the share with any event; C the distinct commits behind those events; M1 clustered divides each co-change commit among the pairs of the list it touches. Pooled random holds three lists, so its pair counts are three times the depth.

#### cobra

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0100** | 0.000 | 0.010 | 1 | 0.0100 | 0.0200 | 0.020 | 0.0030 | 0.007 |
| dupl (t=100, default) | 0 | 3 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 7 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0120** | 0.000 | 0.020 | 2 | 0.0120 | 0.0200 | 0.020 | 0.0025 | 0.003 |
| RRF(doppel, dupl (t=100, default)) | 0 | 100 | **0.0100** | 0.000 | 0.010 | 1 | 0.0100 | 0.0200 | 0.020 | 0.0030 | 0.007 |
| RRF(doppel, dupl (t=50)) | 0 | 100 | **0.0100** | 0.000 | 0.010 | 1 | 0.0100 | 0.0200 | 0.020 | 0.0030 | 0.007 |
| RRF(doppel, token clones (≥50 nodes)) | 0 | 100 | **0.0120** | 0.000 | 0.020 | 2 | 0.0120 | 0.0200 | 0.020 | 0.0025 | 0.005 |

#### gin

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0650** | 0.000 | 0.070 | 2 | 0.0192 | 0.1300 | 0.140 | 0.0000 | 0.000 |
| dupl (t=100, default) | 0 | 16 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 21 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0000** | 0.020 | 0.020 | 1 | 0.0000 | 0.0000 | 0.040 | 0.0000 | 0.000 |
| RRF(doppel, dupl (t=100, default)) | 0 | 100 | **0.0650** | 0.000 | 0.070 | 2 | 0.0192 | 0.1100 | 0.120 | 0.0000 | 0.000 |
| RRF(doppel, dupl (t=50)) | 0 | 100 | **0.0650** | 0.000 | 0.070 | 2 | 0.0192 | 0.0700 | 0.080 | 0.0000 | 0.000 |
| RRF(doppel, token clones (≥50 nodes)) | 0 | 100 | **0.0550** | 0.020 | 0.080 | 2 | 0.0092 | 0.0500 | 0.080 | 0.0025 | 0.003 |

#### prometheus

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.2163** | 0.040 | 0.330 | 47 | 0.2063 | 0.2560 | 0.420 | 0.0535 | 0.087 |
| dupl (t=100, default) | 0 | 66 | **0.1540** | 0.045 | 0.258 | 28 | 0.1470 | 0.2033 | 0.340 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 100 | **0.2092** | 0.040 | 0.290 | 46 | 0.1923 | 0.2233 | 0.360 | 0.0445 | 0.069 |
| token clones (≥50 nodes) | 0 | 100 | **0.1593** | 0.010 | 0.210 | 23 | 0.1393 | 0.1520 | 0.160 | 0.0649 | 0.105 |
| RRF(doppel, dupl (t=100, default)) | 0 | 100 | **0.1980** | 0.040 | 0.320 | 45 | 0.1833 | 0.2633 | 0.420 | 0.0556 | 0.087 |
| RRF(doppel, dupl (t=50)) | 0 | 100 | **0.2025** | 0.040 | 0.300 | 47 | 0.1856 | 0.3183 | 0.460 | 0.0595 | 0.098 |
| RRF(doppel, token clones (≥50 nodes)) | 0 | 100 | **0.1830** | 0.030 | 0.280 | 42 | 0.1730 | 0.2333 | 0.340 | 0.0593 | 0.100 |

#### hugo

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0550** | 0.020 | 0.090 | 8 | 0.0333 | 0.0900 | 0.160 | 0.0112 | 0.015 |
| dupl (t=100, default) | 0 | 1 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 88 | **0.0114** | 0.000 | 0.011 | 1 | 0.0114 | 0.0200 | 0.020 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0000** | 0.010 | 0.010 | 2 | 0.0000 | 0.0000 | 0.000 | 0.0150 | 0.025 |
| RRF(doppel, dupl (t=100, default)) | 0 | 100 | **0.0550** | 0.020 | 0.090 | 8 | 0.0333 | 0.0900 | 0.160 | 0.0112 | 0.015 |
| RRF(doppel, dupl (t=50)) | 0 | 100 | **0.0450** | 0.020 | 0.080 | 8 | 0.0317 | 0.0467 | 0.100 | 0.0138 | 0.018 |
| RRF(doppel, token clones (≥50 nodes)) | 0 | 100 | **0.0450** | 0.020 | 0.080 | 8 | 0.0317 | 0.0600 | 0.120 | 0.0163 | 0.018 |

#### moby

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0400** | 0.020 | 0.060 | 10 | 0.0200 | 0.0600 | 0.080 | 0.0397 | 0.048 |
| dupl (t=100, default) | 0 | 100 | **0.0900** | 0.010 | 0.100 | 6 | 0.0200 | 0.1800 | 0.200 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 100 | **0.1050** | 0.010 | 0.120 | 10 | 0.0350 | 0.2000 | 0.200 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0900** | 0.010 | 0.100 | 4 | 0.0200 | 0.1200 | 0.120 | 0.0125 | 0.022 |
| RRF(doppel, dupl (t=100, default)) | 0 | 100 | **0.0900** | 0.020 | 0.110 | 10 | 0.0200 | 0.1800 | 0.200 | 0.0108 | 0.015 |
| RRF(doppel, dupl (t=50)) | 0 | 100 | **0.1000** | 0.010 | 0.110 | 9 | 0.0300 | 0.2000 | 0.220 | 0.0159 | 0.028 |
| RRF(doppel, token clones (≥50 nodes)) | 0 | 100 | **0.0900** | 0.010 | 0.100 | 6 | 0.0200 | 0.1200 | 0.120 | 0.0247 | 0.035 |

### Primary comparison: M1@100, doppel minus baseline

| baseline | cobra | gin | prometheus | hugo | moby | doppel wins | baseline wins | verdict |
| --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| dupl (t=100, default) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0200, +0.1200] ✓ | +0.0623 [-0.0417, +0.1661] | +0.0550 [+0.0200, +0.0983] ✓ | -0.0500 [-0.1200, +0.0200] | 2 | 0 | not distinguishable |
| dupl (t=50) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0072 [-0.0980, +0.1077] | +0.0436 [-0.0002, +0.0933] | -0.0650 [-0.1350, +0.0050] | 1 | 0 | not distinguishable |
| token clones (≥50 nodes) | -0.0020 [-0.0320, +0.0280] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0570 [-0.0403, +0.1600] | +0.0550 [+0.0183, +0.1017] ✓ | -0.0500 [-0.1200, +0.0200] | 2 | 0 | not distinguishable |
| RRF(doppel, dupl (t=100, default)) | +0.0000 [-0.0300, +0.0300] | +0.0000 [-0.0700, +0.0650] | +0.0183 [-0.0875, +0.1172] | +0.0000 [-0.0600, +0.0533] | -0.0500 [-0.1200, +0.0200] | 0 | 0 | not distinguishable |
| RRF(doppel, dupl (t=50)) | +0.0000 [-0.0300, +0.0300] | +0.0000 [-0.0700, +0.0650] | +0.0138 [-0.0907, +0.1173] | +0.0100 [-0.0450, +0.0683] | -0.0600 [-0.1300, +0.0100] | 0 | 0 | not distinguishable |
| RRF(doppel, token clones (≥50 nodes)) | -0.0020 [-0.0300, +0.0280] | +0.0100 [-0.0500, +0.0750] | +0.0333 [-0.0727, +0.1433] | +0.0100 [-0.0417, +0.0667] | -0.0500 [-0.1200, +0.0200] | 0 | 0 | not distinguishable |

✓ doppel's CI lower bound is above 0 on that corpus; ✗ the upper bound is below 0. A verdict needs 3 corpora one way and none the other.

### Fusion: M1@100, fused minus input

| fused | input | cobra | gin | prometheus | hugo | moby | fused wins | input wins | verdict |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| RRF(doppel, dupl (t=100, default)) | doppel | +0.0000 [-0.0300, +0.0300] | +0.0000 [-0.0700, +0.0650] | -0.0183 [-0.1240, +0.0908] | +0.0000 [-0.0583, +0.0583] | +0.0500 [-0.0200, +0.1200] | 0 | 0 | not distinguishable |
| RRF(doppel, dupl (t=100, default)) | dupl (t=100, default) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0440 [-0.0666, +0.1481] | +0.0550 [+0.0183, +0.0967] ✓ | +0.0000 [-0.0800, +0.0800] | 2 | 0 | not distinguishable |
| RRF(doppel, dupl (t=50)) | doppel | +0.0000 [-0.0300, +0.0300] | +0.0000 [-0.0650, +0.0650] | -0.0138 [-0.1223, +0.0917] | -0.0100 [-0.0633, +0.0450] | +0.0600 [-0.0100, +0.1300] | 0 | 0 | not distinguishable |
| RRF(doppel, dupl (t=50)) | dupl (t=50) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0250, +0.1150] ✓ | -0.0067 [-0.1115, +0.0990] | +0.0336 [-0.0108, +0.0770] | -0.0050 [-0.0900, +0.0800] | 1 | 0 | not distinguishable |
| RRF(doppel, token clones (≥50 nodes)) | doppel | +0.0020 [-0.0280, +0.0300] | -0.0100 [-0.0700, +0.0550] | -0.0333 [-0.1320, +0.0683] | -0.0100 [-0.0633, +0.0433] | +0.0500 [-0.0200, +0.1200] | 0 | 0 | not distinguishable |
| RRF(doppel, token clones (≥50 nodes)) | token clones (≥50 nodes) | +0.0000 [-0.0300, +0.0300] | +0.0550 [+0.0150, +0.1050] ✓ | +0.0237 [-0.0733, +0.1190] | +0.0450 [+0.0133, +0.0833] ✓ | +0.0000 [-0.0800, +0.0800] | 2 | 0 | not distinguishable |

✓ the fused list's CI lower bound is above 0 on that corpus; ✗ the upper bound is below 0. A verdict needs 3 corpora one way and none the other.

### Overlap, size and event counts at 100

Overlap is the number of a method's top 100 also in doppel's top 100. Median smallest side is in lines at T. Pairs with an event is a count, not a share.

| corpus | method | n@100 | in doppel's top 100 | median smallest side | pairs with an event |
| --- | --- | ---: | ---: | ---: | ---: |
| cobra | doppel | 100 | 100 | 12 | 1 |
| cobra | dupl (t=100, default) | 3 | 3 | 24 | 0 |
| cobra | dupl (t=50) | 7 | 5 | 10 | 0 |
| cobra | token clones (≥50 nodes) | 100 | 29 | 13 | 2 |
| cobra | RRF(doppel, dupl (t=100, default)) | 100 | 100 | 12 | 1 |
| cobra | RRF(doppel, dupl (t=50)) | 100 | 98 | 12 | 1 |
| cobra | RRF(doppel, token clones (≥50 nodes)) | 100 | 57 | 15 | 2 |
| gin | doppel | 100 | 100 | 6 | 7 |
| gin | dupl (t=100, default) | 16 | 1 | 3 | 0 |
| gin | dupl (t=50) | 21 | 3 | 3 | 0 |
| gin | token clones (≥50 nodes) | 100 | 4 | 15 | 2 |
| gin | RRF(doppel, dupl (t=100, default)) | 100 | 85 | 6 | 7 |
| gin | RRF(doppel, dupl (t=50)) | 100 | 82 | 6 | 7 |
| gin | RRF(doppel, token clones (≥50 nodes)) | 100 | 48 | 12 | 8 |
| prometheus | doppel | 100 | 100 | 27 | 33 |
| prometheus | dupl (t=100, default) | 66 | 30 | 19 | 17 |
| prometheus | dupl (t=50) | 100 | 52 | 21 | 29 |
| prometheus | token clones (≥50 nodes) | 100 | 59 | 19 | 21 |
| prometheus | RRF(doppel, dupl (t=100, default)) | 100 | 69 | 26 | 32 |
| prometheus | RRF(doppel, dupl (t=50)) | 100 | 68 | 26 | 30 |
| prometheus | RRF(doppel, token clones (≥50 nodes)) | 100 | 78 | 22 | 28 |
| hugo | doppel | 100 | 100 | 11 | 9 |
| hugo | dupl (t=100, default) | 1 | 1 | 26 | 0 |
| hugo | dupl (t=50) | 88 | 32 | 10 | 1 |
| hugo | token clones (≥50 nodes) | 100 | 27 | 31 | 1 |
| hugo | RRF(doppel, dupl (t=100, default)) | 100 | 100 | 11 | 9 |
| hugo | RRF(doppel, dupl (t=50)) | 100 | 58 | 11 | 8 |
| hugo | RRF(doppel, token clones (≥50 nodes)) | 100 | 61 | 26 | 8 |
| moby | doppel | 100 | 100 | 22 | 6 |
| moby | dupl (t=100, default) | 100 | 22 | 11 | 10 |
| moby | dupl (t=50) | 100 | 25 | 17 | 12 |
| moby | token clones (≥50 nodes) | 100 | 44 | 15 | 10 |
| moby | RRF(doppel, dupl (t=100, default)) | 100 | 64 | 20 | 11 |
| moby | RRF(doppel, dupl (t=50)) | 100 | 63 | 19 | 11 |
| moby | RRF(doppel, token clones (≥50 nodes)) | 100 | 68 | 18 | 10 |
