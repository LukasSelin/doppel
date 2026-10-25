/* doppel dashboard — core.
 *
 * Classic scripts, concatenated in the order render.go lists them (core.js,
 * one file per screen, boot.js) into one inline script, so they share one
 * global: DOPPEL. This file owns everything two screens would otherwise each
 * compute — the payload indexes, the derived corpus tables, the formatting
 * helpers, the shared DOM builders and the router. A screen file only renders.
 *
 * The contract with Go is unchanged: the payload carries raw scores, counts
 * and identifiers, and every percentage, colour and label is decided here or
 * in a screen. The one deliberate exception is Edge.rank — analyzer.RankKey is
 * the single definition of corroborated evidence in this repo, and a second
 * one in JavaScript would drift from it silently. Order lists by it; never
 * recompute it.
 *
 * Determinism: an unchanged tree must render byte-identical HTML, and the
 * page bytes are the payload plus these files, so nothing here may depend on
 * time, randomness or object-key order. Every derived list below is sorted
 * with a total order before anyone sees it.
 */
var DOPPEL = (function () {
  "use strict";

  var DATA = JSON.parse(document.getElementById("doppel-data").textContent);

  /* ── Payload ─────────────────────────────────────────────────────────── */

  var units = DATA.units || [];
  var edges = DATA.edges || [];       // rank desc, then a, then b
  var facts = DATA.facts || {};
  var packages = DATA.packages || []; // by name
  var families = DATA.families || [];
  var concepts = DATA.concepts || [];
  var unusedSeeds = DATA.unusedSeeds || [];
  /* Null when the run skipped the moves stage (not a Go module, no
     toolchain, a failed load): "not measured", never "none found". */
  var moves = DATA.moves || null;
  var moveByUnit = Object.create(null);
  if (moves) moves.list.forEach(function (m) { if (m.unit >= 0) moveByUnit[m.unit] = m; });

  var bodyByUnit = Object.create(null);
  (DATA.bodies || []).forEach(function (b) { bodyByUnit[b.unit] = b.text; });

  /* Edges arrive rank-descending, so every per-unit list inherits that order. */
  var edgesByUnit = Object.create(null);
  var edgeByPair = Object.create(null);
  edges.forEach(function (e, i) {
    (edgesByUnit[e.a] || (edgesByUnit[e.a] = [])).push(i);
    (edgesByUnit[e.b] || (edgesByUnit[e.b] = [])).push(i);
    edgeByPair[e.a + "-" + e.b] = i;
  });

  var familiesByUnit = Object.create(null);
  families.forEach(function (f, i) {
    f.members.forEach(function (m) { (familiesByUnit[m] || (familiesByUnit[m] = [])).push(i); });
  });

  /* ── Formatting and DOM: ui.js, shared with the review page ─────────── */

  var U = DOPPEL_UI;
  var fixed = U.fixed, comma = U.comma, pct = U.pct, plural = U.plural, cmp = U.cmp;
  var $ = U.$, el = U.el, svg = U.svg, clear = U.clear, append = U.append, link = U.link;
  var chip = U.chip, bar = U.bar, stat = U.stat, scoreKey = U.scoreKey;

  /* ── Units and pairs ─────────────────────────────────────────────────── */

  function unit(id) { return units[id]; }
  function qualified(u) { return u.package ? u.package + "." + u.name : u.name; }
  function where(u) { return u.file + ":" + u.line; }
  function body(id) { return bodyByUnit[id]; }
  function edgeIndex(a, b) {
    var k = a < b ? a + "-" + b : b + "-" + a;
    return k in edgeByPair ? edgeByPair[k] : -1;
  }
  function neighbours(id) { return edgesByUnit[id] || []; }
  function familiesOf(id) { return familiesByUnit[id] || []; }
  function moveOf(id) { return moveByUnit[id] || null; }

  /* Entry points no Go code can call: init, and main in package main. A pair
     of them cannot be merged by reusing one — the language forbids naming
     either — so they are hidden by default, the same rule the post-edit hook
     applies. Hidden, never dropped: the triage filter shows them on request. */
  function isEntryPoint(u) { return u.name === "init" || (u.name === "main" && u.package === "main"); }
  function isEntryPair(e) { return isEntryPoint(units[e.a]) || isEntryPoint(units[e.b]); }

  /* Pair kinds, in analyzer's precedence order. hidden: the kinds that say a
     pair is something other than a merge — one idea run in two directions, a
     consolidation already done, or a shared shape whose work is elsewhere. A
     kind annotates and never ranks, so hiding is a reader's filter, not a
     verdict: the triage screen lets it be undone. */
  var KINDS = [
    { id: "", label: "no kind — candidate merge", hidden: false },
    { id: "diverged copy", label: "diverged copy", hidden: false },
    { id: "interface implementations", label: "interface implementations", hidden: false },
    { id: "subsystem copies", label: "subsystem copies", hidden: false },
    { id: "mirror operations", label: "mirror operations", hidden: true },
    { id: "thin wrappers", label: "thin wrappers", hidden: true },
    { id: "different calls", label: "different calls", hidden: true }
  ];
  function kindOf(e) { return e.kind || ""; }

  /* The report's default view of the pair list: merge-worthy, no entry-point
     pairs, no hidden kinds. Rank order, because edges already are. */
  var defaultHiddenKinds = Object.create(null);
  KINDS.forEach(function (k) { if (k.hidden) defaultHiddenKinds[k.id] = true; });
  function isHeadline(e) { return e.merge && !isEntryPair(e) && !defaultHiddenKinds[kindOf(e)]; }

  /* ── Derived corpus tables (computed once, on first use) ──────────────── */

  var memo = Object.create(null);
  function once(key, fn) { return key in memo ? memo[key] : (memo[key] = fn()); }

  function headline() {
    return once("headline", function () {
      var out = [];
      edges.forEach(function (e, i) { if (isHeadline(e)) out.push(i); });
      return out;
    });
  }

  function mergeCount() {
    return once("mergeCount", function () {
      var n = 0;
      edges.forEach(function (e) { if (e.merge) n++; });
      return n;
    });
  }

  /* Per package: functions, pairs and merge-worthy pairs inside it, and the
     same crossing out of it. Sorted by merge-worthy pairs inside, then
     functions, then name. */
  function packageStats() {
    return once("packageStats", function () {
      var by = Object.create(null);
      packages.forEach(function (p) {
        by[p.name] = { name: p.name, functions: p.functions, misfits: p.misfits, norm: p.norm,
          within: 0, withinMerge: 0, cross: 0, crossMerge: 0, rank: 0 };
      });
      edges.forEach(function (e) {
        var pa = units[e.a].package, pb = units[e.b].package;
        if (!by[pa] || !by[pb]) return;
        if (pa === pb) {
          by[pa].within++; if (e.merge) by[pa].withinMerge++;
          by[pa].rank += e.rank;
        } else {
          by[pa].cross++; by[pb].cross++;
          if (e.merge) { by[pa].crossMerge++; by[pb].crossMerge++; }
        }
      });
      return Object.keys(by).map(function (k) { return by[k]; }).sort(function (x, y) {
        return (y.withinMerge - x.withinMerge) || (y.functions - x.functions) || cmp(x.name, y.name);
      });
    });
  }

  /* Seams: package pairs joined by pairs that cross the package line. Sorted
     by merge-worthy pairs, then all pairs, then names. */
  function seams() {
    return once("seams", function () {
      var by = Object.create(null);
      edges.forEach(function (e, i) {
        var pa = units[e.a].package, pb = units[e.b].package;
        if (pa === pb) return;
        if (pb < pa) { var t = pa; pa = pb; pb = t; }
        var k = pa + "\u0000" + pb;
        var s = by[k] || (by[k] = { a: pa, b: pb, pairs: 0, merge: 0, rank: 0, edges: [] });
        s.pairs++; if (e.merge) s.merge++;
        s.rank += e.rank;
        s.edges.push(i);
      });
      return Object.keys(by).map(function (k) { return by[k]; }).sort(function (x, y) {
        return (y.merge - x.merge) || (y.pairs - x.pairs) || cmp(x.a, y.a) || cmp(x.b, y.b);
      });
    });
  }

  /* The two unblended scores as four findings. Cut at the family/fork floor,
     0.60 on both axes — the code-shape value the tool already uses for "alike
     enough to guarantee". */
  var QUAD_CUT = 0.6;
  var QUADS = [
    { id: "merge", label: "Merge candidates", note: "alike bodies that also share callers, callees and concepts" },
    { id: "lookalike", label: "Lookalikes", note: "alike bodies in different surroundings — often a local copy of a shared helper" },
    { id: "sibling", label: "Siblings", note: "same neighbourhood, different bodies — usually intentional variants" },
    { id: "weak", label: "Weak both ways", note: "admitted for recall; rarely worth reading one by one" }
  ];
  function quadOf(e) {
    var s = e.shape >= QUAD_CUT, o = e.overlap >= QUAD_CUT;
    return s && o ? "merge" : s ? "lookalike" : o ? "sibling" : "weak";
  }
  function quadrants() {
    return once("quadrants", function () {
      var out = Object.create(null);
      QUADS.forEach(function (q) { out[q.id] = []; });
      edges.forEach(function (e, i) { out[quadOf(e)].push(i); });
      return out;
    });
  }

  /* A 10×10 count grid over (shape, overlap), row 0 = overlap [0, 0.1). */
  function heat() {
    return once("heat", function () {
      var g = [];
      for (var r = 0; r < 10; r++) { g.push([0, 0, 0, 0, 0, 0, 0, 0, 0, 0]); }
      edges.forEach(function (e) {
        var c = Math.min(9, Math.floor(e.shape * 10)), r = Math.min(9, Math.floor(e.overlap * 10));
        g[Math.max(0, r)][Math.max(0, c)]++;
      });
      return g;
    });
  }

  /* Functions drifting from local practice, misfits first, then by how many
     concepts they realize atypically, then by name. */
  function drifting() {
    return once("drifting", function () {
      return units.filter(function (u) { return u.misfit || (u.atypical && u.atypical.length); })
        .sort(function (x, y) {
          return ((y.misfit ? 1 : 0) - (x.misfit ? 1 : 0)) ||
            ((y.atypical || []).length - (x.atypical || []).length) || cmp(x.key, y.key);
        });
    });
  }

  /* ── Router ──────────────────────────────────────────────────────────── */

  /* Routes:
       #triage                 the ranked list (default)
       #atlas  #atlas/<pkg>    the package atlas, optionally with one selected
       #briefing               the scrolling report
       #case/<a>-<b>           one pair, by unit IDs
       #fn/<id>                one function and its neighbours
     A screen registers { render(host, params) } on DOPPEL.screens; the router
     calls it on every navigation to that screen. Screens may cache their DOM
     and only update on a params change. */
  var screens = Object.create(null);
  var current = { screen: null, params: [] };
  var lastList = "#triage";

  /* The ordered list the case file steps through with previous/next. A list
     screen sets it to whatever it is showing (edge indexes, in its order); the
     default is the headline list. */
  var caseList = null;
  function setCaseList(idxs) { caseList = idxs ? idxs.slice() : null; }
  function getCaseList() { return caseList || headline(); }

  function parse(hash) {
    var h = (hash || "").replace(/^#/, "");
    var parts = h.split("/").map(function (p) { try { return decodeURIComponent(p); } catch (_) { return p; } });
    var name = parts[0] || "triage";
    var m;
    if (name === "case" && parts[1] && (m = parts[1].match(/^(\d+)-(\d+)$/))) {
      return { screen: "case", params: ["pair", +m[1], +m[2]] };
    }
    if (name === "fn" && /^\d+$/.test(parts[1] || "")) return { screen: "case", params: ["fn", +parts[1]] };
    if (name === "pair" && parts[1] && (m = parts[1].match(/^(\d+)-(\d+)$/))) {
      return { screen: "case", params: ["pair", +m[1], +m[2]] };  // the previous page's links
    }
    if (name === "unit" && /^\d+$/.test(parts[1] || "")) return { screen: "case", params: ["fn", +parts[1]] };
    if (!screens[name] || name === "case") name = "triage";
    return { screen: name, params: parts.slice(1) };
  }

  /* A package (or directory) name is one hash segment, so it is encoded. */
  function atlasHref(name) { return "#atlas/" + encodeURIComponent(name); }
  function triagePkgHref(name) { return "#triage/pkg/" + encodeURIComponent(name); }
  function caseHref(a, b) { return "#case/" + Math.min(a, b) + "-" + Math.max(a, b); }
  function fnHref(id) { return "#fn/" + id; }
  function go(hash) { if (location.hash === hash) route(); else location.hash = hash; }

  function route() {
    var r = parse(location.hash);
    if (r.screen !== "case") lastList = location.hash || "#triage";
    var changed = r.screen !== current.screen;
    current = r;
    Array.prototype.forEach.call(document.querySelectorAll(".screen"), function (s) {
      s.hidden = s.dataset.screen !== r.screen;
    });
    Array.prototype.forEach.call(document.querySelectorAll("#tabs .tab"), function (t) {
      var on = t.dataset.screen === r.screen;
      if (on) t.setAttribute("aria-current", "page"); else t.removeAttribute("aria-current");
    });
    var s = screens[r.screen];
    if (s) s.render($("screen-" + r.screen), r.params);
    if (changed) window.scrollTo(0, 0);
  }

  /* ── Shared chrome ───────────────────────────────────────────────────── */

  function renderOperatingPoint() {
    var bits = [plural(facts.functions || 0, "function"), plural(facts.packages || packages.length, "package")];
    bits.push(facts.calibrate > 0
      ? "calibrated at " + (facts.calibrate * 100) + "% → threshold " + fixed(facts.threshold) + ", struct-min " + fixed(facts.structMin)
      : "threshold " + fixed(facts.threshold) + ", struct-min " + fixed(facts.structMin));
    bits.push("tests " + (facts.testsMode || "excluded"));
    $("operating-point").textContent = bits.join(" · ");
  }

  function renderColophon() {
    var bits = [];
    bits.push("This page draws the full comparator-scored, struct-min-filtered pair set — " +
      plural(facts.pairs || 0, "pair") + ". The text report showed " + comma(facts.reported || 0) +
      " after --top and --max-per-func" +
      (facts.suppressed ? ", holding back " + comma(facts.suppressed) + " to the per-function cap" : "") +
      "; those are presentation devices and do not bound this page.");
    if (facts.bodiesOmitted) {
      bits.push("Source is inlined for the highest-ranked pairs only: " + plural(facts.bodiesOmitted, "function body", "function bodies") +
        " left out to keep the file a sane size.");
    }
    if (facts.detailOmitted) {
      bits.push(plural(facts.detailOmitted, "lower-ranked pair") + " carry scores without their shared-structure and reason lists.");
    }
    bits.push("Roles, habitat fit, concepts and typicality are corpus-relative: they move when unrelated code moves. " +
      "No network, no model — an unchanged tree renders this page byte for byte.");
    clear($("colophon")).appendChild(el("p", null, bits.join(" ")));
  }

  /* Find a function: a datalist over qualified names; choosing one opens it. */
  function wireJump() {
    var list = $("jump-list"), input = $("jump");
    if (!list || !input) return;
    var byLabel = Object.create(null);
    units.map(function (u) { return { id: u.id, label: qualified(u) + "  " + where(u) }; })
      .sort(function (x, y) { return cmp(x.label, y.label) || (x.id - y.id); })
      .forEach(function (o) {
        byLabel[o.label] = o.id;
        var opt = document.createElement("option");
        opt.value = o.label;
        list.appendChild(opt);
      });
    function pick() {
      var id = byLabel[input.value];
      if (id !== undefined) { input.value = ""; go(fnHref(id)); }
    }
    input.addEventListener("change", pick);
    input.addEventListener("keydown", function (ev) { if (ev.key === "Enter") pick(); });
  }

  function start() {
    if (DATA.schema !== 4) {
      document.querySelector(".screens").textContent =
        "This page's scripts expect payload schema 4 and received " + DATA.schema + ". Re-run doppel with a matching build.";
      return;
    }
    renderOperatingPoint();
    renderColophon();
    wireJump();
    window.addEventListener("hashchange", route);
    route();
  }

  return {
    data: DATA, units: units, edges: edges, facts: facts, packages: packages,
    families: families, concepts: concepts, unusedSeeds: unusedSeeds, moves: moves,

    fixed: fixed, comma: comma, pct: pct, plural: plural, cmp: cmp,
    $: $, el: el, svg: svg, clear: clear, append: append, link: link,
    chip: chip, bar: bar, stat: stat, scoreKey: scoreKey,

    unit: unit, qualified: qualified, where: where, body: body,
    edgeIndex: edgeIndex, neighbours: neighbours, familiesOf: familiesOf, moveOf: moveOf,
    isEntryPoint: isEntryPoint, isEntryPair: isEntryPair,
    KINDS: KINDS, kindOf: kindOf, isHeadline: isHeadline,
    headline: headline, mergeCount: mergeCount, packageStats: packageStats, seams: seams,
    QUAD_CUT: QUAD_CUT, QUADS: QUADS, quadOf: quadOf, quadrants: quadrants, heat: heat,
    drifting: drifting,

    screens: screens, go: go, route: route, caseHref: caseHref, fnHref: fnHref, atlasHref: atlasHref, triagePkgHref: triagePkgHref,
    lastList: function () { return lastList; },
    setCaseList: setCaseList, getCaseList: getCaseList,
    start: start
  };
})();
