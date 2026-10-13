/* doppel dashboard — triage screen.
 *
 * The default landing screen: the corpus at a glance, then the ranked list of
 * pairs to act on first, with a filter rail on one side and the places the
 * duplication concentrates on the other.
 *
 * Every filter is one pass over the edges: each edge is tested against every
 * facet once, and the same pass fills the facet counts (a kind's count is what
 * the list would hold if that kind were checked, everything else unchanged).
 * Only the visible slice of the result is rendered. Ordering is the payload's
 * rank order, or a stable sort on one reported quantity with the edge index as
 * the tie-break — never a recomputed key.
 */
(function (D) {
  "use strict";

  var el = D.el, append = D.append, fixed = D.fixed, comma = D.comma, plural = D.plural;
  var PAGE = 25;

  var SHOWS = [
    { id: "headline", label: "Headline", note: "merge-worthy, no init/main pairs, hidden kinds off" },
    { id: "merge", label: "All merge-worthy", note: "every pair past the merge verdict" },
    { id: "all", label: "All scored pairs", note: "everything the comparator scored" }
  ];
  var RANKS = [
    { id: "rank", label: "corroborated evidence" },
    { id: "shape", label: "code-shape" },
    { id: "overlap", label: "structural overlap" },
    { id: "containment", label: "containment" }
  ];

  /* ── State ───────────────────────────────────────────────────────────── */

  /* Which kinds are checked: the KINDS defaults, or every kind. */
  function kindSet(all) {
    var on = Object.create(null);
    D.KINDS.forEach(function (k) { on[k.id] = all || !k.hidden; });
    return on;
  }
  function defaultKinds() { return kindSet(false); }
  function allKinds() { return kindSet(true); }
  function defaults() {
    return { mergeOnly: true, kinds: defaultKinds(), entry: false, rankBy: "rank",
      within: true, across: true, pkg: "", text: "", quad: "" };
  }

  var S = defaults();
  var built = null;      // cached DOM
  var lastKey = null;    // params the state was last preset from
  var order = [];        // filtered edge indexes, in display order
  var shown = PAGE;      // rows rendered
  var openIdx = -1;      // the expanded edge, or -1

  function sameKinds(a, b) {
    for (var i = 0; i < D.KINDS.length; i++) { var k = D.KINDS[i].id; if (!a[k] !== !b[k]) return false; }
    return true;
  }
  function showOf() {
    if (!S.mergeOnly) return "all";
    return (sameKinds(S.kinds, defaultKinds()) && !S.entry) ? "headline" : "merge";
  }
  function isDefault() {
    var d = defaults();
    return S.mergeOnly === d.mergeOnly && sameKinds(S.kinds, d.kinds) && S.entry === d.entry &&
      S.rankBy === d.rankBy && S.within && S.across && !S.pkg && !S.text && !S.quad;
  }

  /* Lower-cased qualified names, for the text filter. Built once. */
  var lowerName = null;
  function names() {
    if (!lowerName) lowerName = D.units.map(function (u) { return D.qualified(u).toLowerCase(); });
    return lowerName;
  }

  /* Every package any unit lives in, sorted. */
  var pkgNames = null;
  function allPackages() {
    if (pkgNames) return pkgNames;
    var seen = Object.create(null);
    D.units.forEach(function (u) { seen[u.package] = true; });
    D.packages.forEach(function (p) { seen[p.name] = true; });
    pkgNames = Object.keys(seen).sort(D.cmp);
    return pkgNames;
  }

  /* ── The one pass ────────────────────────────────────────────────────── */

  function compute() {
    var edges = D.edges, units = D.units, nm = names();
    var text = S.text.trim().toLowerCase();
    var kindCount = Object.create(null), pkgCount = Object.create(null);
    D.KINDS.forEach(function (k) { kindCount[k.id] = 0; });
    var entryCount = 0, out = [];
    for (var i = 0; i < edges.length; i++) {
      var e = edges[i];
      if (S.mergeOnly && !e.merge) continue;
      var ua = units[e.a], ub = units[e.b];
      var k = e.kind || "";
      var pKind = !!S.kinds[k];
      var isEntry = D.isEntryPoint(ua) || D.isEntryPoint(ub);
      var pEntry = S.entry || !isEntry;
      var pScope = e.cross ? S.across : S.within;
      var pPkg = !S.pkg || ua.package === S.pkg || ub.package === S.pkg;
      var pText = !text || nm[e.a].indexOf(text) >= 0 || nm[e.b].indexOf(text) >= 0;
      var pQuad = !S.quad || D.quadOf(e) === S.quad;
      var rest = pScope && pText && pQuad;
      if (rest && pEntry && pPkg && k in kindCount) kindCount[k]++;
      if (rest && pKind && pPkg && isEntry) entryCount++;
      if (rest && pKind && pEntry) {
        pkgCount[ua.package] = (pkgCount[ua.package] || 0) + 1;
        if (ub.package !== ua.package) pkgCount[ub.package] = (pkgCount[ub.package] || 0) + 1;
        if (pPkg) out.push(i);
      }
    }
    if (S.rankBy !== "rank") {
      var key = S.rankBy;
      out.sort(function (x, y) { return (edges[y][key] - edges[x][key]) || (x - y); });
    }
    return { order: out, kindCount: kindCount, entryCount: entryCount, pkgCount: pkgCount };
  }

  /* ── Small builders ──────────────────────────────────────────────────── */

  var uid = 0;
  function field(labelText, control) {
    var id = "tr-f" + (++uid);
    control.id = id;
    var wrap = el("div", "tr-field");
    var l = el("label", "control-label", labelText);
    l.htmlFor = id;
    return append(wrap, l, control);
  }
  function check(name, type, text, countNode) {
    var lab = el("label", "tr-check");
    var inp = el("input");
    inp.type = type;
    inp.name = name;
    append(lab, inp, el("span", "tr-check-text", text), countNode || null);
    return { label: lab, input: inp };
  }
  function group(legend) {
    var fs = el("fieldset", "tr-group");
    fs.appendChild(el("legend", "control-label", legend));
    return fs;
  }

  /* "a/b/x.go:12 · :40" when both live in one file, else both in full. */
  function locations(ua, ub) {
    return ua.file === ub.file ? D.where(ua) + " · :" + ub.line : D.where(ua) + " · " + D.where(ub);
  }

  /* What the members of a family have in common: one file, one package, or
     the number of packages they span. */
  function familyWhere(f) {
    var files = Object.create(null), pkgs = Object.create(null), nf = 0, np = 0;
    f.members.forEach(function (m) {
      var u = D.unit(m);
      if (!files[u.file]) { files[u.file] = true; nf++; }
      if (!pkgs[u.package]) { pkgs[u.package] = true; np++; }
    });
    if (nf === 1) return "all in " + D.unit(f.members[0]).file;
    if (np === 1) return "all in package " + D.unit(f.members[0]).package + ", " + plural(nf, "file");
    return "across " + plural(np, "package");
  }

  /* ── Build once ──────────────────────────────────────────────────────── */

  function build(host) {
    D.clear(host);
    var root = el("div", "tr");
    host.appendChild(root);

    root.appendChild(buildStrip());

    var grid = el("div", "tr-grid");
    root.appendChild(grid);

    var B = { root: root };
    grid.appendChild(buildRail(B));
    grid.appendChild(buildMain(B));
    grid.appendChild(buildContext());
    return B;
  }

  function buildStrip() {
    var f = D.facts;
    var strip = el("div", "tr-strip");
    strip.setAttribute("role", "list");
    var scored = f.pairs || D.edges.length;
    var tiles = [
      D.stat(comma(D.mergeCount()), "Merge-worthy pairs", "of " + comma(scored) + " scored pairs: alike bodies that also share callers, callees, concepts or package. " +
        comma(D.headline().length) + " remain after hiding init/main pairs and the kinds that are not merges.", null),
      D.stat(comma(f.familyCount || 0), "Near-duplicate families",
        (f.familyCount ? "groups where every two members are alike, covering " + plural(f.familyFuncs || 0, "function") +
          "; the largest has " + comma(f.familyLargest || 0) + " members." : "no group of three or more functions where every two are alike."), null),
      D.stat(f.compression > 0 ? fixed(f.compression, 2) : "—", "Structural repetition",
        "canonical syntax nodes per distinct subtree shape. 1.00 would mean no shape is written twice; higher means more of the code repeats shapes that already exist.",
        f.compression > 0 ? "×" : null),
      D.stat(f.nnScored ? fixed(f.nnP50) : "—", "Nearest-neighbour median",
        f.nnScored ? "half of the " + comma(f.nnScored) + " functions retrieval scored have a twin at least this alike in code-shape; the most-copied tenth reach " +
          fixed(f.nnP90) + " (p90). Functions retrieval never paired are not counted." : "retrieval scored no function against another.",
        f.nnScored ? "code-shape" : null),
      D.stat(comma(f.misfits || 0), "Habitat misfits",
        "functions written unlike the rest of their package and of its parent directory" +
          (f.misfitsExcused ? "; " + comma(f.misfitsExcused) + " more are unlike their package but fit their subsystem, and are excused." : "."),
        f.misfitsExcused ? "+" + comma(f.misfitsExcused) + " excused" : null)
    ];
    tiles.forEach(function (t) { t.classList.add("tr-tile"); t.setAttribute("role", "listitem"); strip.appendChild(t); });
    return strip;
  }

  function buildRail(B) {
    var rail = el("aside", "tr-rail");
    rail.setAttribute("aria-label", "Filters");
    var det = el("details", "tr-rail-box");
    /* Open on a wide screen, where the summary is not drawn; collapsed above
       the list on a narrow one. Widening re-opens it so it cannot hide. */
    var mq = window.matchMedia ? window.matchMedia("(max-width: 760px)") : null;
    det.open = !(mq && mq.matches);
    if (mq && mq.addEventListener) mq.addEventListener("change", function () { if (!mq.matches) det.open = true; });
    var sum = el("summary", "tr-rail-summary", "Filters");
    B.railSummaryN = el("span", "tr-rail-summary-n muted");
    sum.appendChild(B.railSummaryN);
    det.appendChild(sum);
    var body = el("div", "tr-rail-body");
    det.appendChild(body);
    rail.appendChild(det);

    /* Show */
    var gShow = group("Show");
    B.show = {};
    var showCounts = { headline: D.headline().length, merge: D.mergeCount(), all: D.edges.length };
    SHOWS.forEach(function (s) {
      var c = check("tr-show", "radio", s.label, el("span", "tr-n mono", comma(showCounts[s.id])));
      c.input.value = s.id;
      c.label.title = s.note;
      c.input.addEventListener("change", function () {
        if (!c.input.checked) return;
        if (s.id === "headline") { S.mergeOnly = true; S.kinds = defaultKinds(); S.entry = false; }
        else if (s.id === "merge") { S.mergeOnly = true; S.kinds = allKinds(); S.entry = true; }
        else { S.mergeOnly = false; S.kinds = allKinds(); S.entry = true; }
        refresh(true);
      });
      B.show[s.id] = c.input;
      gShow.appendChild(c.label);
    });
    body.appendChild(gShow);

    /* Rank by */
    var sel = el("select", "tr-select");
    RANKS.forEach(function (r) { var o = el("option", null, r.label); o.value = r.id; sel.appendChild(o); });
    sel.addEventListener("change", function () { S.rankBy = sel.value; refresh(true); });
    B.rankBy = sel;
    body.appendChild(field("Rank by", sel));

    /* Pair kind */
    var gKind = group("Pair kind");
    B.kinds = {};
    B.kindN = {};
    D.KINDS.forEach(function (k) {
      var n = el("span", "tr-n mono");
      var c = check("tr-kind", "checkbox", k.label, n);
      c.input.addEventListener("change", function () { S.kinds[k.id] = c.input.checked; refresh(true); });
      B.kinds[k.id] = c.input;
      B.kindN[k.id] = n;
      gKind.appendChild(c.label);
    });
    var en = el("span", "tr-n mono");
    var ce = check("tr-entry", "checkbox", "include init/main pairs", en);
    ce.input.addEventListener("change", function () { S.entry = ce.input.checked; refresh(true); });
    B.entry = ce.input;
    B.entryN = en;
    gKind.appendChild(ce.label);
    gKind.appendChild(el("p", "control-note", "Kinds annotate; unchecking hides a pair, it never re-ranks one. Counts are what each box would show with everything else as set."));
    body.appendChild(gKind);

    /* Scope */
    var gScope = group("Scope");
    var cw = check("tr-within", "checkbox", "within a package");
    var ca = check("tr-across", "checkbox", "across packages");
    cw.input.addEventListener("change", function () { S.within = cw.input.checked; refresh(true); });
    ca.input.addEventListener("change", function () { S.across = ca.input.checked; refresh(true); });
    B.within = cw.input;
    B.across = ca.input;
    append(gScope, cw.label, ca.label);
    body.appendChild(gScope);

    /* Package */
    var ps = el("select", "tr-select");
    var any = el("option", null, "any");
    any.value = "";
    ps.appendChild(any);
    B.pkgOpts = {};
    allPackages().forEach(function (p) {
      var o = el("option");
      o.value = p;
      ps.appendChild(o);
      B.pkgOpts[p] = o;
    });
    ps.addEventListener("change", function () { S.pkg = ps.value; refresh(true); });
    B.pkg = ps;
    body.appendChild(field("Package", ps));

    /* Text */
    var tx = el("input", "tr-text");
    tx.type = "search";
    tx.placeholder = "part of either name";
    tx.autocomplete = "off";
    tx.addEventListener("input", function () { S.text = tx.value; refresh(true); });
    B.text = tx;
    body.appendChild(field("Function name", tx));

    var reset = el("button", "btn tr-reset", "Reset filters");
    reset.type = "button";
    reset.addEventListener("click", resetAll);
    B.reset = reset;
    body.appendChild(reset);
    return rail;
  }

  function buildMain(B) {
    var main = el("div", "tr-main");
    var head = el("div", "tr-main-head");
    var h = el("h2", "section-title", "Act on these first");
    h.id = "tr-list-title";
    head.appendChild(h);
    head.appendChild(D.scoreKey(true));
    main.appendChild(head);

    B.chips = el("div", "tr-active");
    main.appendChild(B.chips);

    B.list = el("ol", "tr-list");
    B.list.setAttribute("aria-labelledby", "tr-list-title");
    B.list.addEventListener("keydown", onListKey);
    main.appendChild(B.list);

    B.empty = el("div", "tr-empty card");
    B.empty.hidden = true;
    main.appendChild(B.empty);

    var foot = el("div", "tr-foot");
    B.count = el("p", "tr-count muted");
    B.count.setAttribute("aria-live", "polite");
    B.more = el("button", "btn tr-more", "Show 25 more");
    B.more.type = "button";
    B.more.addEventListener("click", function () {
      var first = shown;
      shown += PAGE;
      renderRows(first);
      var btn = B.list.children[first] && B.list.children[first].querySelector(".tr-row");
      if (btn) btn.focus();
    });
    append(foot, B.count, B.more);
    main.appendChild(foot);
    return main;
  }

  function buildContext() {
    var ctx = el("aside", "tr-context");
    ctx.setAttribute("aria-label", "Corpus context");

    /* Where it concentrates */
    var c1 = el("section", "card tr-ctx");
    var h1 = el("div", "card-head");
    append(h1, el("h3", "card-title", "Where it concentrates"), el("span", "card-sub", "merge-worthy pairs inside one package"));
    c1.appendChild(h1);
    var hot = D.packageStats().filter(function (p) { return p.withinMerge > 0; });
    var top = hot.slice(0, 8);
    if (!top.length) {
      c1.appendChild(el("p", "muted tr-ctx-p", "No package holds a merge-worthy pair inside itself."));
    } else {
      var max = top[0].withinMerge;
      var ul = el("ul", "tr-hot");
      top.forEach(function (p) {
        var a = D.link(D.atlasHref(p.name), "tr-hot-row");
        a.appendChild(el("span", "tr-hot-name mono", p.name));
        a.appendChild(D.bar(p.withinMerge / max, "neutral", comma(p.withinMerge)));
        a.setAttribute("aria-label", p.name + ": " + plural(p.withinMerge, "merge-worthy pair") + " inside; open in the atlas");
        ul.appendChild(append(el("li"), a));
      });
      c1.appendChild(ul);
      if (hot.length > top.length) c1.appendChild(el("p", "muted tr-ctx-small", "Top 8 of " + comma(hot.length) + " packages with one or more."));
    }
    var seam = D.seams()[0];
    if (seam) {
      var p = el("p", "tr-ctx-p");
      append(p, "Heaviest cross-package link: ", el("span", "mono", seam.a + " ↔ " + seam.b), ", " +
        (seam.merge ? plural(seam.merge, "merge-worthy pair") + " of " + plural(seam.pairs, "scored pair") : plural(seam.pairs, "scored pair") + ", none merge-worthy") + ". ",
        D.link(D.atlasHref(seam.a), null, "See the atlas →"));
      c1.appendChild(p);
    }
    ctx.appendChild(c1);

    /* Families */
    if (D.families.length) {
      var c2 = el("section", "card tr-ctx");
      var h2 = el("div", "card-head");
      append(h2, el("h3", "card-title", "Strongest families"), el("span", "card-sub", "by evidence · " + comma(D.families.length) + " total"));
      c2.appendChild(h2);
      var fl = el("ul", "tr-fams");
      D.families.slice(0, 4).forEach(function (f, i) {
        var li = el("li", "tr-fam");
        var first = D.unit(f.members[0]);
        var head = el("div", "tr-fam-head");
        append(head, el("span", "tr-fam-id", "F" + (i + 1) + " · " + plural(f.members.length, "function")),
          el("span", "mono tr-fam-min", "every pair ≥ " + fixed(f.minEdge)));
        li.appendChild(head);
        li.appendChild(el("div", "tr-fam-where muted", familyWhere(f)));
        var shownNames = f.members.slice(0, 3).map(function (m) { return D.qualified(D.unit(m)); });
        var more = f.members.length - shownNames.length;
        li.appendChild(el("div", "mono tr-fam-names", shownNames.join(", ") + (more > 0 ? " and " + more + " more" : "")));
        li.appendChild(D.link(D.fnHref(first.id), "tr-fam-link", "Open " + D.qualified(first) + " →"));
        fl.appendChild(li);
      });
      c2.appendChild(fl);
      c2.appendChild(el("p", "muted tr-ctx-small", "Every two members of a family clear its floor in code-shape — open two files to falsify it."));
      ctx.appendChild(c2);
    }

    /* Drift */
    var f = D.facts;
    var c3 = el("section", "card tr-ctx");
    c3.appendChild(append(el("div", "card-head"), el("h3", "card-title", "Drift")));
    c3.appendChild(el("p", "tr-ctx-p",
      plural(f.unusual || 0, "function") + " realize a concept unlike its other members here; " +
      plural(f.misfits || 0, "function") + " sit outside their package's habitat" +
      (f.misfitsExcused ? " (" + comma(f.misfitsExcused) + " more excused by subsystem)." : ".")));
    var seeds = D.unusedSeeds;
    if (seeds.length) {
      var head = seeds.slice(0, 4);
      var rest = seeds.length - head.length;
      c3.appendChild(el("p", "tr-ctx-p", "No practice here for " +
        (rest > 0 ? head.join(", ") + " and " + plural(rest, "more seed") : head.length > 1 ? head.slice(0, -1).join(", ") + " and " + head[head.length - 1] : head[0]) + "."));
    }
    c3.appendChild(D.link("#briefing/drift", "tr-ctx-link", "Open local practice →"));
    ctx.appendChild(c3);
    return ctx;
  }

  /* ── Update ──────────────────────────────────────────────────────────── */

  function syncControls(r) {
    var B = built, so = showOf();
    Object.keys(B.show).forEach(function (id) { B.show[id].checked = id === so; });
    B.rankBy.value = S.rankBy;
    D.KINDS.forEach(function (k) {
      B.kinds[k.id].checked = !!S.kinds[k.id];
      B.kindN[k.id].textContent = comma(r.kindCount[k.id] || 0);
    });
    B.entry.checked = S.entry;
    B.entryN.textContent = comma(r.entryCount);
    B.within.checked = S.within;
    B.across.checked = S.across;
    allPackages().forEach(function (p) {
      var t = p + " (" + comma(r.pkgCount[p] || 0) + ")";
      if (B.pkgOpts[p].textContent !== t) B.pkgOpts[p].textContent = t;
    });
    B.pkg.value = S.pkg;
    if (B.text.value !== S.text) B.text.value = S.text;
    B.reset.hidden = isDefault();
    B.railSummaryN.textContent = isDefault() ? " · default" : " · filtered";

    D.clear(B.chips);
    if (S.quad) {
      var q = D.QUADS.filter(function (x) { return x.id === S.quad; })[0];
      var chip = el("button", "chip chip-outline tr-quad-chip");
      chip.type = "button";
      append(chip, "Quadrant: " + q.label + " ", el("span", "tr-x", "×"));
      chip.setAttribute("aria-label", "Remove quadrant filter " + q.label);
      chip.title = q.note;
      chip.addEventListener("click", function () { S.quad = ""; refresh(true); });
      B.chips.appendChild(chip);
      B.chips.appendChild(el("span", "muted tr-quad-note", q.note + " (cut at " + fixed(D.QUAD_CUT) + " on both scores)"));
    }
    B.chips.hidden = !S.quad;
  }

  function refresh(resetPaging) {
    var r = compute();
    order = r.order;
    if (resetPaging) shown = PAGE;
    if (order.indexOf(openIdx) < 0) openIdx = -1;
    D.setCaseList(order);
    syncControls(r);
    D.clear(built.list);
    renderRows(0);
    renderEmpty();
  }

  function scopeWord() {
    var so = showOf();
    return so === "all" ? "scored pairs" : so === "headline" ? "headline pairs" : "merge-worthy pairs";
  }

  function renderRows(from) {
    var B = built;
    var end = Math.min(order.length, shown);
    for (var i = from; i < end; i++) B.list.appendChild(row(order[i], i));
    var base = S.mergeOnly ? (showOf() === "headline" ? D.headline().length : D.mergeCount()) : D.edges.length;
    B.count.textContent = order.length
      ? "Showing " + comma(end) + " of " + comma(order.length) + " " + scopeWord() +
        (order.length < base ? " (" + comma(base) + " before the filters)" : "") +
        ". The page draws every scored pair; --top and --max-per-func bound the text report only."
      : "";
    B.more.hidden = end >= order.length;
    B.more.textContent = "Show " + Math.min(PAGE, order.length - end) + " more";
  }

  function renderEmpty() {
    var B = built;
    B.empty.hidden = order.length > 0;
    B.list.hidden = order.length === 0;
    if (order.length) return;
    D.clear(B.empty);
    var why = [];
    if (S.mergeOnly) why.push(showOf() === "headline" ? "the headline view (merge-worthy, default kinds, no init/main)" : "merge-worthy pairs only");
    var off = D.KINDS.filter(function (k) { return !S.kinds[k.id]; }).map(function (k) { return k.label; });
    if (off.length && showOf() !== "headline") why.push("hidden kinds: " + off.join(", "));
    if (!S.within) why.push("no within-package pairs");
    if (!S.across) why.push("no cross-package pairs");
    if (S.pkg) why.push("package " + S.pkg);
    if (S.text.trim()) why.push("names containing “" + S.text.trim() + "”");
    if (S.quad) why.push("quadrant " + S.quad);
    B.empty.appendChild(el("p", "tr-empty-title", D.edges.length ? "No pair passes these filters." : "This run scored no pairs."));
    if (why.length && D.edges.length) B.empty.appendChild(el("p", "muted", "Active: " + why.join("; ") + "."));
    if (!isDefault()) {
      var b = el("button", "btn btn-primary", "Reset filters");
      b.type = "button";
      b.addEventListener("click", resetAll);
      B.empty.appendChild(b);
    } else if (S.mergeOnly && D.edges.length) {
      var w = el("button", "btn", "Show all scored pairs");
      w.type = "button";
      w.addEventListener("click", function () { S.mergeOnly = false; S.kinds = allKinds(); S.entry = true; refresh(true); });
      B.empty.appendChild(w);
    }
  }

  function row(idx, pos) {
    var e = D.edges[idx], ua = D.unit(e.a), ub = D.unit(e.b);
    var li = el("li", "tr-item");
    var btn = el("button", "tr-row");
    btn.type = "button";
    btn.dataset.idx = idx;
    var open = idx === openIdx;
    btn.setAttribute("aria-expanded", open ? "true" : "false");
    var pid = "tr-p" + idx;
    btn.setAttribute("aria-controls", pid);

    var n = el("span", "rank-n tr-rank", String(pos + 1));
    var who = el("span", "tr-who");
    append(who,
      el("span", "tr-name mono", D.qualified(ua)),
      el("span", "tr-name mono", D.qualified(ub)),
      el("span", "tr-where mono muted", locations(ua, ub)));
    var chips = el("span", "tr-chips");
    if (e.merge) chips.appendChild(D.chip("merge-worthy", "solid"));
    if (e.kind) chips.appendChild(D.chip(e.kind, "outline"));
    if (e.cross) chips.appendChild(D.chip("cross-package", "outline"));
    if (chips.firstChild) who.appendChild(chips);

    var sc = el("span", "tr-scores");
    var s1 = D.bar(e.shape, "shape");
    s1.setAttribute("aria-label", "code-shape " + fixed(e.shape));
    var s2 = D.bar(e.overlap, "overlap");
    s2.setAttribute("aria-label", "overlap " + fixed(e.overlap));
    append(sc, s1, s2, el("span", "tr-minor mono muted", "containment " + fixed(e.containment) + " · evidence " + comma(Math.round(e.total)) + " nats"));

    append(btn, n, who, sc);
    li.appendChild(btn);
    if (open) li.appendChild(preview(e, pid));
    btn.addEventListener("click", function () { toggle(idx, li, btn, pid); });
    if (open) li.classList.add("is-open");
    return li;
  }

  function toggle(idx, li, btn, pid) {
    var B = built;
    if (openIdx >= 0 && openIdx !== idx) {
      var prev = B.list.querySelector(".tr-item.is-open");
      if (prev) {
        prev.classList.remove("is-open");
        var pb = prev.querySelector(".tr-row");
        pb.setAttribute("aria-expanded", "false");
        var pp = prev.querySelector(".tr-preview");
        if (pp) prev.removeChild(pp);
      }
    }
    if (openIdx === idx) {
      openIdx = -1;
      li.classList.remove("is-open");
      btn.setAttribute("aria-expanded", "false");
      var p = li.querySelector(".tr-preview");
      if (p) li.removeChild(p);
    } else {
      openIdx = idx;
      li.classList.add("is-open");
      btn.setAttribute("aria-expanded", "true");
      li.appendChild(preview(D.edges[idx], pid));
    }
    btn.focus();
  }

  function preview(e, pid) {
    var ua = D.unit(e.a), ub = D.unit(e.b);
    var p = el("div", "tr-preview");
    p.id = pid;
    if (e.explain) {
      p.appendChild(append(el("p", "tr-explain"), el("span", "tr-k", "explain "), e.explain));
    }
    if (e.kindNote) p.appendChild(append(el("p", "tr-kindnote"), el("span", "tr-k", (e.kind || "kind") + " "), e.kindNote));
    var facts = el("dl", "tr-facts");
    function fact(k, v) { facts.appendChild(append(el("div", "tr-fact"), el("dt", null, k), el("dd", "mono", v))); }
    fact("retrieved by", e.channels && e.channels.length ? e.channels.join(" · ") : "—");
    fact("evidence", comma(Math.round(e.total)) + " nats");
    fact("trophic", fixed(e.trophic));
    fact("containment", fixed(e.containment));
    p.appendChild(facts);
    if (e.reasons && e.reasons.length) {
      var ul = el("ul", "tr-reasons");
      e.reasons.slice(0, 3).forEach(function (r) { ul.appendChild(el("li", null, r)); });
      p.appendChild(ul);
      if (e.reasons.length > 3) p.appendChild(el("p", "muted tr-small", (e.reasons.length - 3) + " more reasons in the case file."));
    } else if (D.facts.detailOmitted) {
      p.appendChild(el("p", "muted tr-small", "Reasons are carried only for the higher-ranked pairs; this one has scores alone."));
    }
    var acts = el("div", "tr-actions");
    append(acts,
      D.link(D.caseHref(e.a, e.b), "btn btn-primary", "Open case file →"),
      D.link(D.fnHref(ua.id), "btn", "Open A · " + ua.name),
      D.link(D.fnHref(ub.id), "btn", "Open B · " + ub.name));
    p.appendChild(acts);
    return p;
  }

  /* ↑/↓ move between row buttons; Home/End jump to the ends of what is shown. */
  function onListKey(ev) {
    var t = ev.target;
    if (!t.classList || !t.classList.contains("tr-row")) return;
    var rows = built.list.querySelectorAll(".tr-row");
    var i = Array.prototype.indexOf.call(rows, t), j = -1;
    if (ev.key === "ArrowDown") j = Math.min(rows.length - 1, i + 1);
    else if (ev.key === "ArrowUp") j = Math.max(0, i - 1);
    else if (ev.key === "Home") j = 0;
    else if (ev.key === "End") j = rows.length - 1;
    if (j < 0) return;
    ev.preventDefault();
    rows[j].focus();
  }

  function resetAll() {
    S = defaults();
    openIdx = -1;
    refresh(true);
    if (built.text) built.text.blur();
  }

  /* ── Presets from the route ──────────────────────────────────────────── */

  function preset(params) {
    S = defaults();
    openIdx = -1;
    if (params[0] === "pkg" && params[1]) {
      S.pkg = params[1];
      if (!built.pkgOpts[S.pkg]) {   // unknown package: keep the filter, say so via the empty state
        var o = el("option");
        o.value = S.pkg;
        built.pkg.appendChild(o);
        built.pkgOpts[S.pkg] = o;
        pkgNames = allPackages().concat([S.pkg]).sort(D.cmp);
      }
    } else if (params[0] === "quad" && params[1]) {
      if (D.QUADS.some(function (q) { return q.id === params[1]; })) {
        S.quad = params[1];
        S.mergeOnly = false; S.kinds = allKinds(); S.entry = true;   // a quadrant is over every scored pair
      }
    }
  }

  D.screens["triage"] = {
    render: function (host, params) {
      if (!built || !host.contains(built.root)) built = build(host);
      var key = (params || []).join("/");
      if (key !== lastKey) {
        lastKey = key;
        preset(params || []);
        refresh(true);
      } else {
        D.setCaseList(order);
      }
    }
  };
})(DOPPEL);
