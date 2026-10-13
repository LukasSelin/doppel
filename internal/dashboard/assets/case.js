/* doppel dashboard — the case file.
 *
 * The drill-down every other screen links into. Two modes, both from the hash:
 *
 *   #case/<a>-<b>   one pair, laid out to settle it: merge, extract a shared
 *                   part, or leave them. Every number the pipeline produced
 *                   for the pair, the two bodies side by side, the evidence
 *                   behind each score, the families and neighbours a merge
 *                   would reach, and a way to record the verdict.
 *   #fn/<id>        one function and its full ranked neighbour list.
 *
 * Nothing here recomputes a score. Code-shape, overlap and containment are
 * three separate cells and three separate bar colours, never one number; the
 * edge's rank is read, not derived. The only thing this file computes is the
 * line diff between the bodies, and the page says it is textual.
 */
(function (D) {
  "use strict";

  var el = D.el, fixed = D.fixed, plural = D.plural, comma = D.comma;

  /* Bodies past this many lines are shown without the diff: the LCS table is
     quadratic, and a 600×600 table is the most a page should build on click. */
  var DIFF_MAX_LINES = 600;
  var NEIGHBOURS_PER_SIDE = 5;
  var FN_PAGE = 25;
  var FAMILY_MEMBERS_SHOWN = 12;
  var FAMILIES_SHOWN = 6;

  /* The fingerprint blend, in Edge.breakdown order. The weights are
     fingerprint.DefaultWeights; size and containment are reported and never
     scored, which the bar list says rather than implying all six add up. */
  var BREAKDOWN = [
    { name: "WL labels", note: "weight 0.60" },
    { name: "control flow", note: "weight 0.20" },
    { name: "nesting", note: "weight 0.05" },
    { name: "signature", note: "weight 0.15" },
    { name: "size", note: "reported, not scored" },
    { name: "containment", note: "reported, not scored" }
  ];
  var VIEWS = [
    { name: "taxonomy", note: "what the concept tree asserts", feature: false },
    { name: "corpus", note: "the tree read through this corpus's frequencies", feature: false },
    { name: "vocabulary", note: "what the two sides' concepts are made of", feature: true },
    { name: "A in B", note: "A's vocabulary carried by B's concepts", feature: true },
    { name: "B in A", note: "B's vocabulary carried by A's concepts", feature: true }
  ];

  /* ── Small builders ──────────────────────────────────────────────────── */

  function card(title, sub, cls) {
    var c = el("section", "card ca-card" + (cls ? " " + cls : ""));
    var head = el("div", "card-head");
    head.appendChild(el("h2", "card-title", title));
    if (sub) head.appendChild(el("span", "card-sub", sub));
    c.appendChild(head);
    return c;
  }

  function note(text, cls) { return el("p", "ca-note" + (cls ? " " + cls : ""), text); }

  function fnLink(u, cls) {
    var a = D.link(D.fnHref(u.id), "mono ca-fn" + (cls ? " " + cls : ""), D.qualified(u));
    a.title = "Open " + D.qualified(u) + " and all its neighbours";
    return a;
  }

  function otherOf(e, id) { return e.a === id ? e.b : e.a; }

  function roleLine(u) {
    var bits = [u.role || "no role", "fan-in " + u.fanIn + " / fan-out " + u.fanOut, plural(u.nodes, "node")];
    if (u.test) bits.push("test");
    return bits.join(" · ");
  }

  function fitLine(u) {
    if (!(u.fit >= 0)) return null;
    return "habitat fit " + fixed(u.fit) + (u.misfit ? " — a misfit in its package" : "");
  }

  /* A labelled bar with a note under the label. kind: shape | overlap | neutral. */
  function meter(name, sub, value, kind) {
    var row = el("div", "ca-meter");
    var lab = el("div", "ca-meter-label");
    lab.appendChild(el("span", "ca-meter-name", name));
    if (sub) lab.appendChild(el("span", "ca-meter-sub", sub));
    row.appendChild(lab);
    row.appendChild(D.bar(value, kind));
    return row;
  }

  function scoreCell(label, value, notes, cls) {
    var c = el("div", "ca-score" + (cls ? " " + cls : ""));
    c.appendChild(el("span", "ca-score-label", label));
    c.appendChild(el("span", "ca-score-value", value));
    notes.forEach(function (n) { if (n) c.appendChild(el("span", "ca-score-note", n)); });
    return c;
  }

  function conceptList(u) {
    var atyp = Object.create(null);
    (u.atypical || []).forEach(function (c) { atyp[c] = true; });
    var ul = el("ul", "ca-concepts");
    (u.concepts || []).forEach(function (c) {
      var li = el("li", "ca-concept" + (atyp[c.id] ? " is-atypical" : ""));
      li.appendChild(el("span", "mono ca-concept-id", c.id));
      li.appendChild(el("span", "mono ca-concept-conf", fixed(c.confidence)));
      if (atyp[c.id]) li.appendChild(el("span", "ca-atypical", "atypical here"));
      ul.appendChild(li);
    });
    return ul;
  }

  function scopeOf(a, b) {
    if (a.file === b.file) return "same file";
    if (a.package === b.package) return "same package";
    return "cross-package";
  }

  /* ── Clipboard (CORE-CANDIDATE: any screen offering "copy" needs the same
     fallback and the same live confirmation) ────────────────────────────── */

  function copyText(text, status) {
    function done(ok) {
      status.textContent = ok ? "Copied: " + text : "Copy failed — select the text below and copy it by hand.";
    }
    function fallback() {
      var ta = el("textarea", "ca-copy-fallback");
      ta.readOnly = true;
      ta.value = text;
      ta.setAttribute("aria-label", "Text to copy");
      status.parentNode.insertBefore(ta, status.nextSibling);
      ta.focus();
      ta.select();
      var ok = false;
      try { ok = document.execCommand("copy"); } catch (_) { ok = false; }
      if (ok) ta.parentNode.removeChild(ta);
      done(ok);
    }
    var old = status.parentNode.querySelector(".ca-copy-fallback");
    if (old) old.parentNode.removeChild(old);
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () { done(true); }, fallback);
    } else {
      fallback();
    }
  }

  /* ── Bodies and the text diff ────────────────────────────────────────── */

  /* Longest common subsequence over lines; marks the lines unique to each
     side. Bodies are tens to hundreds of lines, so the quadratic table is
     cheap; DIFF_MAX_LINES guards the pathological case. Ties resolve the same
     way on every run, so the shading is deterministic. */
  function diffLines(a, b) {
    var n = a.length, m = b.length, w = m + 1;
    var dp = new Uint32Array((n + 1) * (m + 1));
    for (var i = n - 1; i >= 0; i--) {
      for (var j = m - 1; j >= 0; j--) {
        dp[i * w + j] = a[i] === b[j]
          ? dp[(i + 1) * w + (j + 1)] + 1
          : Math.max(dp[(i + 1) * w + j], dp[i * w + (j + 1)]);
      }
    }
    var ma = [], mb = [], x, y;
    for (x = 0; x < n; x++) ma.push(false);
    for (y = 0; y < m; y++) mb.push(false);
    x = 0; y = 0;
    while (x < n && y < m) {
      if (a[x] === b[y]) { x++; y++; }
      else if (dp[(x + 1) * w + y] >= dp[x * w + (y + 1)]) { ma[x] = true; x++; }
      else { mb[y] = true; y++; }
    }
    while (x < n) ma[x++] = true;
    while (y < m) mb[y++] = true;
    /* A blank line carries no text to differ by; never shade one. */
    for (x = 0; x < n; x++) if (a[x] === "") ma[x] = false;
    for (y = 0; y < m; y++) if (b[y] === "") mb[y] = false;
    return { a: ma, b: mb };
  }

  /* Diffing is on trimmed lines: a re-indent is not a difference worth
     shading, and the score is structural anyway. */
  function norm(lines) { return lines.map(function (l) { return l.replace(/\s+/g, " ").trim(); }); }

  function bodyLines(id) {
    var t = D.body(id);
    return t === undefined || t === null ? null : t.replace(/\r\n?/g, "\n").replace(/\n+$/, "").split("\n");
  }

  function countTrue(xs) { var n = 0; xs.forEach(function (x) { if (x) n++; }); return n; }

  function bodyPane(u, side, lines, marks) {
    var pane = el("div", "ca-body");
    var head = el("div", "ca-body-head");
    if (side) head.appendChild(el("span", "ca-side", side));
    head.appendChild(el("span", "mono ca-body-where", D.where(u)));
    if (lines && marks) {
      var k = countTrue(marks);
      head.appendChild(el("span", "ca-body-count", k === 0 ? "no unique lines" : plural(k, "unique line")));
    } else if (lines) {
      head.appendChild(el("span", "ca-body-count", plural(lines.length, "line")));
    }
    pane.appendChild(head);
    if (!lines) {
      var miss = el("div", "ca-body-missing");
      miss.appendChild(el("p", null, "Source not inlined for " + D.qualified(u) + "."));
      miss.appendChild(el("p", "mono", D.where(u)));
      miss.appendChild(el("p", "muted", D.facts.bodiesOmitted
        ? "The page carries bodies for its highest-ranked pairs only; " +
          plural(D.facts.bodiesOmitted, "function body", "function bodies") + " were left out to bound the file."
        : "This function's body was not carried in the payload."));
      pane.appendChild(miss);
      return pane;
    }
    var box = el("div", "ca-code");
    box.tabIndex = 0;
    box.setAttribute("role", "region");
    box.setAttribute("aria-label", (side ? side + ": " : "") + D.qualified(u) + " source");
    var pre = el("pre", "ca-pre");
    var digits = String(u.line + lines.length - 1).length;
    lines.forEach(function (line, i) {
      var unique = marks && marks[i];
      var row = el("span", "ca-ln" + (unique ? " is-unique" : ""));
      var num = String(u.line + i);
      while (num.length < digits) num = " " + num;
      row.appendChild(el("span", "ca-ln-n", num));
      row.appendChild(el("span", "ca-ln-t", line === "" ? " " : line));
      pre.appendChild(row);
    });
    box.appendChild(pre);
    pane.appendChild(box);
    return pane;
  }

  function bodiesSection(a, b) {
    var sec = el("section", "ca-bodies-wrap");
    sec.setAttribute("aria-label", "Both bodies");
    var head = el("div", "ca-section-head");
    head.appendChild(el("h2", "card-title", "The two bodies"));
    sec.appendChild(head);

    var la = bodyLines(a.id), lb = bodyLines(b.id);
    var marks = { a: null, b: null }, diffNote;
    if (la && lb && la.length <= DIFF_MAX_LINES && lb.length <= DIFF_MAX_LINES) {
      marks = diffLines(norm(la), norm(lb));
      diffNote = "Shaded lines have no textual counterpart on the other side (whitespace ignored). " +
        "The diff is textual; the scores are structural, read off the canonical trees — so this is a reading aid, not the evidence.";
    } else if (la && lb) {
      diffNote = "No line diff: a body runs past " + DIFF_MAX_LINES + " lines, the bound this page sets on the comparison.";
    } else {
      diffNote = "No line diff: at least one body is not inlined in this page.";
    }
    var grid = el("div", "ca-bodies");
    grid.appendChild(bodyPane(a, "A", la, marks.a));
    grid.appendChild(bodyPane(b, "B", lb, marks.b));
    sec.appendChild(grid);
    sec.appendChild(note(diffNote));
    return sec;
  }

  /* ── Pair mode ───────────────────────────────────────────────────────── */

  function sideBlock(u, side) {
    var s = el("div", "ca-sidehead");
    s.appendChild(el("span", "ca-side", side));
    var name = el("div", "ca-sidehead-main");
    name.appendChild(fnLink(u, "ca-name"));
    name.appendChild(el("span", "mono ca-where", D.where(u)));
    var meta = el("span", "ca-meta", roleLine(u));
    name.appendChild(meta);
    var fit = fitLine(u);
    if (fit) name.appendChild(el("span", "ca-meta" + (u.misfit ? " is-misfit" : ""), fit));
    s.appendChild(name);
    return s;
  }

  function headerCard(e, a, b, pos) {
    var h = el("header", "card ca-header");

    var kick = el("div", "ca-kicker-row");
    var pkg = a.package === b.package ? (a.package || "no package") : (a.package + " ↔ " + b.package);
    kick.appendChild(el("span", "kicker", (pos > 0 ? "Case " + comma(pos) + " · " : "Case · ") + pkg));
    var chips = el("div", "ca-chips");
    if (e) {
      chips.appendChild(e.merge ? D.chip("merge-worthy", "solid") : D.chip("not merge-worthy", "quiet"));
      if (e.kind) chips.appendChild(D.chip(e.kind, "outline"));
    }
    chips.appendChild(D.chip(scopeOf(a, b), "outline"));
    if (a.test && b.test) chips.appendChild(D.chip("test pair", "outline"));
    kick.appendChild(chips);
    h.appendChild(kick);

    var sides = el("div", "ca-sides");
    sides.appendChild(sideBlock(a, "A"));
    sides.appendChild(sideBlock(b, "B"));
    h.appendChild(sides);

    if (!e) return h;

    if (e.explain) {
      var lead = el("div", "ca-lead");
      lead.appendChild(el("span", "ca-lead-label", "The pipeline's own sentence"));
      lead.appendChild(el("p", "ca-lead-text", e.explain));
      h.appendChild(lead);
    }
    if (e.kindNote) h.appendChild(el("p", "ca-kindnote", "Why the kind: " + e.kindNote + "."));

    var row = el("div", "ca-scores");
    row.setAttribute("role", "group");
    row.setAttribute("aria-label", "Scores, each reported on its own");
    var f = D.facts;
    row.appendChild(scoreCell("code-shape", fixed(e.shape), ["how alike the bodies are",
      e.shape < f.threshold ? "below the " + fixed(f.threshold) + " admission floor — retrieved by concept or call evidence" : null],
      "is-shape"));
    row.appendChild(scoreCell("overlap", fixed(e.overlap), ["how much context they share",
      f.structMin > 0 ? "struct-min " + fixed(f.structMin) : null], "is-overlap"));
    row.appendChild(scoreCell("containment", fixed(e.containment), ["how much of the smaller body the larger one holds"]));
    row.appendChild(scoreCell("trophic", fixed(e.trophic), ["share of informative structure in common"]));
    row.appendChild(scoreCell("evidence", fixed(e.total, 1), ["retrieval mass, nats" +
      (e.channels && e.channels.length ? " — via " + e.channels.join(" + ") : "")]));
    if (a.test && b.test) {
      row.appendChild(scoreCell("call-sim", fixed(e.callSim), ["shared calls under test — discounts test pairs in ranking"]));
    }
    h.appendChild(row);
    h.appendChild(el("p", "ca-ranknote", "Ranking key " + fixed(e.rank, 3) +
      " — evidence × overlap × shape² × trophic²" + (a.test && b.test ? " × call-sim" : "") +
      ". It orders lists; it is not a score."));
    return h;
  }

  function bodyEvidenceCard(e) {
    var c = card("Why the bodies match", "code-shape " + fixed(e.shape), "ca-ev-shape");
    var bars = el("div", "ca-meters");
    BREAKDOWN.forEach(function (b, i) { bars.appendChild(meter(b.name, b.note, e.breakdown[i], "shape")); });
    c.appendChild(bars);
    if (e.chains && e.chains.length) {
      c.appendChild(el("h3", "ca-subhead", "Strongest shared structure"));
      var ol = el("ol", "ca-chains");
      e.chains.forEach(function (ch) {
        var li = el("li");
        li.appendChild(el("span", "mono ca-chain-r", ch.render));
        li.appendChild(el("span", "mono ca-chain-d", "h" + ch.depth));
        li.appendChild(el("span", "mono ca-chain-e", fixed(ch.energy)));
        ol.appendChild(li);
      });
      c.appendChild(ol);
      c.appendChild(note("Energy is the label's corpus surprisal, ln(N/df), times the count both bodies share; h is the refinement round. A label hashes a whole subtree, so it is named and counted here rather than highlighted in the code."));
    }
    return c;
  }

  function contextEvidenceCard(e) {
    var c = card("Why the context matches", "overlap " + fixed(e.overlap), "ca-ev-overlap");
    if (e.reasons && e.reasons.length) {
      var ul = el("ul", "ca-reasons");
      e.reasons.forEach(function (r) { ul.appendChild(el("li", null, r)); });
      c.appendChild(ul);
    }
    if (!(e.reasons && e.reasons.length) && !(e.chains && e.chains.length)) {
      c.appendChild(note(D.facts.detailOmitted
        ? "Detail omitted for this pair: the page carries reasons and shared structure for its " +
          "highest-ranked pairs only (" + plural(D.facts.detailOmitted, "pair") + " carry scores alone). " +
          "Every score on this page is still complete."
        : "The comparator recorded no shared-context reasons for this pair."));
    } else if (!(e.reasons && e.reasons.length)) {
      c.appendChild(note("The comparator recorded no shared-context reasons for this pair."));
    }
    if (e.channels && e.channels.length) {
      c.appendChild(note("Retrieved by the " + e.channels.join(", ") + " channel" + (e.channels.length > 1 ? "s" : "") +
        "; evidence " + fixed(e.total, 1) + " nats in total."));
    }
    return c;
  }

  function conceptCard(e, a, b) {
    var c = card("What the concepts say", null, "ca-ev-concepts");
    var measured = e.views && e.views[2] >= 0;
    var bars = el("div", "ca-meters");
    VIEWS.forEach(function (v, i) {
      if (v.feature && !measured) return;
      if (!e.views || e.views[i] < 0) return;
      bars.appendChild(meter(v.name, v.note, e.views[i], "neutral"));
    });
    if (bars.firstChild) c.appendChild(bars);
    if (!measured) c.appendChild(note("The vocabulary views were not measured in this run."));
    if (e.viewsDisagree && measured) {
      c.appendChild(el("p", "ca-disagree", e.views[2] > e.views[0]
        ? "The views disagree: the taxonomy misses shared vocabulary — these concepts are made of the same things and the tree cannot see it."
        : "The views disagree: the taxonomy asserts kinship the vocabularies lack."));
    }
    if (e.sharedVocab && e.sharedVocab.length) {
      var sv = el("p", "ca-vocab");
      sv.appendChild(document.createTextNode("Shared vocabulary: "));
      sv.appendChild(el("span", "mono", e.sharedVocab.join(", ")));
      c.appendChild(sv);
    }
    var both = el("div", "ca-concept-sides");
    [[a, "A"], [b, "B"]].forEach(function (p) {
      var col = el("div", "ca-concept-side");
      col.appendChild(el("h3", "ca-subhead", p[1] + " carries"));
      if (p[0].concepts && p[0].concepts.length) col.appendChild(conceptList(p[0]));
      else col.appendChild(note("no learned concept"));
      both.appendChild(col);
    });
    c.appendChild(both);
    c.appendChild(note("Confidence is graded membership; \"atypical here\" means the function realizes that concept unlike its other members in this corpus."));
    return c;
  }

  function familyRow(fi, highlight) {
    var f = D.families[fi];
    var li = el("li", "ca-family");
    var head = el("div", "ca-family-head");
    head.appendChild(el("span", "ca-family-n", "Family " + (fi + 1)));
    head.appendChild(el("span", "ca-family-meta", plural(f.members.length, "member") +
      " · every pair ≥ " + fixed(f.minEdge) + " code-shape · mean " + fixed(f.meanEdge) +
      (f.added ? " · " + plural(f.added, "edge") + " from completion" : "")));
    if (f.kind) head.appendChild(D.chip(f.kind, "outline"));
    li.appendChild(head);
    var ul = el("ul", "ca-family-members");
    var shown = f.members.slice(0, FAMILY_MEMBERS_SHOWN);
    /* Keep the functions this case is about visible even in a large family. */
    f.members.forEach(function (m) { if (highlight[m] && shown.indexOf(m) < 0) shown.push(m); });
    shown.sort(function (x, y) { return x - y; });
    shown.forEach(function (m) {
      var mli = el("li", highlight[m] ? "is-here" : null);
      mli.appendChild(fnLink(D.unit(m)));
      ul.appendChild(mli);
    });
    li.appendChild(ul);
    if (shown.length < f.members.length) {
      li.appendChild(note("and " + plural(f.members.length - shown.length, "more member")));
    }
    return li;
  }

  function familiesCard(ids) {
    var highlight = Object.create(null), seen = Object.create(null), all = [], shared = [];
    ids.forEach(function (id) { highlight[id] = true; });
    ids.forEach(function (id) {
      D.familiesOf(id).forEach(function (fi) {
        if (seen[fi]) return;
        seen[fi] = true;
        var f = D.families[fi];
        var hits = 0;
        ids.forEach(function (x) { if (f.members.indexOf(x) >= 0) hits++; });
        (hits === ids.length && ids.length > 1 ? shared : all).push(fi);
      });
    });
    shared.sort(function (x, y) { return x - y; });
    all.sort(function (x, y) { return x - y; });
    var c = card("Families", ids.length > 1
      ? (shared.length ? plural(shared.length, "family", "families") + " hold both" : "none holds both")
      : plural(shared.length + all.length, "family", "families"));
    if (!shared.length && !all.length) {
      c.appendChild(note(ids.length > 1
        ? "Neither function belongs to a near-duplicate family (a clique where every two members reach the family floor)."
        : "This function belongs to no near-duplicate family."));
      return c;
    }
    var ul = el("ul", "ca-families");
    var order = shared.concat(all);
    order.slice(0, FAMILIES_SHOWN).forEach(function (fi) { ul.appendChild(familyRow(fi, highlight)); });
    c.appendChild(ul);
    if (order.length > FAMILIES_SHOWN) {
      var more = el("button", "btn ca-more-btn", "Show all " + comma(order.length) + " families");
      more.type = "button";
      more.addEventListener("click", function () {
        order.slice(FAMILIES_SHOWN).forEach(function (fi) { ul.appendChild(familyRow(fi, highlight)); });
        more.parentNode.removeChild(more);
      });
      c.appendChild(note("Showing " + FAMILIES_SHOWN + " of " + comma(order.length) +
        (ids.length > 1 && shared.length ? ", the ones holding both first" : "") + ", in census order."));
      c.appendChild(more);
    }
    if (ids.length > 1 && shared.length) {
      c.appendChild(note("A family is a clique: merging this pair touches every member listed with it."));
    }
    return c;
  }

  function neighbourRow(ei, fromId, side) {
    var e = D.edges[ei];
    var o = D.unit(otherOf(e, fromId));
    var a = D.link(D.caseHref(e.a, e.b), "ca-nrow");
    if (side) a.appendChild(el("span", "ca-side ca-nrow-side", side));
    var name = el("span", "ca-nrow-name");
    name.appendChild(el("span", "mono ca-nrow-fn", D.qualified(o)));
    var sub = [D.where(o)];
    if (e.merge) sub.push("merge-worthy");
    if (e.kind) sub.push(e.kind);
    name.appendChild(el("span", "ca-nrow-sub", sub.join(" · ")));
    a.appendChild(name);
    var sc = el("span", "ca-nrow-scores");
    sc.appendChild(D.bar(e.shape, "shape"));
    sc.appendChild(D.bar(e.overlap, "overlap"));
    a.appendChild(sc);
    a.setAttribute("aria-label", D.qualified(o) + ", code-shape " + fixed(e.shape) + ", overlap " + fixed(e.overlap) +
      (e.merge ? ", merge-worthy" : ""));
    return a;
  }

  function neighboursCard(a, b, ei) {
    var c = card("Their other neighbours", "would a merge reach further?");
    c.appendChild(D.scoreKey());
    var any = false;
    [[a, "A"], [b, "B"]].forEach(function (p) {
      var list = D.neighbours(p[0].id).filter(function (x) { return x !== ei; });
      var block = el("div", "ca-nblock");
      block.appendChild(el("h3", "ca-subhead", p[1] + " · " + D.qualified(p[0]) +
        (list.length > NEIGHBOURS_PER_SIDE ? " — top " + NEIGHBOURS_PER_SIDE + " of " + comma(list.length) : "")));
      if (!list.length) {
        block.appendChild(note("No other scored neighbour."));
      } else {
        any = true;
        list.slice(0, NEIGHBOURS_PER_SIDE).forEach(function (x) { block.appendChild(neighbourRow(x, p[0].id, null)); });
        if (list.length > NEIGHBOURS_PER_SIDE) {
          var more = D.link(D.fnHref(p[0].id), "ca-more", "All " + comma(list.length) + " neighbours of " + p[1] + " →");
          block.appendChild(more);
        }
      }
      c.appendChild(block);
    });
    if (any) c.appendChild(note("In rank order. Each row opens that pair's case file."));
    return c;
  }

  function verdictCard(a, b) {
    var c = card("Record a verdict", "copied, not stored", "ca-verdict");
    c.appendChild(note("doppel ranks and explains; it does not decide. This page keeps no state, so a verdict is " +
      "copied as an entry for a labels file — the golden-label format the benchmark scores (examples/labels/)."));
    var status = el("p", "ca-copy-status");
    status.setAttribute("aria-live", "polite");
    status.setAttribute("role", "status");
    /* Spelled the way the committed labels files are, one entry per line. */
    function entry(cls) {
      var q = JSON.stringify;
      return "{\"a\": " + q(a.key) + ", \"b\": " + q(b.key) +
        (a.key === b.key ? ", \"aFile\": " + q(a.file) + ", \"bFile\": " + q(b.file) : "") +
        ", \"class\": " + q(cls) + ", \"note\": \"\"}";
    }
    var row = el("div", "ca-verdict-row");
    [["merge", "Merge them", true], ["refactor", "Extract a shared part", false],
     ["false_positive", "Leave them — not a duplicate", false]].forEach(function (v) {
      var btn = el("button", "btn" + (v[2] ? " btn-primary" : ""), v[1]);
      btn.type = "button";
      btn.title = "Copy a \"" + v[0] + "\" label entry";
      btn.addEventListener("click", function () { copyText(entry(v[0]), status); });
      row.appendChild(btn);
    });
    var loc = el("button", "btn ca-btn-quiet", "Copy file:line pair");
    loc.type = "button";
    loc.addEventListener("click", function () { copyText(D.where(a) + " " + D.where(b), status); });
    row.appendChild(loc);
    c.appendChild(row);
    c.appendChild(status);
    return c;
  }

  /* ── Top bar ─────────────────────────────────────────────────────────── */

  function backLabel(hash) {
    var s = (hash || "").replace(/^#/, "").split("/")[0];
    if (s === "atlas") return "← Back to the atlas";
    if (s === "briefing") return "← Back to the briefing";
    return "← Back to the ranked list";
  }

  function topBar(ei, compact) {
    var bar = el("nav", "ca-top");
    bar.setAttribute("aria-label", "Case navigation");
    var back = D.link(D.lastList(), "ca-back", backLabel(D.lastList()));
    bar.appendChild(back);
    if (compact) bar.appendChild(compact);
    var nav = el("div", "ca-pager");
    var i = -1;
    if (ei >= 0) {
      var list = D.getCaseList();
      i = list.indexOf(ei);
      if (i < 0) {
        nav.appendChild(el("span", "ca-pos muted", "not in the current list"));
      } else {
        nav.appendChild(pagerLink(i > 0 ? list[i - 1] : -1, "‹ previous", "prev"));
        nav.appendChild(el("span", "mono ca-pos", comma(i + 1) + " of " + comma(list.length)));
        nav.appendChild(pagerLink(i + 1 < list.length ? list[i + 1] : -1, "next ›", "next"));
      }
      bar.appendChild(nav);
    }
    return { node: bar, pos: i + 1 };
  }

  function pagerLink(ei, text, rel) {
    if (ei < 0) {
      var s = el("span", "btn ca-pager-btn is-disabled", text);
      s.setAttribute("aria-disabled", "true");
      return s;
    }
    var e = D.edges[ei];
    var a = D.link(D.caseHref(e.a, e.b), "btn ca-pager-btn", text);
    a.rel = rel;
    a.setAttribute("data-dir", rel);
    a.title = D.qualified(D.unit(e.a)) + " ↔ " + D.qualified(D.unit(e.b)) + " (" + (rel === "prev" ? "←" : "→") + ")";
    return a;
  }

  /* The compact summary that rides in the sticky bar once the header has
     scrolled away — the two names and the two scores, still unblended. */
  function compactSummary(e, a, b) {
    var c = el("div", "ca-compact");
    c.setAttribute("aria-hidden", "true");
    c.appendChild(el("span", "mono ca-compact-names", D.qualified(a) + " ↔ " + D.qualified(b)));
    if (e) {
      c.appendChild(el("span", "mono ca-compact-shape", "shape " + fixed(e.shape)));
      c.appendChild(el("span", "mono ca-compact-overlap", "overlap " + fixed(e.overlap)));
    }
    return c;
  }

  /* ── Screen ──────────────────────────────────────────────────────────── */

  var observer = null;
  function watchHeader(root, header) {
    if (observer) { observer.disconnect(); observer = null; }
    if (typeof IntersectionObserver !== "function") return;
    observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) { root.classList.toggle("is-scrolled", !en.isIntersecting); });
    }, { threshold: 0 });
    observer.observe(header);
  }

  function renderPair(root, aId, bId) {
    if (aId > bId) { var t = aId; aId = bId; bId = t; }
    var a = D.unit(aId), b = D.unit(bId);
    if (!a || !b || aId === bId) return renderMissing(root, "No such pair in this report.");
    var ei = D.edgeIndex(aId, bId);
    var e = ei >= 0 ? D.edges[ei] : null;

    var top = topBar(ei, compactSummary(e, a, b));
    root.appendChild(top.node);
    var header = headerCard(e, a, b, top.pos);
    root.appendChild(header);

    if (!e) {
      var nb = card("No scored pair", null, "ca-nopair");
      nb.appendChild(note("These two functions are not a pair in this report: retrieval never proposed them, " +
        "or the comparator scored them below the overlap filter. Their bodies are shown for reading only."));
      var links = el("div", "ca-verdict-row");
      links.appendChild(D.link(D.fnHref(a.id), "btn", "A · " + D.qualified(a) + " →"));
      links.appendChild(D.link(D.fnHref(b.id), "btn", "B · " + D.qualified(b) + " →"));
      nb.appendChild(links);
      root.appendChild(nb);
      root.appendChild(bodiesSection(a, b));
      watchHeader(root, header);
      return;
    }

    root.appendChild(bodiesSection(a, b));

    var ev = el("div", "ca-evidence");
    ev.appendChild(bodyEvidenceCard(e));
    ev.appendChild(contextEvidenceCard(e));
    ev.appendChild(conceptCard(e, a, b));
    root.appendChild(ev);

    var lower = el("div", "ca-lower");
    lower.appendChild(neighboursCard(a, b, ei));
    var right = el("div", "ca-lower-col");
    right.appendChild(familiesCard([a.id, b.id]));
    right.appendChild(verdictCard(a, b));
    lower.appendChild(right);
    root.appendChild(lower);
    watchHeader(root, header);
  }

  function renderMissing(root, msg) {
    var bar = el("nav", "ca-top");
    bar.appendChild(D.link(D.lastList(), "ca-back", backLabel(D.lastList())));
    root.appendChild(bar);
    root.appendChild(el("p", "empty", msg));
  }

  function fnHeader(u) {
    var h = el("header", "card ca-header");
    var kick = el("div", "ca-kicker-row");
    kick.appendChild(el("span", "kicker", "Function · " + (u.package || "no package")));
    var chips = el("div", "ca-chips");
    if (u.test) chips.appendChild(D.chip("test", "outline"));
    if (u.misfit) chips.appendChild(D.chip("habitat misfit", "outline"));
    kick.appendChild(chips);
    h.appendChild(kick);

    h.appendChild(el("h2", "mono ca-fn-title", D.qualified(u)));
    var meta = el("div", "ca-fn-meta");
    meta.appendChild(el("span", "mono ca-where", D.where(u)));
    meta.appendChild(el("span", "ca-meta", roleLine(u)));
    var fit = fitLine(u);
    if (fit) meta.appendChild(el("span", "ca-meta" + (u.misfit ? " is-misfit" : ""), fit));
    if (u.package) meta.appendChild(D.link(D.atlasHref(u.package), "ca-meta", "package " + u.package + " in the atlas →"));
    h.appendChild(meta);

    if (u.concepts && u.concepts.length) {
      var cs = el("div", "ca-fn-concepts");
      cs.appendChild(el("h3", "ca-subhead", "Concepts it carries"));
      cs.appendChild(conceptList(u));
      h.appendChild(cs);
    }
    return h;
  }

  function fnRow(ei, id, n) {
    var li = el("li", "ca-fnrow");
    li.appendChild(el("span", "rank-n ca-fnrow-n", String(n)));
    li.appendChild(neighbourRow(ei, id, null));
    return li;
  }

  function renderFn(root, id) {
    var u = D.unit(id);
    if (!u) return renderMissing(root, "No function with id " + id + " in this report.");
    var list = D.neighbours(id);
    D.setCaseList(list);

    var bar = el("nav", "ca-top");
    bar.setAttribute("aria-label", "Case navigation");
    bar.appendChild(D.link(D.lastList(), "ca-back", backLabel(D.lastList())));
    bar.appendChild(fnCompact(u));
    root.appendChild(bar);
    var header = fnHeader(u);
    root.appendChild(header);

    var split = el("div", "ca-fn-split");
    var left = el("div", "ca-lower-col");
    var nc = card("Neighbours", list.length ? plural(list.length, "scored pair") + ", in rank order" : "none");
    if (!list.length) {
      nc.appendChild(note("No pair involving this function survived scoring and filtering."));
    } else {
      nc.appendChild(D.scoreKey());
      var ol = el("ol", "ca-fnlist");
      nc.appendChild(ol);
      var shown = 0;
      var more = el("button", "btn ca-more-btn");
      more.type = "button";
      var showMore = function () {
        var stop = Math.min(list.length, shown + FN_PAGE);
        var first = null;
        for (; shown < stop; shown++) {
          var r = fnRow(list[shown], id, shown + 1);
          if (!first) first = r;
          ol.appendChild(r);
        }
        if (shown >= list.length) { if (more.parentNode) more.parentNode.removeChild(more); }
        else more.textContent = "Show " + Math.min(FN_PAGE, list.length - shown) + " more (" + comma(list.length - shown) + " left)";
        return first;
      };
      showMore();
      if (shown < list.length) {
        more.addEventListener("click", function () {
          var first = showMore();
          var a = first && first.querySelector("a");
          if (a) a.focus();
        });
        nc.appendChild(more);
      }
      nc.appendChild(note("Each row opens the pair's case file; previous/next there steps through this list."));
    }
    left.appendChild(nc);
    split.appendChild(left);
    var right = el("div", "ca-lower-col");
    right.appendChild(familiesCard([id]));
    split.appendChild(right);
    root.appendChild(split);

    var bsec = el("section", "ca-bodies-wrap");
    bsec.setAttribute("aria-label", "Body");
    var bh = el("div", "ca-section-head");
    bh.appendChild(el("h2", "card-title", "Body"));
    bsec.appendChild(bh);
    var one = el("div", "ca-bodies ca-bodies-one");
    one.appendChild(bodyPane(u, null, bodyLines(id), null));
    bsec.appendChild(one);
    root.appendChild(bsec);
    watchHeader(root, header);
  }

  function fnCompact(u) {
    var c = el("div", "ca-compact");
    c.setAttribute("aria-hidden", "true");
    c.appendChild(el("span", "mono ca-compact-names", D.qualified(u)));
    return c;
  }

  /* ←/→ step through the case list when nothing that takes text has focus. */
  var keysWired = false;
  function wireKeys(host) {
    if (keysWired) return;
    keysWired = true;
    document.addEventListener("keydown", function (ev) {
      if (host.hidden || ev.defaultPrevented || ev.altKey || ev.ctrlKey || ev.metaKey || ev.shiftKey) return;
      if (ev.key !== "ArrowLeft" && ev.key !== "ArrowRight") return;
      var t = ev.target;
      if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
      if (t && t.closest && t.closest(".ca-code")) return; // arrows scroll a code box
      var link = host.querySelector('.ca-pager a[data-dir="' + (ev.key === "ArrowLeft" ? "prev" : "next") + '"]');
      if (!link) return;
      ev.preventDefault();
      D.go(link.getAttribute("href"));
    });
  }

  var last = null;
  D.screens["case"] = {
    render: function (host, params) {
      wireKeys(host);
      var key = params.join("/") + "|" + D.lastList() + "|" + D.getCaseList().length;
      if (key === last && host.firstChild) return;
      var paramsChanged = !last || last.split("|")[0] !== params.join("/");
      last = key;
      var root = el("div", "ca-root");
      if (params[0] === "pair") renderPair(root, +params[1], +params[2]);
      else if (params[0] === "fn") renderFn(root, +params[1]);
      else renderMissing(root, "Open a pair from one of the lists.");
      D.clear(host).appendChild(root);
      /* The router scrolls only when the screen changes; stepping from one
         case to the next is a new page as far as a reader is concerned. */
      if (paramsChanged) window.scrollTo(0, 0);
    }
  };
})(DOPPEL);
