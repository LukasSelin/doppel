/* doppel dashboard — briefing screen.
 *
 * The shareable, scrolling reading of one run: what you would hand a teammate
 * or print. Every sentence below is generated from the payload — the phrasing
 * rules are chosen to stay true on any corpus, including one with no
 * merge-worthy pair, one package or no family — and every list says how much
 * of the whole it shows.
 *
 * The payload never changes while the page is open, so the DOM is built once
 * and later navigations only re-point the case list and scroll to a section.
 */
(function (D) {
  "use strict";

  var el = D.el, append = D.append, fixed = D.fixed, comma = D.comma, plural = D.plural, cmp = D.cmp;
  var F = D.facts;

  var SECTIONS = [
    { id: "lede", label: "Summary" },
    { id: "act", label: "Act on these first" },
    { id: "scores", label: "Two scores, four findings" },
    { id: "where", label: "Where it lives" },
    { id: "families", label: "Families" },
    { id: "drift", label: "Drift from practice" },
    { id: "method", label: "How this was measured" }
  ];

  var ACT_N = 6, LOOK_N = 5, PKG_N = 8, SEAM_N = 5, FAM_N = 6, FAM_MEMBERS = 6, DRIFT_N = 12, DRIFT_STEP = 25;

  /* ── Small builders ─────────────────────────────────────────────────── */

  function mono(text) { return el("span", "mono", text); }
  function name(id) { return D.qualified(D.unit(id)); }
  function fnLink(id, text) {
    var a = D.link(D.fnHref(id), "br-fn mono", text || name(id));
    a.title = D.where(D.unit(id));
    return a;
  }

  /* Two sides of a pair can share a qualified name (two files of one
     package, or a lexically read script); the file then tells them apart. */
  function sideName(id, other) {
    var n = name(id);
    return n === name(other) ? n + " · " + D.where(D.unit(id)) : n;
  }
  function prose(cls) { return el("p", "br-prose" + (cls ? " " + cls : "")); }
  function share(n, d) { return d > 0 ? " (" + D.pct(n, d) + "%)" : ""; }

  /* A route segment for a package name, which may hold a slash. */
  // CORE-CANDIDATE: atlas and triage links both need the escaped package segment.

  function section(id, n, title) {
    var s = el("section", "br-section");
    s.id = "br-" + id;
    s.dataset.section = id;
    var h = el("div", "br-section-head");
    if (n) h.appendChild(el("span", "br-section-n", String(n)));
    var h2 = el("h2", "br-h2", title);
    h2.id = "br-" + id + "-title";
    s.setAttribute("aria-labelledby", h2.id);
    h.appendChild(h2);
    s.appendChild(h);
    return s;
  }

  /* The three reported quantities, never blended: two bars in their own
     colours and containment as a plain number. */
  function scoreTrio(e) {
    var box = el("div", "br-trio");
    var s = el("div", "br-trio-cell");
    append(s, el("span", "br-trio-l", "code-shape"), D.bar(e.shape, "shape"));
    var o = el("div", "br-trio-cell");
    append(o, el("span", "br-trio-l", "overlap"), D.bar(e.overlap, "overlap"));
    var c = el("div", "br-trio-cell br-trio-c");
    append(c, el("span", "br-trio-l", "containment"), el("span", "mono br-trio-v", fixed(e.containment)));
    append(box, s, o, c);
    return box;
  }

  /* ── Derived numbers for the prose ──────────────────────────────────── */

  /* Where the merge-worthy pairs sit: the packages holding the most of them
     with both sides inside. */
  function mergeHomes() {
    return D.packageStats().filter(function (p) { return p.withinMerge > 0; })
      .slice().sort(function (x, y) { return (y.withinMerge - x.withinMerge) || cmp(x.name, y.name); });
  }

  function largestFamily() {
    var best = -1;
    D.families.forEach(function (f, i) {
      if (best < 0 || f.members.length > D.families[best].members.length) best = i;
    });
    return best;
  }

  /* ── Sections ───────────────────────────────────────────────────────── */

  function buildLede() {
    var s = el("header", "br-section br-lede");
    s.id = "br-lede";
    s.dataset.section = "lede";
    s.appendChild(el("div", "br-rule"));

    var op = [plural(F.functions || D.units.length, "function") + " in " + plural(F.packages || D.packages.length, "package")];
    op.push(F.calibrate > 0
      ? "calibrated at " + (F.calibrate * 100) + "% of random pairs → threshold " + fixed(F.threshold) + ", struct-min " + fixed(F.structMin)
      : "fixed thresholds → threshold " + fixed(F.threshold) + ", struct-min " + fixed(F.structMin));
    op.push("tests " + popWord(F.testsMode) + ", generated files " + popWord(F.generated));
    s.appendChild(el("div", "br-op mono", op.join(" · ")));

    var M = D.mergeCount(), pairs = D.edges.length;
    var h1 = el("h2", "br-headline");
    h1.id = "br-lede-title";
    s.setAttribute("aria-labelledby", h1.id);
    var lines = [];
    if (M > 0) {
      lines.push(M === 1
        ? "One pair of functions is alike in body and shares its surroundings."
        : comma(M) + " pairs of functions are alike in body and share their surroundings.");
      var homes = mergeHomes();
      var np = D.packages.length;
      if (homes.length === 0) {
        lines.push(M === 1 ? "It crosses a package line." : "Every one of them crosses a package line.");
      } else if (np === 1 || (homes.length === 1 && homes[0].withinMerge === M)) {
        lines.push((M === 1 ? "It sits" : "All of them sit") + " inside one package, " + homes[0].name + ".");
      } else if (homes.length === 1 || homes[0].withinMerge * 2 >= M) {
        lines.push(comma(homes[0].withinMerge) + " of them" + share(homes[0].withinMerge, M) +
          " sit inside one package, " + homes[0].name + ".");
      } else {
        var two = homes[0].withinMerge + homes[1].withinMerge;
        lines.push(comma(two) + " of them" + share(two, M) + " sit inside two packages: " +
          homes[0].name + " and " + homes[1].name + ".");
      }
    } else if (pairs > 0) {
      lines.push("No pair of functions is both alike in body and shares its surroundings.");
      lines.push(plural(pairs, "pair was", "pairs were") + " scored; none cleared the merge-worthy verdict.");
    } else {
      lines.push("doppel scored no pair of functions in this corpus.");
    }
    h1.textContent = lines.join(" ");
    s.appendChild(h1);

    var support = [];
    var lf = largestFamily();
    if (lf >= 0) {
      var f = D.families[lf];
      support.push("The largest family has " + f.members.length + " members, every pair at code-shape " +
        fixed(f.minEdge) + " or above.");
    } else {
      support.push("No group of three or more functions is alike pairwise.");
    }
    var seams = D.seams();
    if (seams.length && seams[0].merge > 0) {
      support.push("The heaviest seam joins " + seams[0].a + " and " + seams[0].b + ": " +
        plural(seams[0].merge, "merge-worthy pair") + " of " + comma(seams[0].pairs) + " crossing between them.");
    } else if (seams.length) {
      support.push("No merge-worthy pair crosses a package line; the most pairs cross between " +
        seams[0].a + " and " + seams[0].b + " (" + comma(seams[0].pairs) + ").");
    } else if (pairs > 0) {
      support.push("No scored pair crosses a package line.");
    }
    var p = prose("br-support");
    p.textContent = support.join(" ");
    s.appendChild(p);
    return s;
  }

  function popWord(mode) {
    if (mode === "exclude" || mode === "excluded") return "excluded";
    if (mode === "include" || mode === "included") return "included";
    if (mode === "only") return "only";
    return mode || "excluded";
  }

  function buildAct() {
    var s = section("act", 1, "Act on these first");
    var H = D.headline(), M = D.mergeCount();
    var intro = prose();
    if (H.length) {
      intro.textContent = "The merge-worthy pairs in rank order, with entry points (init, main) and the kinds that say a " +
        "pair is something other than a merge — mirror operations, thin wrappers, different calls — set aside (" +
        comma(H.length) + " of " + comma(M) + " remain). " +
        (H.length > ACT_N ? "The first " + ACT_N + " are below." : H.length === 1 ? "It is below." : "All of them are below.");
    } else if (M > 0) {
      intro.textContent = "All " + plural(M, "merge-worthy pair") + " are entry points or carry a kind that says the pair is " +
        "something other than a merge, so none is listed here; the triage screen shows them on request.";
    } else {
      intro.textContent = "No pair cleared the merge-worthy verdict, so there is nothing to act on first. " +
        "The triage screen still ranks every scored pair.";
    }
    s.appendChild(intro);

    if (H.length) {
      var grid = el("div", "br-cards");
      H.slice(0, ACT_N).forEach(function (ei, i) {
        var e = D.edges[ei];
        var a = D.link(D.caseHref(e.a, e.b), "br-card card");
        var head = el("span", "br-card-head");
        append(head, el("span", "rank-n br-card-n", String(i + 1)), D.chip("merge-worthy", "solid"));
        a.appendChild(head);
        var names = el("span", "br-card-names mono");
        append(names, el("span", null, sideName(e.a, e.b)), el("span", null, sideName(e.b, e.a)));
        a.appendChild(names);
        var why = e.kindNote || (e.explain ? "Canonicalizer: " + e.explain + "." : "");
        if (why) a.appendChild(el("span", "br-card-why", why));
        a.appendChild(scoreTrio(e));
        grid.appendChild(a);
      });
      s.appendChild(grid);
    }
    var more = D.link("#triage", "btn br-more",
      H.length ? "All " + comma(H.length) + ", ranked and filterable →" : "Open the ranked list →");
    s.appendChild(more);
    return s;
  }

  /* ── Two scores ─────────────────────────────────────────────────────── */

  /* A heat cell's quadrant is core's rule applied at the cell's midpoint. */
  function cellQuad(r, c) { return D.quadOf({ shape: (c + 0.5) / 10, overlap: (r + 0.5) / 10 }); }
  function quadLabel(id) {
    for (var i = 0; i < D.QUADS.length; i++) if (D.QUADS[i].id === id) return D.QUADS[i].label;
    return id;
  }
  function range(i) { return fixed(i / 10, 1) + "–" + fixed((i + 1) / 10, 1); }

  function buildHeat() {
    var g = D.heat(), max = 0, top = 9, bottom = 0;
    g.forEach(function (row) { row.forEach(function (n) { if (n > max) max = n; }); });
    function empty(r) { return g[r].every(function (n) { return n === 0; }); }
    while (bottom < top && empty(bottom)) bottom++;

    var fig = el("figure", "br-heat");
    var wrap = el("div", "br-heat-scroll");
    var t = el("table", "br-heat-table");
    var cap = el("caption", "visually-hidden",
      "Count of scored pairs by overlap (rows, high to low) and code-shape (columns, low to high).");
    t.appendChild(cap);
    var tb = el("tbody");
    var readout = el("div", "br-heat-readout mono");
    readout.setAttribute("aria-live", "polite");
    var idle = "Hover or focus a cell for its count.";
    readout.textContent = idle;

    function say(r, c, n) {
      readout.textContent = "overlap " + range(r) + ", code-shape " + range(c) + ": " +
        plural(n, "pair") + " — " + quadLabel(cellQuad(r, c)).toLowerCase();
    }
    for (var r = top; r >= bottom; r--) {
      var tr = el("tr");
      var th = el("th", "br-heat-rl mono", fixed(r / 10, 1));
      th.scope = "row";
      tr.appendChild(th);
      for (var c = 0; c < 10; c++) {
        var n = g[r][c];
        var td = el("td", "br-heat-cell" + (n ? " br-q-" + cellQuad(r, c) : " br-heat-zero"));
        var sw = el("span", "br-heat-sw");
        if (n) sw.style.opacity = fixed(0.18 + 0.82 * Math.log(n + 1) / Math.log(max + 1), 3);
        td.appendChild(sw);
        td.appendChild(el("span", "visually-hidden", plural(n, "pair")));
        td.title = "overlap " + range(r) + ", code-shape " + range(c) + ": " + plural(n, "pair");
        if (n) {
          td.tabIndex = 0;
          (function (r, c, n) {
            function on() { say(r, c, n); }
            td.addEventListener("mouseenter", on);
            td.addEventListener("focus", on);
          })(r, c, n);
        }
        tr.appendChild(td);
      }
      tb.appendChild(tr);
    }
    var foot = el("tr", "br-heat-foot");
    var corner = el("td", "br-heat-corner");
    foot.appendChild(corner);
    for (var k = 0; k < 10; k++) {
      var ch = el("th", "br-heat-cl mono", fixed(k / 10, 1));
      ch.scope = "col";
      foot.appendChild(ch);
    }
    tb.appendChild(foot);
    t.appendChild(tb);
    tb.addEventListener("mouseleave", function () { readout.textContent = idle; });
    wrap.appendChild(t);
    fig.appendChild(wrap);
    fig.appendChild(readout);

    var capText = "Code-shape across, low to high; overlap up. Darker means more pairs, on a log scale. " +
      "Quadrants split at " + fixed(D.QUAD_CUT) + " on both axes.";
    if (bottom > 0) {
      var cut = fixed(bottom / 10, 1);
      capText += F.structMin > 0 && F.structMin <= bottom / 10 + 1e-9
        ? " Rows below overlap " + cut + " are not drawn: pairs under struct-min " + fixed(F.structMin) + " were filtered before scoring reached this page."
        : " Rows below overlap " + cut + " are not drawn: no scored pair falls there.";
    }
    var fc = el("figcaption", "br-caption", capText);
    fig.appendChild(fc);
    var key = el("div", "br-heat-key");
    D.QUADS.forEach(function (q) {
      var k = el("span", "br-heat-key-item");
      append(k, el("i", "br-q-" + q.id), q.label);
      key.appendChild(k);
    });
    fig.appendChild(key);
    return fig;
  }

  function buildScores() {
    var s = section("scores", 2, "Two scores, four findings");
    var p = prose();
    p.textContent = "Every one of the " + plural(D.edges.length, "scored pair") + " placed by how alike the two bodies are " +
      "(code-shape, across) and how much architectural context they share (overlap, up). The corners are not a " +
      "stronger and weaker version of one thing — they ask different questions.";
    s.appendChild(p);
    s.appendChild(D.scoreKey());
    if (!D.edges.length) {
      s.appendChild(el("p", "empty", "No pair was scored, so there is nothing to place."));
      return s;
    }
    var row = el("div", "br-scores");
    row.appendChild(buildHeat());
    var quads = el("div", "br-quads");
    var Q = D.quadrants();
    D.QUADS.forEach(function (q) {
      var a = D.link("#triage/quad/" + q.id, "br-quad card br-quad-" + q.id);
      append(a, el("span", "br-quad-n", comma(Q[q.id].length)), el("span", "br-quad-name", q.label),
        el("span", "br-quad-note", q.note));
      quads.appendChild(a);
    });
    row.appendChild(quads);
    s.appendChild(row);
    s.appendChild(buildLookalikes(Q.lookalike));
    return s;
  }

  function buildLookalikes(idxs) {
    var box = el("div", "br-looks");
    box.appendChild(el("h3", "br-h3", "Lookalikes worth a second look"));
    var list = idxs.filter(function (i) { return !D.isEntryPair(D.edges[i]); });
    var entries = idxs.length - list.length;
    list.sort(function (x, y) {
      var a = D.edges[x], b = D.edges[y];
      return (b.shape - a.shape) || (b.rank - a.rank) || (x - y);
    });
    var p = prose("br-small");
    if (!list.length) {
      p.textContent = idxs.length
        ? "Every lookalike pair involves an entry point (init, main), which no code can call, so none is listed."
        : "No pair has alike bodies in different surroundings at the " + fixed(D.QUAD_CUT) + " cut.";
      box.appendChild(p);
      return box;
    }
    p.textContent = "Alike bodies whose surroundings differ — the shape a local copy of a shared helper takes. " +
      "The " + Math.min(LOOK_N, list.length) + " with the highest code-shape of " + comma(list.length) +
      ", ties broken by rank" + (entries ? "; " + plural(entries, "entry-point pair") + " set aside" : "") + ".";
    box.appendChild(p);
    var ul = el("ul", "br-look-list");
    list.slice(0, LOOK_N).forEach(function (i) {
      var e = D.edges[i];
      var li = el("li", "br-look");
      var a = D.link(D.caseHref(e.a, e.b), "br-look-names mono");
      append(a, el("span", null, sideName(e.a, e.b)), el("span", "br-look-sep", " ↔ "), el("span", null, sideName(e.b, e.a)));
      li.appendChild(a);
      var meta = el("span", "br-look-meta");
      append(meta, el("span", "br-ink-shape mono", "code-shape " + fixed(e.shape)),
        el("span", "br-ink-overlap mono", "overlap " + fixed(e.overlap)),
        e.cross ? D.chip("cross-package", "outline") : null,
        e.kind ? D.chip(e.kind, "outline") : null);
      li.appendChild(meta);
      ul.appendChild(li);
    });
    box.appendChild(ul);
    return box;
  }

  /* ── Where ──────────────────────────────────────────────────────────── */

  function buildWhere() {
    var s = section("where", 3, "Where it lives");
    var homes = mergeHomes();
    var p = prose();
    if (!homes.length) {
      p.textContent = D.mergeCount()
        ? "No package holds a merge-worthy pair with both sides inside it; they all cross package lines."
        : "No package holds a merge-worthy pair.";
      s.appendChild(p);
    } else {
      p.textContent = "Packages by merge-worthy pairs with both sides inside, beside how many functions each holds. " +
        (homes.length > PKG_N ? "The first " + PKG_N + " of " + comma(homes.length) + " packages holding any." :
          plural(homes.length, "package holds", "packages hold") + " any.");
      s.appendChild(p);
      var max = homes[0].withinMerge;
      var ul = el("ul", "br-pkgs");
      homes.slice(0, PKG_N).forEach(function (h) {
        var li = el("li");
        var a = D.link(D.atlasHref(h.name), "br-pkg");
        a.appendChild(el("span", "br-pkg-name mono", h.name));
        var track = el("span", "br-pkg-track");
        var fill = el("span", "br-pkg-fill");
        fill.style.width = (100 * h.withinMerge / max) + "%";
        track.appendChild(fill);
        a.appendChild(track);
        a.appendChild(el("span", "br-pkg-n mono", plural(h.withinMerge, "pair") + " / " + plural(h.functions, "fn")));
        li.appendChild(a);
        ul.appendChild(li);
      });
      s.appendChild(ul);
    }

    var seams = D.seams();
    s.appendChild(el("h3", "br-h3", "Seams"));
    var sp = prose("br-small");
    if (!seams.length) {
      sp.textContent = "No scored pair crosses a package line.";
      s.appendChild(sp);
      return s;
    }
    var first = seams[0];
    sp.textContent = (first.merge > 0
      ? "The heaviest seam crosses " + first.a + " and " + first.b + " (" + plural(first.merge, "merge-worthy pair") +
        " of " + comma(first.pairs) + ")."
      : "No merge-worthy pair crosses a package line; seams are ordered by all pairs crossing them.") +
      " " + (seams.length > SEAM_N ? "The first " + SEAM_N + " of " + comma(seams.length) + " package pairs joined by a scored pair." :
        plural(seams.length, "package pair is", "package pairs are") + " joined by a scored pair.");
    s.appendChild(sp);
    var ol = el("ol", "br-seams");
    seams.slice(0, SEAM_N).forEach(function (sm) {
      var li = el("li", "br-seam");
      var names = el("span", "br-seam-names");
      append(names, D.link(D.atlasHref(sm.a), "mono", sm.a), el("span", "br-look-sep", " ↔ "),
        D.link(D.atlasHref(sm.b), "mono", sm.b));
      li.appendChild(names);
      li.appendChild(el("span", "br-seam-n mono", comma(sm.merge) + " merge-worthy / " + plural(sm.pairs, "pair")));
      ol.appendChild(li);
    });
    s.appendChild(ol);
    return s;
  }

  /* ── Families ───────────────────────────────────────────────────────── */

  function buildFamilies() {
    var s = section("families", 4, "Families");
    var fams = D.families, total = F.familyCount || fams.length;
    var p = prose();
    if (!fams.length) {
      p.textContent = "No group of three or more functions is pairwise alike at the family floor.";
      s.appendChild(p);
      return s;
    }
    p.textContent = plural(total, "group") + " in which every member is alike every other one, covering " +
      plural(F.familyFuncs || 0, "function") + ". Ranked by evidence, not size" +
      (fams.length > FAM_N ? "; the first " + FAM_N + " are shown." : ".");
    s.appendChild(p);
    var ol = el("ol", "br-fams");
    fams.slice(0, FAM_N).forEach(function (f, i) {
      var li = el("li", "br-fam");
      li.appendChild(el("span", "rank-n br-fam-n", "F" + (i + 1)));
      var meta = el("div", "br-fam-meta");
      append(meta, el("span", null, plural(f.members.length, "member")),
        el("span", "mono muted br-fam-min", "every pair ≥ " + fixed(f.minEdge) + " code-shape"));
      if (f.kind || f.tag) {
        var tags = el("span", "br-fam-tags");
        append(tags, f.kind ? D.chip(f.kind, "outline") : null, f.tag ? D.chip("concept " + f.tag, "quiet") : null);
        meta.appendChild(tags);
      }
      li.appendChild(meta);
      var mem = el("div", "br-fam-members");
      f.members.slice(0, FAM_MEMBERS).forEach(function (m) { mem.appendChild(fnLink(m)); });
      if (f.members.length > FAM_MEMBERS) {
        mem.appendChild(el("span", "muted br-fam-more", plural(f.members.length - FAM_MEMBERS, "more", "more")));
      }
      li.appendChild(mem);
      ol.appendChild(li);
    });
    s.appendChild(ol);
    return s;
  }

  /* ── Drift ──────────────────────────────────────────────────────────── */

  function driftBlock(value, text, extra) {
    var b = el("div", "br-drift-stat");
    append(b, el("span", "br-drift-n", value), el("span", "br-drift-l", text), extra || null);
    return b;
  }

  function buildDrift() {
    var s = section("drift", 5, "Drift from local practice");
    var stats = el("div", "br-drift-stats");
    var u = F.unusual || 0;
    stats.appendChild(driftBlock(comma(u), u === 1
      ? "concept membership is realized unlike the concept's other members"
      : "concept memberships are realized unlike the concept's other members"));
    var mf = F.misfits || 0, ex = F.misfitsExcused || 0;
    stats.appendChild(driftBlock(comma(mf),
      (mf === 1 ? "function is" : "functions are") + " out of place in their package" +
      (ex ? (mf ? " (" + comma(ex) + " more excused by their subsystem)" :
        "; " + comma(ex) + " package " + (ex === 1 ? "misfit is" : "misfits are") + " excused by their subsystem") : "")));
    var seeds = D.unusedSeeds;
    var list = null;
    if (seeds.length) {
      list = el("span", "br-drift-seeds");
      seeds.forEach(function (sd, i) { append(list, i ? ", " : null, mono(sd)); });
    }
    stats.appendChild(driftBlock(comma(seeds.length),
      seeds.length === 1 ? "seed practice this corpus grew no concept for:" :
        seeds.length ? "seed practices this corpus grew no concept for:" : "seed practices absent — every seed grew a concept", list));
    s.appendChild(stats);

    var dr = D.drifting();
    var p = prose("br-small");
    if (!dr.length) {
      p.textContent = "No function is a misfit in its package or realizes a concept atypically.";
      s.appendChild(p);
      return s;
    }
    p.textContent = "The drifting functions by name: misfits first, then by how many concepts each realizes atypically. " +
      "A drifting function in no pair is as worth reading as one in a pair.";
    s.appendChild(p);
    var ul = el("ul", "br-drift-list");
    var shown = 0;
    var tail = el("div", "br-drift-tail");
    function more(n) {
      dr.slice(shown, shown + n).forEach(function (u) {
        var li = el("li", "br-drift-row");
        var head = el("span", "br-drift-head");
        append(head, fnLink(u.id, u.name), el("span", "mono muted br-drift-pkg", u.package),
          u.misfit ? D.chip("misfit", "solid") : null);
        li.appendChild(head);
        if (u.atypical && u.atypical.length) {
          var at = el("span", "br-drift-at");
          at.appendChild(el("span", "muted", "atypical: "));
          u.atypical.forEach(function (c, i) { append(at, i ? ", " : null, mono(c)); });
          li.appendChild(at);
        }
        ul.appendChild(li);
      });
      shown = Math.min(dr.length, shown + n);
      D.clear(tail);
      if (shown < dr.length) {
        var rest = dr.length - shown;
        tail.appendChild(el("span", "muted", plural(rest, "more drifting function") + " not shown."));
        var b = el("button", "btn", "Show " + Math.min(DRIFT_STEP, rest) + " more");
        b.type = "button";
        b.addEventListener("click", function () { more(DRIFT_STEP); });
        tail.appendChild(b);
      }
    }
    more(DRIFT_N);
    s.appendChild(ul);
    s.appendChild(tail);
    return s;
  }

  /* ── Method ─────────────────────────────────────────────────────────── */

  function buildMethod() {
    var s = section("method", 6, "How this was measured");
    s.classList.add("br-method");
    var grid = el("div", "br-method-grid");
    function item(text) { grid.appendChild(el("p", "br-method-item", text)); }
    item(F.calibrate > 0
      ? "Thresholds from the corpus's own null distribution: threshold " + fixed(F.threshold) + ", struct-min " +
        fixed(F.structMin) + ", each admitting at most " + (F.calibrate * 100) + "% of random unrelated pairs."
      : "Fixed thresholds: threshold " + fixed(F.threshold) + (F.structMin > 0 ? ", struct-min " + fixed(F.structMin) : ", no overlap filter") + ".");
    item(comma(F.candidatePairs || 0) + " candidate pairs retrieved (shape " + comma(F.shapePairs || 0) +
      " · concept " + comma(F.conceptPairs || 0) + " · call " + comma(F.callPairs || 0) +
      "; a pair can arrive by more than one channel); " + comma(F.pairs || D.edges.length) +
      (F.structMin > 0 ? " survived the overlap filter." : " were scored."));
    if (F.compression > 0) {
      item(fixed(F.compression, 1) + "× structural repetition: canonical syntax nodes per distinct subtree shape across the corpus.");
    }
    if (F.nnScored > 0) {
      item("Median nearest-neighbour code-shape " + fixed(F.nnP50) + " (p90 " + fixed(F.nnP90) + ") over the " +
        plural(F.nnScored, "function") + " retrieval paired with anyone; functions it never paired are not counted.");
    }
    item("Tests " + popWord(F.testsMode) + "; generated files " + popWord(F.generated) +
      ". No network, no model — the same tree renders this page byte for byte.");
    s.appendChild(grid);
    return s;
  }

  /* ── Contents and scrolling ─────────────────────────────────────────── */

  function scrollTo(id) {
    var t = document.getElementById("br-" + id);
    if (t) t.scrollIntoView({ block: "start" });
  }

  function buildToc(buttons) {
    var nav = el("nav", "br-toc");
    nav.setAttribute("aria-label", "Briefing contents");
    nav.appendChild(el("div", "br-toc-kicker", "Contents"));
    var ol = el("ol", "br-toc-list");
    SECTIONS.forEach(function (sec, i) {
      var li = el("li");
      var b = el("button", "br-toc-btn");
      b.type = "button";
      append(b, i ? el("span", "br-toc-n", String(i)) : null, sec.label);
      b.addEventListener("click", function () {
        scrollTo(sec.id);
        try { history.replaceState(null, "", "#briefing/" + sec.id); } catch (_) { /* file:// may refuse */ }
        setActive(sec.id);
      });
      buttons[sec.id] = b;
      li.appendChild(b);
      ol.appendChild(li);
    });
    nav.appendChild(ol);
    return nav;
  }

  var built = null;

  function setActive(id) {
    if (!built) return;
    SECTIONS.forEach(function (sec) {
      var b = built.buttons[sec.id];
      if (sec.id === id) b.setAttribute("aria-current", "true"); else b.removeAttribute("aria-current");
    });
    /* On a narrow screen the contents are a horizontal strip: keep the
       current entry inside it without moving the page. */
    var cur = built.buttons[id], strip = built.strip;
    if (strip.scrollWidth > strip.clientWidth) {
      var left = cur.offsetLeft - strip.offsetLeft;
      if (left < strip.scrollLeft || left + cur.offsetWidth > strip.scrollLeft + strip.clientWidth) {
        strip.scrollLeft = Math.max(0, left - 16);
      }
    }
  }

  /* Highlight the last section whose top has passed a line a fifth of the way
     down the viewport. */
  function watch(article) {
    if (typeof IntersectionObserver === "undefined") return;
    var visible = Object.create(null);
    var obs = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) { visible[en.target.dataset.section] = en.isIntersecting; });
      for (var i = 0; i < SECTIONS.length; i++) {
        if (visible[SECTIONS[i].id]) { setActive(SECTIONS[i].id); return; }
      }
    }, { rootMargin: "-20% 0px -60% 0px" });
    Array.prototype.forEach.call(article.querySelectorAll("[data-section]"), function (n) { obs.observe(n); });
  }

  function build(host) {
    var buttons = Object.create(null);
    var shell = el("div", "br-shell");
    var article = el("article", "br-article");
    append(article, buildLede(), buildAct(), buildScores(), buildWhere(), buildFamilies(), buildDrift(), buildMethod());
    append(shell, buildToc(buttons), article);
    D.clear(host).appendChild(shell);
    built = { buttons: buttons, strip: shell.querySelector(".br-toc-list") };
    setActive("lede");
    watch(article);
  }

  D.screens["briefing"] = {
    render: function (host, params) {
      if (!built || !host.firstChild) build(host);
      D.setCaseList(D.headline());
      var id = params && params[0];
      var known = SECTIONS.some(function (sec) { return sec.id === id; });
      if (known) {
        setActive(id);
        /* The router scrolls to the top after a screen change, so the jump
           waits for it. */
        setTimeout(function () { scrollTo(id); }, 0);
      }
    }
  };
})(DOPPEL);
