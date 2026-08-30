package reporter

import "html/template"

// htmlTemplate is the Similarity Report page.
//
// The markup follows the approved design closely, inline styles included. That
// is the canvas idiom and it is kept deliberately: fidelity to a design someone
// signed off on matters more here than stylesheet purity, and a generated
// document has no cascade worth protecting. Every colour, space and radius
// still comes from a Broadsheet token — the one rule the design system calls
// out as its failure mode.
//
// html/template does the escaping. Every name in this page comes from analysed
// source, so nothing may be concatenated into markup by hand.
var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"comma": comma,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>doppel — {{ .Report.Target }}</title>
<style>{{ .CSS }}</style>
<style>
  body { background: var(--color-bg); }
  .mono { font-variant-numeric: tabular-nums; }
  .page { min-height:100vh; background:var(--color-bg); color:var(--color-text);
          padding:56px 48px 120px; font-family:var(--font-body) }
  .col { max-width:1180px; display:flex; flex-direction:column; gap:64px }
  .kicker { font-size:12px; letter-spacing:0.12em; text-transform:uppercase }
  .rule-heavy { height:5px; background:var(--color-text) }
  .rule-fine { height:1px; background:var(--color-text) }
  .stat { display:flex; flex-direction:column; gap:6px }
  .stat-n { font-size:64px; line-height:0.9; font-family:var(--font-heading); font-weight:600 }
  .stat-l { font-size:12px; letter-spacing:0.1em; text-transform:uppercase; color:var(--color-neutral-700) }
  .stat-s { font-size:13px; color:var(--color-neutral-700) }
  .lede { font-size:17px; line-height:1.6; text-wrap:pretty }
</style>
</head>
<body>
<div class="page">
<div class="col">

  <header style="display:flex; flex-direction:column; gap:0">
    <div class="rule-heavy"></div>
    <div style="display:flex; justify-content:space-between; align-items:baseline; gap:24px; padding:10px 0 12px">
      <span style="font-size:11px; letter-spacing:0.14em; text-transform:uppercase">doppel analyze</span>
      <span style="font-size:11px; letter-spacing:0.14em; text-transform:uppercase; color:var(--color-neutral-700)">structural report · deterministic · no model</span>
      <span class="mono" style="font-size:11px; letter-spacing:0.14em; text-transform:uppercase">threshold {{ printf "%.2f" .Report.Threshold }}</span>
    </div>
    <div class="rule-fine"></div>
    <h1 style="font-size:76px; line-height:0.98; letter-spacing:-0.02em; margin:36px 0 0; max-width:14em">Where this codebase repeats itself</h1>
    <p style="font-size:20px; line-height:1.5; max-width:34em; margin:22px 0 0; text-wrap:pretty">A single run over <span class="mono">{{ .Report.Target }}</span>, read as structure rather than text. Nothing below comes from a model or from matching source strings: every figure is a count, a score or a distance computed from the syntax tree.</p>
  </header>

  <section style="display:flex; flex-wrap:wrap; gap:56px 72px; align-items:flex-start">
    <div class="stat">
      <div class="cmyk-num mono stat-n">
        <span class="paper">{{ comma .Report.Functions }}</span>
        <span class="plate plate-c" aria-hidden="true">{{ comma .Report.Functions }}</span>
        <span class="plate plate-m" aria-hidden="true">{{ comma .Report.Functions }}</span>
        <span class="plate plate-y" aria-hidden="true">{{ comma .Report.Functions }}</span>
      </div>
      <span class="stat-l">functions analysed</span>
      <span class="stat-s">across {{ .Report.Packages }} packages, tests {{ .Report.TestsMode }}</span>
    </div>
    <div class="stat">
      <div class="mono stat-n">{{ .Report.PairsFound }}</div>
      <span class="stat-l">pairs reported</span>
      <span class="stat-s">out of {{ comma .Report.CandidatePairs }} compared</span>
    </div>
    <div class="stat">
      <div class="mono stat-n">{{ comma .Report.FamilyCount }}</div>
      <span class="stat-l">families</span>
      <span class="stat-s">{{ comma .Report.FamilyFuncs }} functions, largest {{ .Report.FamilyLargest }} members</span>
    </div>
    <div class="stat">
      <div class="mono stat-n">{{ comma .Report.MisfitCount }}</div>
      <span class="stat-l">misfits</span>
      <span class="stat-s">alien to package <em>and</em> subsystem</span>
    </div>
  </section>

{{ if .Report.Strips }}
  <section style="display:flex; flex-direction:column; gap:32px">
    <div style="max-width:36em">
      <span class="kicker" style="color:var(--color-accent-700)">The strip view</span>
      <h2 style="font-size:44px; line-height:1.02; margin:10px 0 16px">Repetition has a silhouette</h2>
      <p class="lede">Each bar below is one function, drawn at the length of its declaration. Read a strip top to bottom: where a family is real, the bars line up. The bar is span only — doppel measures the tree, not the text — so under each strip sit the pairs it actually reported, with the five shape components that earned them their score.</p>
    </div>

  {{ range .Report.Strips }}
    <div style="display:flex; flex-direction:column; gap:18px; padding-bottom:16px">
      <div style="display:flex; flex-wrap:wrap; align-items:baseline; gap:8px 20px">
        <h3 style="font-size:26px; margin:0">{{ .Title }}</h3>
        <span class="mono" style="font-size:13px; color:var(--color-neutral-700)">{{ .File }}</span>
        {{ if .Tag }}<span class="tag tag-accent">{{ .Tag }}</span>{{ end }}
        <span class="mono" style="font-size:13px">every pair ≥ {{ .MinLabel }} code-shape</span>
      </div>
      <p style="font-size:15px; line-height:1.6; max-width:40em; margin:0; color:var(--color-neutral-900)">{{ .Note }}</p>

      <div style="display:flex; flex-direction:column; gap:5px; padding-top:4px">
      {{ range .Members }}
        <div style="display:grid; grid-template-columns:minmax(0,22em) 1fr auto; align-items:center; gap:16px">
          <span class="mono" style="font-size:14px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis">{{ .Name }}</span>
          <div style="height:15px; display:flex; align-items:center">
            {{ if .HasSpan }}<div style="width:{{ .Pct }}%; height:15px; background:var(--color-accent); border-left:2px solid var(--color-text)"></div>{{ end }}
            <span class="mono" style="font-size:11px; padding-left:8px; color:var(--color-neutral-700)">{{ .SpanLabel }}</span>
          </div>
          <span class="mono" style="font-size:12px; color:var(--color-neutral-600)">:{{ .Line }}</span>
        </div>
      {{ end }}
      </div>

      {{ if .Pairs }}
      <div style="display:flex; flex-wrap:wrap; gap:20px; padding-top:8px">
      {{ range .Pairs }}
        <div style="flex:1 1 300px; min-width:280px; display:flex; flex-direction:column; gap:10px; padding:16px 18px; background:var(--color-surface); border-radius:var(--radius-md)">
          <div style="display:flex; justify-content:space-between; align-items:baseline; gap:12px">
            <span style="font-size:10px; letter-spacing:0.12em; text-transform:uppercase; color:var(--color-accent-700)">Match {{ .Label }}</span>
            <span class="mono" style="font-size:13px">shape {{ .ShapeLabel }} · containment {{ .ContainmentLabel }} · overlap {{ .OverlapLabel }}</span>
          </div>
          <div class="mono" style="font-size:14px; line-height:1.5">{{ .A }}<br>{{ .B }}</div>
          <div style="display:flex; flex-direction:column; gap:3px; padding-top:2px">
          {{ range .Components }}
            <div style="display:grid; grid-template-columns:4.5em 1fr 3em; align-items:center; gap:8px">
              <span class="mono" style="font-size:11px; color:var(--color-neutral-700)">{{ .Name }}</span>
              <div style="height:7px; background:var(--color-neutral-300)">
                <div style="width:{{ .Pct }}%; height:7px; background:{{ if .Hot }}var(--color-accent-2){{ else }}var(--color-neutral-800){{ end }}"></div>
              </div>
              <span class="mono" style="font-size:11px; text-align:right">{{ .Label }}</span>
            </div>
          {{ end }}
          </div>
          <p class="mono" style="font-size:12px; margin:0; color:var(--color-neutral-700); line-height:1.5">{{ .Footer }}</p>
        </div>
      {{ end }}
      </div>
      {{ end }}
    </div>
  {{ end }}
  </section>
{{ end }}

  <section style="display:flex; flex-wrap:wrap; gap:56px">
    <div style="flex:1 1 420px; min-width:340px; display:flex; flex-direction:column; gap:20px">
      <div style="max-width:32em">
        <span class="kicker" style="color:var(--color-accent-700)">Vocabulary</span>
        <h2 style="font-size:38px; line-height:1.04; margin:10px 0 14px">What this codebase does</h2>
        <p style="font-size:16px; line-height:1.6; text-wrap:pretty">Intent is read from the tree into a fixed taxonomy. Indented names are abstract — they exist to relate the leaves, so two functions sharing only a branch still score partial credit. The bar is <em>convention</em>: how uniformly this corpus realises the concept.</p>
      </div>
      <div style="display:flex; flex-direction:column; gap:2px">
      {{ range .Report.Taxonomy }}
        <div style="display:grid; grid-template-columns:1fr 4.5em 88px; align-items:center; gap:12px; padding:3px 0">
          <span class="mono" style="font-size:{{ if .Abstract }}13{{ else }}14{{ end }}px; padding-left:{{ .Indent }}px; color:{{ if .Absent }}var(--color-accent-2-700){{ else if .Abstract }}var(--color-neutral-600){{ else }}var(--color-text){{ end }}; font-style:{{ if .Abstract }}italic{{ else }}normal{{ end }}">{{ .Label }}</span>
          <span class="mono" style="font-size:13px; text-align:right; color:{{ if .Absent }}var(--color-accent-2-700){{ else if .Abstract }}var(--color-neutral-600){{ else }}var(--color-text){{ end }}">{{ .CountLabel }}</span>
          <div style="height:8px; background:var(--color-neutral-200)">
            <div style="width:{{ .ConvPct }}%; height:8px; background:{{ if .Loose }}var(--color-accent-2-400){{ else }}var(--color-accent-600){{ end }}"></div>
          </div>
        </div>
      {{ end }}
      </div>
      {{ if .Report.AbsentConcepts }}
      <p style="font-size:14px; line-height:1.6; max-width:32em; margin:0; color:var(--color-neutral-900)">Carried by nothing at all — {{ range $i, $c := .Report.AbsentConcepts }}{{ if $i }}, {{ end }}<span class="mono">{{ $c }}</span>{{ end }}. That is a direct answer to "does this codebase already do X": it does not.</p>
      {{ end }}
    </div>

    <div style="flex:1 1 380px; min-width:320px; display:flex; flex-direction:column; gap:20px">
      <div style="max-width:30em">
        <span class="kicker" style="color:var(--color-accent-2-700)">Habitats</span>
        <h2 style="font-size:38px; line-height:1.04; margin:10px 0 14px">How settled each package is</h2>
        <p style="font-size:16px; line-height:1.6; text-wrap:pretty">Norm is how uniform a package's practice is. A misfit is a function alien to its package <em>and</em> to the subsystem around it — one that fits its neighbours a directory up is normal here and goes unreported.{{ if .Report.HabitatsMore }} The least settled shown, of {{ .Report.HabitatsMore }} more modelled.{{ end }}</p>
      </div>
      <div style="display:flex; flex-direction:column; gap:4px">
      {{ range .Report.Habitats }}
        <div style="display:grid; grid-template-columns:8em 1fr 5.5em; align-items:center; gap:12px">
          <span class="mono" style="font-size:14px">{{ .Name }}</span>
          <div style="height:13px; background:var(--color-neutral-200); position:relative">
            <div style="width:{{ .NormPct }}%; height:13px; background:var(--color-neutral-800)"></div>
            <div style="position:absolute; top:0; left:0; width:{{ .MisfitPct }}%; height:13px; background:var(--color-accent-2)"></div>
          </div>
          <span class="mono" style="font-size:12px; color:var(--color-neutral-700)">{{ .Meta }}</span>
        </div>
      {{ end }}
      </div>
      {{ if .Report.MostUniform }}
      <p style="font-size:14px; line-height:1.6; margin:0; color:var(--color-neutral-900)">Magenta is the share of the package's functions flagged as misfits; grey is its norm. Most uniform in the run is <span class="mono">{{ .Report.MostUniform }}</span> at {{ printf "%.2f" .Report.MostUniformN }}; most varied is <span class="mono">{{ .Report.MostVaried }}</span> at {{ printf "%.2f" .Report.MostVariedN }}.</p>
      {{ end }}
    </div>
  </section>

{{ if .Report.Drift }}
  <section style="display:flex; flex-direction:column; gap:20px">
    <div style="max-width:36em">
      <span class="kicker" style="color:var(--color-accent-2-700)">Drift</span>
      <h2 style="font-size:38px; line-height:1.04; margin:10px 0 14px">Carries the tag, looks nothing like it</h2>
      <p style="font-size:16px; line-height:1.6; text-wrap:pretty">Typicality is measured against the concept's own median, so a varied concept lowers its own bar. Rows marked <em>no near-duplicate</em> appear in no reported pair: nothing else in this report explains them, which makes them drift rather than duplication.</p>
    </div>
    <table class="table" style="max-width:1000px">
      <thead>
        <tr><th>Function</th><th>Concept</th><th style="text-align:right">Typicality</th><th style="text-align:right">Median</th><th></th></tr>
      </thead>
      <tbody>
      {{ range .Report.Drift }}
        <tr>
          <td>
            <span class="mono" style="font-size:14px">{{ .Name }}</span>
            <span class="mono" style="display:block; font-size:11px; color:var(--color-neutral-600)">{{ .File }}:{{ .Line }}</span>
          </td>
          <td><span class="mono" style="font-size:13px">{{ .Tag }}</span></td>
          <td class="mono" style="text-align:right">{{ printf "%.2f" .Typicality }}</td>
          <td class="mono" style="text-align:right; color:var(--color-neutral-600)">{{ printf "%.2f" .Median }}</td>
          <td style="font-size:12px; color:var(--color-accent-2-700); white-space:nowrap">{{ if .Unpaired }}no near-duplicate{{ end }}</td>
        </tr>
      {{ end }}
      </tbody>
    </table>
    {{ if or .Report.DriftMore .Report.MisfitExcused }}
    <p style="font-size:13px; margin:0; color:var(--color-neutral-700)">{{ if .Report.DriftMore }}{{ .Report.DriftMore }} further unusual realisations are not listed.{{ end }}{{ if .Report.MisfitExcused }} A further {{ .Report.MisfitExcused }} functions fit poorly in their package but match the wider subsystem, so they are not reported at all.{{ end }}</p>
    {{ end }}
  </section>
{{ end }}

{{ if .Report.Families }}
  <section style="display:flex; flex-direction:column; gap:20px">
    <div style="max-width:36em">
      <span class="kicker" style="color:var(--color-accent-700)">Census</span>
      <h2 style="font-size:38px; line-height:1.04; margin:10px 0 14px">Families, not pairs</h2>
      <p style="font-size:16px; line-height:1.6; text-wrap:pretty">A family is a clique: every member is at least as alike to <em>every</em> other member as the floor printed here. Chained groups are the classic failure of clone detection — a claim about members that were never compared — so where retrieval left a gap, doppel scored the missing edges before grouping and says how many it added.</p>
    </div>
    <table class="table" style="max-width:1000px">
      <thead>
        <tr><th>#</th><th>What repeats</th><th>Where</th><th style="text-align:right">Members</th><th style="text-align:right">Floor</th><th style="text-align:right">Evidence</th><th style="text-align:right">Edges added</th></tr>
      </thead>
      <tbody>
      {{ range .Report.Families }}
        <tr>
          <td class="mono" style="color:var(--color-neutral-600)">{{ .N }}</td>
          <td>{{ .What }}{{ if .Tag }} <span class="tag tag-neutral" style="margin-left:6px">{{ .Tag }}</span>{{ end }}</td>
          <td class="mono" style="font-size:12px; color:var(--color-neutral-700)">{{ .Where }}</td>
          <td class="mono" style="text-align:right">{{ .Members }}</td>
          <td class="mono" style="text-align:right">{{ .MinLabel }}</td>
          <td class="mono" style="text-align:right">{{ .EvidenceLabel }}</td>
          <td class="mono" style="text-align:right; color:var(--color-neutral-700)">{{ .AddedLabel }}</td>
        </tr>
      {{ end }}
      </tbody>
    </table>
    <p style="font-size:13px; margin:0; color:var(--color-neutral-700)">{{ if .Report.FamiliesMore }}{{ comma .Report.FamiliesMore }} further families are not listed. {{ end }}{{ comma .Report.EdgesScored }} edges across the run were scored during grouping that retrieval never proposed.</p>
  </section>
{{ end }}

{{ if .Report.HasMetrics }}
  <section style="display:flex; flex-direction:column; gap:20px">
    <div style="max-width:36em">
      <span class="kicker" style="color:var(--color-accent-700)">Corpus metrics</span>
      <h2 style="font-size:38px; line-height:1.04; margin:10px 0 14px">How much of this repeats, by shape</h2>
    </div>
    <div style="display:flex; flex-wrap:wrap; gap:56px 72px; align-items:flex-start">
      <div class="stat">
        <div class="mono stat-n">{{ printf "%.2f" .Report.Metrics.Ratio }}x</div>
        <span class="stat-l">compression ratio</span>
        <span class="stat-s" style="max-width:26em; display:block">{{ comma .Report.Metrics.TotalNodes }} canonical AST nodes hash-cons (same node kind and every child, all the way down) to {{ comma .Report.Metrics.UniqueSubtrees }} distinct subtree shapes — nodes divided by shapes, always &ge; 1.0. Never feeds a score.</span>
      </div>
      <div class="stat">
        <div class="mono stat-n">{{ printf "%.2f" .Report.Metrics.NNP50 }} / {{ printf "%.2f" .Report.Metrics.NNP90 }} / {{ printf "%.2f" .Report.Metrics.NNP99 }}</div>
        <span class="stat-l">nearest-neighbour code-shape (p50 / p90 / p99)</span>
        <span class="stat-s" style="max-width:30em; display:block">Of {{ comma .Report.Metrics.NNTotal }} functions, {{ comma .Report.Metrics.NNScored }} had a code-shape neighbour among the pairs retrieval actually scored — {{ printf "%.0f" .Report.Metrics.PctAtOrAboveThreshold }}% of those ({{ .Report.Metrics.NNAtOrAboveThreshold }} of {{ .Report.Metrics.NNScored }}) already clear this run's threshold. <strong>Not</strong> an exhaustive nearest-neighbour search — it is bounded by the same three retrieval channels the pair list is, so the remaining {{ comma .Report.Metrics.Unscored }} functions have no <em>scored</em> neighbour, not necessarily no similar one.</span>
      </div>
    </div>
  </section>
{{ end }}

  <section style="display:flex; flex-direction:column; gap:24px; max-width:1000px">
    <div class="rule-fine"></div>
    <div style="display:flex; flex-wrap:wrap; gap:48px">
      <div style="flex:1 1 300px; min-width:280px">
        <h4 style="margin:0 0 10px">How these candidates were found</h4>
        <p style="font-size:15px; line-height:1.6; margin:0">Three channels nominate candidates independently — shared rare structure, shared concepts, shared calls — and their union is compared. This run: {{ comma .Report.CandidatePairs }} candidate pairs (shape {{ comma .Report.ShapePairs }}, concept {{ comma .Report.ConceptPairs }}, call {{ comma .Report.CallPairs }}), {{ .Report.CallOnlyPct }}% of them arriving on call evidence alone and {{ .Report.ConceptOnlyPct }}% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.</p>
      </div>
      <div style="flex:1 1 300px; min-width:280px">
        <h4 style="margin:0 0 10px">Two scores, never blended</h4>
        <p style="font-size:15px; line-height:1.6; margin:0">Code-shape is the fingerprint blend; structural overlap is twelve weighted context signals. Each combination is a different finding, so averaging them would collapse a lookalike in an unrelated subsystem into a genuine merge candidate. <em>Merge-worthy</em> asserts both axes at once — it is a threshold, not a verdict.</p>
      </div>
      <div style="flex:1 1 300px; min-width:280px">
        <h4 style="margin:0 0 10px">What the strip does not show</h4>
        <p style="font-size:15px; line-height:1.6; margin:0">Bar length is the span between one declaration and the next in the same file, which is why the last member of each strip has none. It is a reading aid, not a measurement: the scores beneath it are what the analysis produced.</p>
      </div>
    </div>
  </section>

</div>
</div>
</body>
</html>
`))
