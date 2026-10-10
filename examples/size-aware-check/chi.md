### Units

| unit | T | T date | end date | window commits replayed | sweep scope | functions at T | calibration | union | unresolved |
| --- | --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: |
| chi o2 | `44932d207` | 2018-08-16 | 2022-08-12 | 132 | commit | 140 | threshold 0.43 struct-min 0.57 | 512 | 0 |

### M1@100 per unit

Each cell is M1@100 over the list's own top 100 (n in brackets when shorter), the pairs with any event, the distinct commits behind them, and the median smaller side in lines.

| method | chi o2 |
| --- | ---: |
| doppel | 0.015 · 2 · 6 · 12 |
| dupl (t=100, default) | — |
| dupl (t=50) | 0.500 (1) · 1 · 3 · 14 |
| token clones (≥100 nodes) | 0.000 · 0 · 0 · 22 |
| token clones (≥50 nodes) | 0.015 · 2 · 6 · 14 |
| token clones (no floor) | 0.015 · 2 · 6 · 3 |
| code-shape (≥50 nodes) | 0.015 · 2 · 6 · 14 |
| variant: K-size | 0.015 · 2 · 6 · 13 |
| variant: K-subtree | 0.015 · 2 · 6 · 11 |
| variant: K-floor | 0.015 · 2 · 6 · 11 |

### Primary: each variant against doppel, M1@100

Per unit: the difference at matched depth (each list's top min(nX, nY, 100)), its paired-clustered 95% CI (2000 replicates), ✓/✗ when the CI excludes 0, and in brackets the independent event clusters in the frame; · marks a unit below the floor of 3, which is not pooled; @n a depth below 100. Pooled D is the mean over corpora of the mean over each corpus's counting units, with the CI read off the replicate-wise pooled replicates.

| comparison | chi o2 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | ---: | --- | --- |
| variant: K-size − doppel | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-subtree − doppel | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-floor − doppel | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |

### The gap to the clone detectors

Every method against each detector, at matched depth, under the same rule.

| comparison | chi o2 | units · corpora | pooled D [95% CI] | verdict |
| --- | --- | ---: | --- | --- |
| doppel − dupl (t=100, default) | — | 0 · 0 | — | not scored |
| variant: K-size − dupl (t=100, default) | — | 0 · 0 | — | not scored |
| variant: K-subtree − dupl (t=100, default) | — | 0 · 0 | — | not scored |
| variant: K-floor − dupl (t=100, default) | — | 0 · 0 | — | not scored |
| doppel − dupl (t=50) | -0.500 [-0.500, -0.500] ✗ (1 ·) @1 | 0 · 0 | — | not scored |
| variant: K-size − dupl (t=50) | -0.500 [-0.500, -0.500] ✗ (1 ·) @1 | 0 · 0 | — | not scored |
| variant: K-subtree − dupl (t=50) | -0.500 [-0.500, -0.500] ✗ (1 ·) @1 | 0 · 0 | — | not scored |
| variant: K-floor − dupl (t=50) | -0.500 [-0.500, -0.500] ✗ (1 ·) @1 | 0 · 0 | — | not scored |
| doppel − token clones (≥100 nodes) | +0.015 [+0.000, +0.040] (2 ·) | 0 · 0 | — | not scored |
| variant: K-size − token clones (≥100 nodes) | +0.015 [+0.000, +0.040] (2 ·) | 0 · 0 | — | not scored |
| variant: K-subtree − token clones (≥100 nodes) | +0.015 [+0.000, +0.040] (2 ·) | 0 · 0 | — | not scored |
| variant: K-floor − token clones (≥100 nodes) | +0.015 [+0.000, +0.040] (2 ·) | 0 · 0 | — | not scored |
| doppel − token clones (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-size − token clones (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-subtree − token clones (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-floor − token clones (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| doppel − code-shape (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-size − code-shape (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-subtree − code-shape (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |
| variant: K-floor − code-shape (≥50 nodes) | +0.000 [+0.000, +0.000] (2 ·) | 0 · 0 | — | not scored |

### Gap change per detector

M1(variant) − M1(doppel) at the detector's matched depth, pooled over every unit where the detector lists a pair (no cluster floor): how far the variant moves doppel's difference with that detector. Then the two verdicts against the detector.

| variant | detector | gap change [95% CI] | doppel vs detector | variant vs detector |
| --- | --- | --- | --- | --- |
| variant: K-size | dupl (t=100, default) | — | not scored | not scored |
| variant: K-size | dupl (t=50) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-size | token clones (≥100 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-size | token clones (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-size | code-shape (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-subtree | dupl (t=100, default) | — | not scored | not scored |
| variant: K-subtree | dupl (t=50) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-subtree | token clones (≥100 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-subtree | token clones (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-subtree | code-shape (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-floor | dupl (t=100, default) | — | not scored | not scored |
| variant: K-floor | dupl (t=50) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-floor | token clones (≥100 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-floor | token clones (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |
| variant: K-floor | code-shape (≥50 nodes) | +0.0000 [+0.0000, +0.0000] | not scored | not scored |

### Criteria (a) and (b)

(a) the variant beats doppel: the pooled lower bound above 0. (b) the gap closes: the gap change is above 0 for at least 4 of the 5 detectors, and against no detector is the variant's verdict worse than doppel's.

| variant | (a) | gap change > 0 | verdicts worse | (b) |
| --- | --- | ---: | ---: | --- |
| variant: K-size | no | 0 of 5 | 0 | no |
| variant: K-subtree | no | 0 of 5 | 0 | no |
| variant: K-floor | no | 0 of 5 | 0 | no |

### Small-family cost

Pairs in doppel's top 100 whose smaller side has fewer than 23 lines at T (below a clone detector's floor), and how many of them each variant's top 100 drops; with an event in brackets.

| variant | chi o2 |
| --- | --- |
| doppel: small pairs in top 100 | 95 (2) |
| variant: K-size: dropped | 5 (0) |
| variant: K-subtree: dropped | 4 (0) |
| variant: K-floor: dropped | 4 (0) |

### Overlap of each top 100 with doppel's (descriptive)

| method | chi o2 |
| --- | ---: |
| dupl (t=100, default) | 0 |
| dupl (t=50) | 1 |
| token clones (≥100 nodes) | 10 |
| token clones (≥50 nodes) | 51 |
| token clones (no floor) | 25 |
| code-shape (≥50 nodes) | 56 |
| variant: K-size | 95 |
| variant: K-subtree | 95 |
| variant: K-floor | 95 |
