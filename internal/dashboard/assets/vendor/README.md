# Vendored assets

Files here are third-party or design-system artifacts. **Do not hand-edit them.**
Re-vendor from the source below instead, so a resync is a diff rather than a merge.

It is inlined into every HTML report doppel writes, which is why size matters
and why it may never grow a runtime fetch: the page must open from `file://`
with nothing to load.

| File | Version | Source | License |
| --- | --- | --- | --- |
| `broadsheet.css` | — | Claude Design project `6a2b7669-f530-41ec-ba69-44fdf55b8299`, `_ds/broadsheet-091493d9-cff2-43db-8460-44db76e46de7/styles.css` | see file header |

## No vendored JavaScript

There is none, deliberately, and it is worth recording why since a graph
library is the obvious reach here.

Cytoscape.js was vendored first, for a map of package boxes with function dots
and similarity edges. It was dropped when the map became a **power diagram** —
packages as polygons tiling the canvas with shared borders. That is plain SVG
plus about 200 lines of geometry, and the neighbourhood screen never used a
library at all, so the dependency was 373KB in every report written for a
screen that no longer needed it.

Cytoscape was also actively wrong for the picture. Packages were compound
parent nodes, which pin every function inside its own box, so its force layout
could only jiggle within a territory rather than pull related functions across
one. On a corpus with many disconnected components it detonated outright —
measured at y ≈ 1.3e6 on this repo, off-screen and unfittable.

If a library is ever needed again, the bar is that it does something the page
cannot do in a few hundred lines of its own.

## broadsheet.css

A deliberate subset of the design system — see the file's own header for what was
dropped and why. It moved here from `internal/reporter/` when the broadsheet
report was replaced by the dashboard; the file itself is unchanged.
