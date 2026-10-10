### Validity: doppel's list against the earlier study's

| corpus | pairs | identical, in order |
| --- | ---: | --- |
| cobra | 500 | yes |
| gin | 500 | yes |
| prometheus | 500 | yes |
| hugo | 500 | yes |
| moby | 500 | yes |

### Operating points

| corpus | T | T date | pin date | window commits | functions at T | calibration | union | listed units unresolved |
| --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: |
| cobra | `9e1d6f1c2` | 2021-11-16 | 2025-12-03 | 113 | 259 | threshold 0.46 struct-min 0.52 | 1337 | 0 |
| gin | `ecdbbbe94` | 2024-02-19 | 2026-02-28 | 131 | 417 | threshold 0.42 struct-min 0.50 | 1916 | 0 |
| prometheus | `e86e4ed87` | 2024-08-16 | 2026-08-17 | 1382 | 4086 | threshold 0.33 struct-min 0.35 | 22072 | 0 |
| hugo | `2192cf7ec` | 2024-08-12 | 2026-08-12 | 764 | 4631 | threshold 0.34 struct-min 0.31 | 25469 | 0 |
| moby | `02011af7b` | 2023-11-04 | 2025-11-05 | 1810 | 7299 | threshold 0.36 struct-min 0.31 | 36846 | 0 |

### Rates per method

M1 is mean co-changes per edit of the less-edited side; M2 the share with a lagged or unpropagated event; E the share with any event; C the distinct commits behind those events; M1 clustered divides each co-change commit among the pairs of the list it touches. Pooled random holds three lists, so its pair counts are three times the depth.

#### cobra

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0100** | 0.000 | 0.010 | 1 | 0.0100 | 0.0200 | 0.020 | 0.0030 | 0.007 |
| dupl (t=100, default) | 0 | 3 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 7 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| token clones (≥100 nodes) | 0 | 100 | **0.0020** | 0.000 | 0.010 | 1 | 0.0020 | 0.0040 | 0.020 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0120** | 0.000 | 0.020 | 2 | 0.0120 | 0.0200 | 0.020 | 0.0025 | 0.003 |
| token clones (no floor) | 0 | 100 | **0.0300** | 0.000 | 0.030 | 1 | 0.0100 | 0.0000 | 0.000 | 0.0025 | 0.005 |
| code-shape (≥50 nodes) | 0 | 100 | **0.0120** | 0.000 | 0.020 | 2 | 0.0120 | 0.0240 | 0.040 | 0.0000 | 0.000 |

#### gin

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0650** | 0.000 | 0.070 | 2 | 0.0192 | 0.1300 | 0.140 | 0.0000 | 0.000 |
| dupl (t=100, default) | 0 | 16 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 21 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| token clones (≥100 nodes) | 0 | 100 | **0.0000** | 0.020 | 0.020 | 1 | 0.0000 | 0.0000 | 0.040 | 0.0000 | 0.000 |
| token clones (≥50 nodes) | 0 | 100 | **0.0000** | 0.020 | 0.020 | 1 | 0.0000 | 0.0000 | 0.040 | 0.0000 | 0.000 |
| token clones (no floor) | 0 | 100 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0088 | 0.010 |
| code-shape (≥50 nodes) | 0 | 100 | **0.0000** | 0.020 | 0.020 | 1 | 0.0000 | 0.0000 | 0.040 | 0.0000 | 0.000 |

#### prometheus

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.2163** | 0.040 | 0.330 | 47 | 0.2063 | 0.2560 | 0.420 | 0.0535 | 0.087 |
| dupl (t=100, default) | 0 | 66 | **0.1540** | 0.045 | 0.258 | 28 | 0.1470 | 0.2033 | 0.340 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 100 | **0.2092** | 0.040 | 0.290 | 46 | 0.1923 | 0.2233 | 0.360 | 0.0445 | 0.069 |
| token clones (≥100 nodes) | 0 | 100 | **0.1840** | 0.040 | 0.290 | 44 | 0.1790 | 0.2287 | 0.360 | 0.0430 | 0.070 |
| token clones (≥50 nodes) | 0 | 100 | **0.1593** | 0.010 | 0.210 | 23 | 0.1393 | 0.1520 | 0.160 | 0.0649 | 0.105 |
| token clones (no floor) | 0 | 100 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| code-shape (≥50 nodes) | 0 | 100 | **0.1785** | 0.040 | 0.270 | 38 | 0.1663 | 0.1253 | 0.200 | 0.0482 | 0.068 |

#### hugo

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0600** | 0.020 | 0.090 | 8 | 0.0383 | 0.1000 | 0.160 | 0.0112 | 0.015 |
| dupl (t=100, default) | 0 | 1 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 88 | **0.0114** | 0.000 | 0.011 | 1 | 0.0114 | 0.0200 | 0.020 | 0.0000 | 0.000 |
| token clones (≥100 nodes) | 0 | 100 | **0.0100** | 0.010 | 0.020 | 3 | 0.0100 | 0.0000 | 0.000 | 0.0187 | 0.028 |
| token clones (≥50 nodes) | 0 | 100 | **0.0000** | 0.010 | 0.010 | 2 | 0.0000 | 0.0000 | 0.000 | 0.0163 | 0.025 |
| token clones (no floor) | 0 | 100 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| code-shape (≥50 nodes) | 0 | 100 | **0.0233** | 0.010 | 0.040 | 5 | 0.0167 | 0.0200 | 0.020 | 0.0179 | 0.022 |

#### moby

| method | dropped | n@100 | **M1@100** | M2@100 | E@100 | C@100 | M1 clustered@100 | M1@50 | E@50 | M1 101-500 | E 101-500 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 0 | 100 | **0.0400** | 0.020 | 0.060 | 10 | 0.0200 | 0.0600 | 0.080 | 0.0372 | 0.045 |
| dupl (t=100, default) | 0 | 100 | **0.0900** | 0.010 | 0.100 | 6 | 0.0200 | 0.1800 | 0.200 | 0.0000 | 0.000 |
| dupl (t=50) | 0 | 100 | **0.1050** | 0.010 | 0.120 | 10 | 0.0350 | 0.2000 | 0.200 | 0.0000 | 0.000 |
| token clones (≥100 nodes) | 0 | 100 | **0.0000** | 0.040 | 0.040 | 10 | 0.0000 | 0.0000 | 0.020 | 0.0351 | 0.052 |
| token clones (≥50 nodes) | 0 | 100 | **0.0900** | 0.010 | 0.100 | 4 | 0.0200 | 0.1200 | 0.120 | 0.0125 | 0.022 |
| token clones (no floor) | 0 | 100 | **0.0000** | 0.000 | 0.000 | 0 | 0.0000 | 0.0000 | 0.000 | 0.0000 | 0.000 |
| code-shape (≥50 nodes) | 0 | 100 | **0.0900** | 0.000 | 0.090 | 2 | 0.0200 | 0.0600 | 0.060 | 0.0187 | 0.033 |

### Primary comparison: M1@100, doppel minus baseline

| baseline | cobra | gin | prometheus | hugo | moby | doppel wins | baseline wins | verdict |
| --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| dupl (t=100, default) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0200, +0.1200] ✓ | +0.0623 [-0.0417, +0.1661] | +0.0600 [+0.0200, +0.1067] ✓ | -0.0500 [-0.1200, +0.0200] | 2 | 0 | not distinguishable |
| dupl (t=50) | +0.0100 [+0.0000, +0.0300] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0072 [-0.0980, +0.1077] | +0.0486 [+0.0026, +0.1000] ✓ | -0.0650 [-0.1350, +0.0050] | 2 | 0 | not distinguishable |
| token clones (≥100 nodes) | +0.0080 [-0.0040, +0.0300] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0323 [-0.0763, +0.1393] | +0.0500 [+0.0033, +0.1033] ✓ | +0.0400 [+0.0100, +0.0800] ✓ | 3 | 0 | **doppel beats it** |
| token clones (≥50 nodes) | -0.0020 [-0.0320, +0.0280] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0570 [-0.0403, +0.1600] | +0.0600 [+0.0200, +0.1100] ✓ | -0.0500 [-0.1200, +0.0200] | 2 | 0 | not distinguishable |
| token clones (no floor) | -0.0200 [-0.0600, +0.0200] | +0.0650 [+0.0200, +0.1150] ✓ | +0.2163 [+0.1447, +0.2923] ✓ | +0.0600 [+0.0200, +0.1100] ✓ | +0.0400 [+0.0100, +0.0800] ✓ | 4 | 0 | **doppel beats it** |
| code-shape (≥50 nodes) | -0.0020 [-0.0320, +0.0260] | +0.0650 [+0.0200, +0.1150] ✓ | +0.0378 [-0.0693, +0.1400] | +0.0367 [-0.0133, +0.0900] | -0.0500 [-0.1200, +0.0200] | 1 | 0 | not distinguishable |

✓ doppel's CI lower bound is above 0 on that corpus; ✗ the upper bound is below 0. A verdict needs 3 corpora one way and none the other.

### Overlap, size and event counts at 100

Overlap is the number of a method's top 100 also in doppel's top 100. Median smallest side is in lines at T. Pairs with an event is a count, not a share.

| corpus | method | n@100 | in doppel's top 100 | median smallest side | pairs with an event |
| --- | --- | ---: | ---: | ---: | ---: |
| cobra | doppel | 100 | 100 | 12 | 1 |
| cobra | dupl (t=100, default) | 3 | 3 | 24 | 0 |
| cobra | dupl (t=50) | 7 | 5 | 10 | 0 |
| cobra | token clones (≥100 nodes) | 100 | 14 | 26 | 1 |
| cobra | token clones (≥50 nodes) | 100 | 29 | 13 | 2 |
| cobra | token clones (no floor) | 100 | 28 | 3 | 3 |
| cobra | code-shape (≥50 nodes) | 100 | 29 | 14 | 2 |
| gin | doppel | 100 | 100 | 6 | 7 |
| gin | dupl (t=100, default) | 16 | 1 | 3 | 0 |
| gin | dupl (t=50) | 21 | 3 | 3 | 0 |
| gin | token clones (≥100 nodes) | 100 | 1 | 26 | 2 |
| gin | token clones (≥50 nodes) | 100 | 4 | 15 | 2 |
| gin | token clones (no floor) | 100 | 21 | 3 | 0 |
| gin | code-shape (≥50 nodes) | 100 | 4 | 16 | 2 |
| prometheus | doppel | 100 | 100 | 27 | 33 |
| prometheus | dupl (t=100, default) | 66 | 30 | 19 | 17 |
| prometheus | dupl (t=50) | 100 | 52 | 21 | 29 |
| prometheus | token clones (≥100 nodes) | 100 | 60 | 31 | 29 |
| prometheus | token clones (≥50 nodes) | 100 | 59 | 19 | 21 |
| prometheus | token clones (no floor) | 100 | 1 | 3 | 0 |
| prometheus | code-shape (≥50 nodes) | 100 | 66 | 19 | 27 |
| hugo | doppel | 100 | 100 | 11 | 9 |
| hugo | dupl (t=100, default) | 1 | 1 | 26 | 0 |
| hugo | dupl (t=50) | 88 | 32 | 10 | 1 |
| hugo | token clones (≥100 nodes) | 100 | 25 | 31 | 2 |
| hugo | token clones (≥50 nodes) | 100 | 27 | 31 | 1 |
| hugo | token clones (no floor) | 100 | 0 | 3 | 0 |
| hugo | code-shape (≥50 nodes) | 100 | 47 | 23 | 4 |
| moby | doppel | 100 | 100 | 22 | 6 |
| moby | dupl (t=100, default) | 100 | 22 | 11 | 10 |
| moby | dupl (t=50) | 100 | 25 | 17 | 12 |
| moby | token clones (≥100 nodes) | 100 | 43 | 24 | 4 |
| moby | token clones (≥50 nodes) | 100 | 44 | 15 | 10 |
| moby | token clones (no floor) | 100 | 0 | 3 | 0 |
| moby | code-shape (≥50 nodes) | 100 | 50 | 15 | 9 |
