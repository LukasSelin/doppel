/* doppel timeline.
 *
 * A classic script, inlined into a file:// page, no library — the same
 * contract app.js holds, for the same reasons.
 *
 * The division of labour with Go is also the same: everything arriving in the
 * payload is a raw count, score, key or class name, and every colour, label and
 * percentage is decided here. Nothing on this page is recomputed from evidence
 * — the classifications, the scores and the explanation sentences were all
 * settled by internal/identity against two snapshots that carried the bodies,
 * and this page has neither body nor bag.
 */
(function () {
  "use strict";

  var DATA = JSON.parse(document.getElementById("doppel-data").textContent);

  var $ = function (id) { return document.getElementById(id); };

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined && text !== null) n.textContent = String(text);
    return n;
  }
  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); }
  function fixed(v, n) { return (v === undefined || v === null) ? "—" : Number(v).toFixed(n === undefined ? 2 : n); }
  function comma(n) { return String(n === undefined || n === null ? 0 : n).replace(/\B(?=(\d{3})+(?!\d))/g, ","); }
  function plural(n, one, many) { return comma(n) + " " + (n === 1 ? one : (many || one + "s")); }
  function signed(n) { return (n > 0 ? "+" : "") + comma(n); }

  var steps = DATA.steps || [];
  var changes = DATA.changes || [];
  var tracks = DATA.tracks || [];
  var bounds = DATA.bounds || {};

  /* The class vocabulary, in identity's own report order: the structural
     relocations a key-equality diff cannot produce at all lead, then renames,
     then edits, then the population changes, then unchanged. */
  var CLASSES = ["split", "merged", "moved", "renamed", "edited", "new", "deleted", "unchanged"];

  /* ── Where we are ─────────────────────────────────────────────────────── */

  var at = steps.length - 1;   /* the selected revision; the newest by default */
  var current = "steps";
  var filterClass = "";
  var trackQuery = "";

  function clampStep(i) { return Math.max(0, Math.min(steps.length - 1, i)); }

  /* changeInto(i) is the transition that produced step i. Step 0 has none —
     nothing was observed before it, which is a different statement from
     "nothing changed" and the page says so rather than drawing an empty
     ribbon. */
  function changeInto(i) { return i > 0 ? changes[i - 1] : null; }

  function countOf(c, cls) {
    if (!c) return 0;
    for (var i = 0; i < (c.counts || []).length; i++) {
      if (c.counts[i].class === cls) return c.counts[i].count;
    }
    return 0;
  }

  /* A transition is quiet when nothing but `unchanged` carries a count and no
     pair moved. */
  function quiet(c) {
    if (!c) return true;
    if ((c.createdTotal || 0) > 0 || (c.dissolvedTotal || 0) > 0) return false;
    for (var i = 0; i < CLASSES.length - 1; i++) {
      if (countOf(c, CLASSES[i]) > 0) return false;
    }
    return true;
  }

  /* ── Header ───────────────────────────────────────────────────────────── */

  function fact(n, label, sub) {
    var f = el("div", "fact");
    f.appendChild(el("div", "fact-n mono", n));
    f.appendChild(el("div", "fact-l", label));
    if (sub) f.appendChild(el("div", "fact-s", sub));
    return f;
  }

  function renderFacts() {
    var host = $("facts");
    clear(host);
    var first = steps[0] || {}, last = steps[steps.length - 1] || {};
    var touched = 0, created = 0, dissolved = 0;
    changes.forEach(function (c) {
      created += c.createdTotal || 0;
      dissolved += c.dissolvedTotal || 0;
      CLASSES.forEach(function (k) { if (k !== "unchanged") touched += countOf(c, k); });
    });

    host.appendChild(fact(comma(steps.length), "revisions",
      (first.label || "?") + " → " + (last.label || "?")));
    host.appendChild(fact(comma(last.functions), "functions",
      signed((last.functions || 0) - (first.functions || 0)) + " across the series"));
    host.appendChild(fact(comma(last.pairs), "scored pairs",
      signed((last.pairs || 0) - (first.pairs || 0)) + " across the series"));
    host.appendChild(fact(comma(last.mergeWorthy), "merge-worthy",
      "at the last revision"));
    host.appendChild(fact(comma(touched), "classified changes",
      "excluding unchanged"));
    host.appendChild(fact(comma(created), "pairs created",
      comma(dissolved) + " dissolved"));
  }

  function renderColophon() {
    var host = $("colophon");
    var bits = [];
    bits.push("Each step is a separate doppel run over one revision, and every step in this " +
      "series was analysed at the same pinned operating point — " + (DATA.params || "unrecorded") +
      ". A series whose steps recalibrated per revision would be comparing answers to different " +
      "questions, so `doppel timeline` refuses one.");
    bits.push("Functions are matched across a step by body, not by name: an unchanged snapshot key " +
      "first, then an identical fingerprint digest, then greedy matching on Weisfeiler-Lehman " +
      "overlap. Nothing is matched across a gap — a track is the transitive closure of " +
      "consecutive one-to-one matches, and it claims exactly that much.");
    bits.push("Pair changes are marked attributable when a classified change on either side " +
      "explains them. The rest moved because retrieval re-ranked around them — retrieval keeps a " +
      "bounded top-K per function, so a pair can enter or leave the candidate set with neither " +
      "body touched. Those are shown after everything the revision can be held responsible for.");
    if (bounds.reportTop || bounds.reportMaxPerFunc) {
      bits.push("The pair half of this page is bounded at the source: these runs were written with " +
        "--top " + comma(bounds.reportTop) + " and --max-per-func " + comma(bounds.reportMaxPerFunc) +
        ", so each snapshot stores its ranked report list rather than the full candidate set. " +
        "Re-analyse the series with --top 0 --max-per-func 0 for the whole set. The function " +
        "classifications above are unaffected — they are read from every unit, not from the pair list.");
    }
    if (bounds.flatTracks) {
      bits.push(comma(bounds.flatTracks) + " function lifelines ran the whole series unchanged and " +
        "are not listed" + (bounds.tracksOmitted ? ", along with " + comma(bounds.tracksOmitted) +
          " further tracks held back by the display cap" : "") + ".");
    }
    bits.push("Source bodies are not on this page. Per-revision detail — the map, the pair " +
      "evidence, the side-by-side bodies — is what `doppel analyze -o report.html` renders for a " +
      "single run.");
    bits.push("Concept vocabularies, roles, habitat fit and the nearest-neighbour percentiles are " +
      "all corpus-relative, so a per-step figure describes that revision's corpus rather than " +
      "forming a trend.");
    clear(host);
    host.appendChild(el("p", null, bits.join(" ")));
  }

  /* ── Tabs ─────────────────────────────────────────────────────────────── */

  var screens = { steps: $("screen-steps"), tracks: $("screen-tracks") };

  function show(name) {
    if (!screens[name]) name = "steps";
    current = name;
    Object.keys(screens).forEach(function (k) { screens[k].hidden = k !== name; });
    Array.prototype.forEach.call($("tabs").children, function (b) {
      b.setAttribute("aria-selected", b.dataset.screen === name ? "true" : "false");
    });
  }
  Array.prototype.forEach.call($("tabs").children, function (b) {
    b.addEventListener("click", function () { show(b.dataset.screen); render(); writeHash(); });
  });

  /* ── The stepper ──────────────────────────────────────────────────────── */

  function renderStrip() {
    var host = $("step-strip");
    clear(host);
    steps.forEach(function (s, i) {
      var c = changeInto(i);
      var b = el("button", "step-chip" + (i > 0 && quiet(c) ? " quiet" : ""), s.label || String(i));
      b.type = "button";
      b.setAttribute("role", "tab");
      b.setAttribute("aria-selected", i === at ? "true" : "false");
      b.title = (s.label || String(i)) + " — " + plural(s.functions, "function") +
        ", " + plural(s.pairs, "pair");
      b.addEventListener("click", function () { goto(i); });
      host.appendChild(b);
    });
    var sel = host.children[at];
    if (sel && sel.scrollIntoView) sel.scrollIntoView({ block: "nearest", inline: "nearest" });
  }

  function goto(i) {
    at = clampStep(i);
    render();
    writeHash();
  }

  $("step-prev").addEventListener("click", function () { goto(at - 1); });
  $("step-next").addEventListener("click", function () { goto(at + 1); });
  $("step-range").addEventListener("input", function () { goto(Number(this.value)); });

  /* Arrow keys are the point of the page. They are ignored while a text field
     has focus, where they mean caret movement. */
  document.addEventListener("keydown", function (e) {
    var t = e.target;
    if (t && (t.tagName === "INPUT" || t.tagName === "SELECT" || t.tagName === "TEXTAREA")) return;
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "ArrowLeft") { goto(at - 1); e.preventDefault(); }
    else if (e.key === "ArrowRight") { goto(at + 1); e.preventDefault(); }
    else if (e.key === "Home") { goto(0); e.preventDefault(); }
    else if (e.key === "End") { goto(steps.length - 1); e.preventDefault(); }
  });

  /* ── The steps screen ─────────────────────────────────────────────────── */

  function renderStepFacts() {
    var s = steps[at] || {};
    var host = $("step-facts");
    clear(host);
    var rows = [
      ["functions", comma(s.functions)],
      ["scored pairs", comma(s.pairs)],
      ["merge-worthy", comma(s.mergeWorthy)],
      ["learned concepts", comma(s.concepts)],
      ["compression", fixed(s.compression) + "×"],
      ["median nearest", fixed(s.nnP50) + " · p90 " + fixed(s.nnP90)]
    ];
    rows.forEach(function (r) {
      var line = el("div", "bar-row");
      line.appendChild(el("span", null, r[0]));
      line.appendChild(el("span", null, ""));
      line.appendChild(el("span", "mono", r[1]));
      host.appendChild(line);
    });
    if ((s.unusedSeeds || []).length) {
      host.appendChild(el("p", "control-note",
        "No practice grown for: " + s.unusedSeeds.join(", ") + "."));
    }
  }

  function renderHead() {
    var s = steps[at] || {};
    var c = changeInto(at);
    var host = $("step-head");
    clear(host);
    var h = el("h2", null, s.label || ("revision " + at));
    host.appendChild(h);
    if (!c) {
      host.appendChild(el("p", "control-note",
        "The first revision in the series. Nothing was observed before it, so there is no " +
        "transition into it — step forward to see what changed."));
      return;
    }
    var prev = steps[at - 1] || {};
    host.appendChild(el("p", "control-note",
      "Change from " + (prev.label || ("revision " + (at - 1))) + ". " +
      signed((s.functions || 0) - (prev.functions || 0)) + " functions, " +
      signed((s.pairs || 0) - (prev.pairs || 0)) + " scored pairs. " +
      plural(c.createdTotal || 0, "pair") + " created (" +
      comma(c.attributableNew || 0) + " attributable to a classified change), " +
      plural(c.dissolvedTotal || 0, "pair") + " dissolved."));
  }

  function renderCounts() {
    var host = $("class-counts");
    clear(host);
    var c = changeInto(at);
    if (!c) return;
    CLASSES.forEach(function (k) {
      var n = countOf(c, k);
      var chip = el("div", "class-chip" + (n === 0 ? " zero" : ""));
      chip.appendChild(el("span", "class-dot c-" + k));
      chip.appendChild(el("span", "n", comma(n)));
      chip.appendChild(el("span", "l", k));
      host.appendChild(chip);
    });
  }

  function renderFilter() {
    var sel = $("class-filter");
    var want = filterClass;
    clear(sel);
    var any = el("option", null, "every class");
    any.value = "";
    sel.appendChild(any);
    var c = changeInto(at);
    CLASSES.forEach(function (k) {
      if (k === "unchanged") return;   /* never listed individually */
      var n = countOf(c, k);
      if (!n) return;
      var o = el("option", null, k + " (" + comma(n) + ")");
      o.value = k;
      sel.appendChild(o);
    });
    sel.value = want;
    if (sel.value !== want) { filterClass = ""; sel.value = ""; }
  }
  $("class-filter").addEventListener("change", function () {
    filterClass = this.value;
    renderChanges();
  });

  function keys(list) { return (list || []).join(", "); }

  function renderChanges() {
    var host = $("change-list");
    var note = $("change-count");
    clear(host);
    clear(note);
    var c = changeInto(at);
    if (!c) {
      host.appendChild(el("li", "empty-note", "No transition into the first revision."));
      return;
    }
    var rows = (c.changes || []).filter(function (r) {
      return !filterClass || r.class === filterClass;
    });
    var shown = (c.changes || []).length;
    note.textContent = comma(rows.length) + " shown · " + comma(shown) +
      " listed of " + comma(c.changesTotal || 0) + " classified" +
      ((c.changesTotal || 0) > shown ? " (the rest held back by the display cap)" : "");

    if (!rows.length) {
      host.appendChild(el("li", "empty-note",
        shown ? "No change of that class at this revision." :
          "Nothing but unchanged functions at this revision."));
      return;
    }
    rows.forEach(function (r) {
      var li = el("li");
      var row = el("div", "change-row");
      row.appendChild(el("span", "class-dot c-" + r.class));
      row.appendChild(el("span", "change-class", r.class));
      if ((r.old || []).length) row.appendChild(el("span", "change-keys mono", keys(r.old)));
      if ((r.old || []).length && (r.new || []).length) row.appendChild(el("span", "change-arrow", "→"));
      if ((r.new || []).length) row.appendChild(el("span", "change-keys mono", keys(r.new)));
      li.appendChild(row);

      var ev = [];
      if (r.file) ev.push(r.file + (r.line ? ":" + r.line : ""));
      if (r.class !== "new" && r.class !== "deleted" && r.class !== "split" && r.class !== "merged") {
        ev.push("jaccard " + fixed(r.jaccard) + " · containment " + fixed(r.containment) +
          " · digest " + (r.digestEqual ? "equal" : "differs"));
      }
      if (r.nameChanged) ev.push("also renamed");
      if (r.packageChanged && r.class !== "moved") ev.push("also moved package");
      if (ev.length) li.appendChild(el("span", "change-meta", ev.join("  ·  ")));
      host.appendChild(li);
    });
  }

  function renderPairs(hostID, rows, total, empty) {
    var host = $(hostID);
    clear(host);
    if (!rows || !rows.length) {
      host.appendChild(el("li", "empty-note", empty));
      return;
    }
    rows.forEach(function (p) {
      var li = el("li", p.attributable ? null : "churn");
      var row = el("div", "change-row");
      if (p.mergeWorthy) row.appendChild(el("span", "badge", "merge"));
      row.appendChild(el("span", "change-keys mono", p.a));
      row.appendChild(el("span", "change-arrow", "↔"));
      row.appendChild(el("span", "change-keys mono", p.b));
      li.appendChild(row);

      var ev = ["code-shape " + fixed(p.score) + " · overlap " + fixed(p.overlap)];
      var cause = [];
      if (p.aClass && p.aClass !== "unchanged") cause.push(p.a + " " + p.aClass);
      if (p.bClass && p.bClass !== "unchanged") cause.push(p.b + " " + p.bClass);
      ev.push(cause.length ? "because " + cause.join(", ") : "retrieval churn, not this revision");
      if (p.explain) ev.push(p.explain);
      li.appendChild(el("span", "pair-meta", ev.join("  ·  ")));
      host.appendChild(li);
    });
    if (total > rows.length) {
      host.appendChild(el("li", "empty-note",
        comma(total - rows.length) + " more not listed — the display cap keeps the least " +
        "corroborated back, never the attributable ones."));
    }
  }

  /* ── The tracks screen ────────────────────────────────────────────────── */

  $("track-search").addEventListener("input", function () {
    trackQuery = this.value.trim().toLowerCase();
    renderTracks();
  });

  function renderLegend() {
    var host = $("track-legend");
    clear(host);
    CLASSES.forEach(function (k) {
      var row = el("div", "legend-row");
      var dot = el("span", "legend-dot c-" + k);
      row.appendChild(dot);
      row.appendChild(el("span", "legend-id", k));
      host.appendChild(row);
    });
    var absent = el("div", "legend-row");
    var box = el("span", "legend-dot");
    box.style.background = "transparent";
    box.style.border = "1px dashed var(--color-neutral-400)";
    absent.appendChild(box);
    absent.appendChild(el("span", "legend-id", "not present"));
    host.appendChild(absent);
  }

  function renderTracks() {
    var host = $("track-list");
    var head = $("track-head");
    clear(host);
    clear(head);

    var rows = tracks.filter(function (t) {
      if (!trackQuery) return true;
      return t.label.toLowerCase().indexOf(trackQuery) >= 0 ||
        t.points.some(function (p) { return p.key.toLowerCase().indexOf(trackQuery) >= 0; });
    });

    head.appendChild(el("span", null,
      comma(rows.length) + " of " + comma(tracks.length) + " tracks shown. Each row is one " +
      "function across " + plural(steps.length, "revision") + "; the outlined column is the " +
      "revision selected above."));

    if (!rows.length) {
      host.appendChild(el("li", "empty-note", "No track matches that filter."));
      return;
    }

    /* A track's points are contiguous and ascending, so the grid is filled by
       walking them once against the step axis rather than searching per cell. */
    rows.forEach(function (t) {
      var li = el("li");
      var name = el("div", "track-name");
      name.appendChild(el("span", "mono", t.label));
      var renames = t.points.filter(function (p) { return p.class === "renamed" || p.class === "moved"; });
      var meta = [];
      if (renames.length) {
        meta.push(t.points[0].key + " → " + t.label);
      }
      if (t.fate) meta.push(t.fate + " at " + ((steps[t.last + 1] || {}).label || ("revision " + (t.last + 1))));
      if (t.first > 0) meta.push("first seen at " + ((steps[t.first] || {}).label || ("revision " + t.first)));
      if (meta.length) name.appendChild(el("span", "track-fate", "  " + meta.join(" · ")));
      li.appendChild(name);

      var cells = el("div", "track-cells");
      var byStep = Object.create(null);
      t.points.forEach(function (p) { byStep[p.step] = p; });
      for (var i = 0; i < steps.length; i++) {
        var p = byStep[i];
        var cell = el("div", "track-cell" + (p ? " c-" + (p.class || "unchanged") : " absent") +
          (i === at ? " at" : ""));
        cell.title = (steps[i] || {}).label + (p ? " — " + p.key + (p.class ? " (" + p.class + ")" : "") : " — not present");
        cells.appendChild(cell);
      }
      li.appendChild(cells);
      host.appendChild(li);
    });
  }

  function renderNotes() {
    var host = $("notes-panel");
    clear(host);
    if (!(DATA.notes || []).length) return;
    host.appendChild(el("div", "control-label", "Noted across the series"));
    DATA.notes.forEach(function (n) { host.appendChild(el("p", "control-note", n)); });
  }

  /* ── Hash routing ─────────────────────────────────────────────────────── */

  function writeHash() {
    var h = current === "tracks" ? "#tracks" : "#step/" + at;
    if (location.hash !== h) history.replaceState(null, "", h);
  }

  function readHash() {
    var h = (location.hash || "").replace(/^#/, "");
    if (h === "tracks") { show("tracks"); return; }
    var m = /^step\/(\d+)$/.exec(h);
    if (m) at = clampStep(Number(m[1]));
    show("steps");
  }

  /* ── Render ───────────────────────────────────────────────────────────── */

  function render() {
    var range = $("step-range");
    range.max = String(Math.max(0, steps.length - 1));
    range.value = String(at);
    $("step-prev").disabled = at <= 0;
    $("step-next").disabled = at >= steps.length - 1;

    renderStrip();
    renderHead();
    renderStepFacts();
    renderCounts();
    renderFilter();
    renderChanges();

    var c = changeInto(at);
    renderPairs("created-list", c && c.created, c ? c.createdTotal : 0,
      c ? "No near-duplicate pair appeared at this revision." : "—");
    renderPairs("dissolved-list", c && c.dissolved, c ? c.dissolvedTotal : 0,
      c ? "No near-duplicate pair went away at this revision." : "—");

    renderTracks();
  }

  if (!steps.length) {
    document.querySelector(".page").appendChild(
      el("p", "empty-note", "This timeline has no revisions in it."));
    return;
  }

  $("tracks-bound").textContent = bounds.flatTracks
    ? comma(bounds.flatTracks) + " lifelines ran the whole series unchanged and are not listed" +
    (bounds.tracksOmitted ? "; a further " + comma(bounds.tracksOmitted) +
      " were held back by the display cap" : "") + "."
    : "Every track in the series is listed.";

  renderFacts();
  renderColophon();
  renderNotes();
  renderLegend();
  readHash();
  render();
})();
