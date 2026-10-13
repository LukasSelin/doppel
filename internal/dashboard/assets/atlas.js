/* doppel dashboard — atlas screen.
 *
 * Answers "where in the repo does the duplication live, and which package
 * boundaries does it cross". Territories (packages, or directories) are a
 * squarified treemap whose area is the function count and whose tint is one
 * per-function rate. Cross-territory evidence is a ranked list of seams, not
 * a geometry: the cartogram this replaced painted it on shared borders, and a
 * border width is a number nobody can read.
 *
 * Everything here is a pure function of the payload plus the container width,
 * and every list has a total order, so an unchanged tree draws the same map.
 */
(function (D) {
  "use strict";

  var el = D.el, append = D.append;

  /* ── Territory models (one pass over units and edges per mode, cached) ── */

  var MODES = [
    { id: "pkg", label: "by package", noun: "package", nouns: "packages" },
    { id: "dir", label: "by directory", noun: "directory", nouns: "directories" }
  ];
  function modeById(id) { return id === "dir" ? MODES[1] : MODES[0]; }

  function dirOf(file) {
    var i = (file || "").lastIndexOf("/");
    return i < 0 ? "." : file.slice(0, i);
  }

  var models = Object.create(null);
  function model(mode) {
    if (models[mode]) return models[mode];
    var keyOf = mode === "dir" ? function (u) { return dirOf(u.file); } : function (u) { return u.package; };
    var by = Object.create(null);
    var unitT = new Array(D.units.length);
    D.units.forEach(function (u, i) {
      var k = keyOf(u);
      var t = by[k] || (by[k] = { name: k, functions: 0, misfits: 0, norm: -1, within: 0, withinMerge: 0,
        withinEdges: [], cross: 0, crossMerge: 0, families: 0, seams: [] });
      t.functions++;
      if (u.misfit) t.misfits++;
      unitT[i] = t;
    });
    if (mode === "pkg") {
      D.packages.forEach(function (p) { if (by[p.name]) by[p.name].norm = p.norm; });
    }
    var seamBy = Object.create(null);
    D.edges.forEach(function (e, i) {
      var ta = unitT[e.a], tb = unitT[e.b];
      if (ta === tb) {
        ta.within++;
        if (e.merge) ta.withinMerge++;
        ta.withinEdges.push(i); // edges are rank-descending, so this list is too
        return;
      }
      ta.cross++; tb.cross++;
      if (e.merge) { ta.crossMerge++; tb.crossMerge++; }
      var a = ta.name, b = tb.name;
      if (b < a) { var t = a; a = b; b = t; }
      var k = a + "\u0000" + b;
      var s = seamBy[k] || (seamBy[k] = { a: a, b: b, pairs: 0, merge: 0, edges: [] });
      s.pairs++;
      if (e.merge) s.merge++;
      s.edges.push(i);
    });
    // The same ordering rule core's seams() uses for packages.
    var seams = Object.keys(seamBy).map(function (k) { return seamBy[k]; }).sort(function (x, y) {
      return (y.merge - x.merge) || (y.pairs - x.pairs) || D.cmp(x.a, y.a) || D.cmp(x.b, y.b);
    });
    seams.forEach(function (s) { by[s.a].seams.push(s); by[s.b].seams.push(s); });
    D.families.forEach(function (f) {
      var seen = Object.create(null);
      f.members.forEach(function (m) {
        var t = unitT[m];
        if (t && !seen[t.name]) { seen[t.name] = true; t.families++; }
      });
    });
    var list = Object.keys(by).map(function (k) { return by[k]; }).sort(function (x, y) {
      return (y.functions - x.functions) || D.cmp(x.name, y.name);
    });
    var withinMerge = 0, crossMerge = 0;
    list.forEach(function (t) { withinMerge += t.withinMerge; });
    seams.forEach(function (s) { crossMerge += s.merge; });
    return (models[mode] = { mode: mode, list: list, by: by, seams: seams, withinMerge: withinMerge, crossMerge: crossMerge });
  }

  /* ── Squarified treemap (Bruls, Huizing, van Wijk) ─────────────────────
     Pure: (values sorted descending, width, height) → one rect per value, in
     input order. Each row is grown along the shorter side for as long as that
     improves its worst aspect ratio. */
  function squarify(values, W, H) {
    var n = values.length, total = 0, i;
    for (i = 0; i < n; i++) total += values[i];
    var rects = new Array(n);
    if (!(total > 0) || !(W > 0) || !(H > 0)) return rects;
    var scale = (W * H) / total;
    var area = values.map(function (v) { return v * scale; });
    function worst(sum, max, min, side) {
      var s2 = sum * sum, w2 = side * side;
      return Math.max((w2 * max) / s2, s2 / (w2 * min));
    }
    var x = 0, y = 0, w = W, h = H;
    i = 0;
    while (i < n) {
      var side = Math.min(w, h);
      var sum = area[i], best = worst(sum, area[i], area[i], side), j = i + 1;
      while (j < n) {
        var next = sum + area[j], r = worst(next, area[i], area[j], side);
        if (r > best) break;
        best = r; sum = next; j++;
      }
      var k;
      if (w >= h) {
        var cw = sum / h, cy = y;
        for (k = i; k < j; k++) { var kh = area[k] / cw; rects[k] = { x: x, y: cy, w: cw, h: kh }; cy += kh; }
        x += cw; w -= cw;
      } else {
        var ch = sum / w, cx = x;
        for (k = i; k < j; k++) { var kw = area[k] / ch; rects[k] = { x: cx, y: y, w: kw, h: ch }; cx += kw; }
        y += ch; h -= ch;
      }
      i = j;
    }
    return rects;
  }

  /* ── Tints: five steps of the accent ramp, bin edges stated as numbers ── */

  var TINTS = [
    { id: "merge", label: "merge-worthy pairs per function",
      value: function (t) { return t.withinMerge / t.functions; },
      step: function (v) { return v <= 0 ? 0 : v < 0.1 ? 1 : v < 0.25 ? 2 : v < 0.5 ? 3 : 4; },
      bins: ["0", "< 0.10", "0.10–0.25", "0.25–0.50", "≥ 0.50"],
      low: "none", high: "more duplication per function",
      note: "merge-worthy pairs with both functions inside the territory, divided by its functions" },
    { id: "norm", label: "habitat norm", pkgOnly: true,
      value: function (t) { return t.norm; },
      step: function (v) { return v < 0 ? -1 : v >= 0.95 ? 0 : v >= 0.9 ? 1 : v >= 0.8 ? 2 : v >= 0.65 ? 3 : 4; },
      bins: ["≥ 0.95", "0.90–0.95", "0.80–0.90", "0.65–0.80", "< 0.65"],
      low: "uniform", high: "functions unlike their neighbours",
      note: "mean habitat fit of the package's functions; packages under five functions have no model" },
    { id: "misfit", label: "misfits per function",
      value: function (t) { return t.misfits / t.functions; },
      step: function (v) { return v <= 0 ? 0 : v < 0.05 ? 1 : v < 0.1 ? 2 : v < 0.2 ? 3 : 4; },
      bins: ["0", "< 0.05", "0.05–0.10", "0.10–0.20", "≥ 0.20"],
      low: "none", high: "more functions out of place",
      note: "functions alien to their package (and, where modelled, their subsystem), divided by its functions" }
  ];
  function tintById(id) { for (var i = 0; i < TINTS.length; i++) if (TINTS[i].id === id) return TINTS[i]; return TINTS[0]; }

  /* ── Small helpers ────────────────────────────────────────────────────── */

  function hrefOf(name) { return D.atlasHref(name); }
  function label(name) { return name === "" ? "(no package)" : name; }
  /* A directory's last segment is what tells two tiles apart; the full path
     is on the tooltip and the accessible name. */
  function tileLabel(name) {
    var i = S.mode === "dir" ? name.lastIndexOf("/") : -1;
    return i < 0 ? label(name) : "…/" + name.slice(i + 1);
  }
  function rate(v) { return v.toFixed(2); }
  function numbersOf(t, mode) {
    var bits = [D.plural(t.functions, "function"),
      D.comma(t.withinMerge) + " merge-worthy of " + D.plural(t.within, "pair") + " inside",
      rate(t.withinMerge / t.functions) + " merge-worthy per function",
      D.plural(t.misfits, "misfit")];
    if (mode === "pkg") bits.push(t.norm >= 0 ? "habitat norm " + D.fixed(t.norm) : "no habitat model");
    return bits;
  }
  function select(id, labelText, options) {
    var wrap = el("div", "at-control");
    var l = el("label", "at-control-label", labelText);
    l.htmlFor = id;
    var s = el("select", "at-select");
    s.id = id;
    options.forEach(function (o) {
      var opt = el("option", null, o.label);
      opt.value = o.id;
      s.appendChild(opt);
    });
    append(wrap, l, s);
    return { wrap: wrap, select: s };
  }

  /* A pair as one row: names, the two scores as two numbers, a link. */
  function pairRow(i, n, onClick, home) {
    var e = D.edges[i], a = D.unit(e.a), b = D.unit(e.b);
    // Inside a selected package the package prefix says nothing twice.
    function nameOf(u) { return home !== undefined && u.package === home ? u.name : D.qualified(u); }
    var row = D.link(D.caseHref(e.a, e.b), "at-pair");
    row.title = D.qualified(a) + "  " + D.where(a) + "\n" + D.qualified(b) + "  " + D.where(b);
    if (onClick) row.addEventListener("click", onClick);
    var names = el("span", "at-pair-names");
    append(names, el("span", "mono", nameOf(a)), el("span", "mono", nameOf(b)));
    var meta = el("span", "at-pair-meta");
    if (e.merge) meta.appendChild(D.chip("merge-worthy", "solid"));
    if (e.kind) meta.appendChild(el("span", "muted", e.kind));
    if (meta.firstChild) names.appendChild(meta);
    var nums = el("span", "at-pair-nums mono");
    var sh = el("span", "at-n-shape", D.fixed(e.shape)); sh.title = "code-shape";
    var ov = el("span", "at-n-overlap", D.fixed(e.overlap)); ov.title = "overlap";
    append(nums, sh, ov);
    append(row, el("span", "rank-n", String(n)), names, nums);
    return row;
  }

  /* ── State ────────────────────────────────────────────────────────────── */

  var S = { built: false, mode: "pkg", tint: "merge", sel: null, missing: null,
    drawnMode: null, drawnW: 0, tiles: Object.create(null), seamShown: 10 };
  var dom = {};

  function build(host) {
    D.clear(host);
    var root = el("div", "at");
    var head = el("div", "at-head");
    dom.title = el("h2", "section-title", "Atlas");
    dom.summary = el("p", "at-summary");
    append(head, dom.title, dom.summary);

    var layout = el("div", "at-layout");
    var main = el("div", "at-main");

    var bar = el("div", "at-toolbar");
    var mode = select("at-mode", "Territories", MODES);
    var tint = select("at-tint", "Tint by", TINTS);
    var pick = select("at-pick", "Go to", []);
    dom.mode = mode.select; dom.tint = tint.select; dom.pick = pick.select;
    dom.mode.addEventListener("change", function () {
      S.mode = dom.mode.value;
      S.seamShown = 10;
      if (S.mode === "dir" && S.tint === "norm") S.tint = "merge";
      if (S.sel || S.missing) D.go("#atlas"); else update();
    });
    dom.tint.addEventListener("change", function () { S.tint = dom.tint.value; update(); });
    dom.pick.addEventListener("change", function () {
      var v = dom.pick.value;
      D.go(v === "\u0000" ? "#atlas" : hrefOf(v));
    });
    append(bar, mode.wrap, tint.wrap, pick.wrap);

    dom.legend = el("div", "at-legend");
    dom.legend.setAttribute("aria-label", "Tint legend");

    dom.map = el("div", "at-map");
    dom.map.setAttribute("role", "group");
    dom.tip = el("div", "at-tip");
    dom.tip.setAttribute("aria-hidden", "true");
    dom.tip.hidden = true;

    dom.mapNote = el("p", "at-note");

    var seamCard = el("section", "card at-seams");
    seamCard.setAttribute("aria-labelledby", "at-seams-title");
    var sh = el("div", "card-head");
    var st = el("h3", "card-title", "Seams");
    st.id = "at-seams-title";
    dom.seamSub = el("span", "card-sub");
    append(sh, st, dom.seamSub);
    dom.seamList = el("ol", "at-seam-list");
    dom.seamMore = el("button", "btn at-more", "");
    dom.seamMore.type = "button";
    dom.seamMore.addEventListener("click", function () { S.seamShown += 10; drawSeams(); });
    append(seamCard, sh, dom.seamList, dom.seamMore);

    append(main, bar, dom.legend, dom.map, dom.mapNote, seamCard);

    dom.rail = el("aside", "at-rail card");
    dom.rail.setAttribute("aria-label", "Selected territory");

    append(layout, main, dom.rail);
    append(root, head, layout);
    host.appendChild(root);

    var timer = null;
    window.addEventListener("resize", function () {
      clearTimeout(timer);
      timer = setTimeout(function () { if (!host.hidden && dom.map.clientWidth !== S.drawnW) drawMap(); }, 150);
    });
    S.built = true;
  }

  /* ── Drawing ──────────────────────────────────────────────────────────── */

  function update() {
    var m = model(S.mode), md = modeById(S.mode);
    dom.mode.value = S.mode;
    Array.prototype.forEach.call(dom.tint.options, function (o) { o.disabled = S.mode !== "pkg" && tintById(o.value).pkgOnly; });
    dom.tint.value = S.tint;
    drawSummary(m, md);
    drawPick(m, md);
    drawLegend();
    drawMap();
    drawRail(m, md);
    if (S.seamMode !== S.mode) { S.seamMode = S.mode; drawSeams(); }
    D.setCaseList(S.sel ? S.sel.withinEdges : null);
  }

  function drawSummary(m, md) {
    var all = m.withinMerge + m.crossMerge;
    var s = D.plural(m.list.length, md.noun, md.nouns) + " holding " + D.plural(D.units.length, "function") + ". ";
    s += all > 0
      ? D.comma(m.withinMerge) + " of " + D.comma(all) + " merge-worthy pairs (" + D.pct(m.withinMerge, all) +
        "%) sit inside one " + md.noun + "; " + D.comma(m.crossMerge) + " cross a " + md.noun + " line, across " +
        D.plural(m.seams.length, "seam") + "."
      : "No pair on this page is merge-worthy; " + D.plural(m.seams.length, "seam") + " carry the pairs that cross a " + md.noun + " line.";
    dom.summary.textContent = s;
  }

  function drawPick(m, md) {
    if (dom.pick.dataset.mode !== S.mode) {
      D.clear(dom.pick);
      var none = el("option", null, "— choose a " + md.noun + " —");
      none.value = "\u0000";
      dom.pick.appendChild(none);
      m.list.slice().sort(function (x, y) { return D.cmp(x.name, y.name); }).forEach(function (t) {
        var o = el("option", null, label(t.name) + " (" + t.functions + ")");
        o.value = t.name;
        dom.pick.appendChild(o);
      });
      dom.pick.dataset.mode = S.mode;
      dom.pick.previousSibling.textContent = "Go to " + md.noun;
    }
    dom.pick.value = S.sel ? S.sel.name : "\u0000";
  }

  function drawLegend() {
    var t = tintById(S.tint);
    D.clear(dom.legend);
    var title = el("span", "at-legend-title", "Tint: " + t.label);
    var ramp = el("ol", "at-ramp");
    t.bins.forEach(function (b, i) {
      var li = el("li", "at-ramp-step");
      append(li, el("span", "at-sw at-s" + i), el("span", "mono", b));
      ramp.appendChild(li);
    });
    if (t.id === "norm") {
      var li = el("li", "at-ramp-step");
      append(li, el("span", "at-sw at-snone"), el("span", null, "no model"));
      ramp.appendChild(li);
    }
    var ends = el("span", "at-legend-ends muted", "light: " + t.low + " · dark: " + t.high);
    append(dom.legend, title, ramp, ends);
  }

  function stepClass(t) {
    var tint = tintById(S.tint), s = tint.step(tint.value(t));
    return s < 0 ? "at-snone" : "at-s" + s;
  }

  function drawMap() {
    var m = model(S.mode), md = modeById(S.mode);
    var W = dom.map.clientWidth;
    if (W > 0 && (W !== S.drawnW || S.drawnMode !== S.mode)) {
      var H = Math.max(360, Math.round(W * 0.55));
      dom.map.style.height = H + "px";
      D.clear(dom.map);
      S.tiles = Object.create(null);
      var rects = squarify(m.list.map(function (t) { return t.functions; }), W, H);
      m.list.forEach(function (t, i) {
        var r = rects[i];
        if (!r) return;
        var b = el("button", "at-tile");
        b.type = "button";
        var x0 = Math.round(r.x), y0 = Math.round(r.y), x1 = Math.round(r.x + r.w), y1 = Math.round(r.y + r.h);
        var w = Math.max(0, x1 - x0 - 2), h = Math.max(0, y1 - y0 - 2);
        b.style.left = (x0 + 1) + "px"; b.style.top = (y0 + 1) + "px";
        b.style.width = w + "px"; b.style.height = h + "px";
        b.dataset.name = t.name;
        if (w >= 44 && h >= 22) {
          b.appendChild(el("span", "at-tile-name mono", tileLabel(t.name)));
          if (w >= 96 && h >= 44) {
            b.appendChild(el("span", "at-tile-n", w >= 150
              ? t.functions + " fns · " + t.withinMerge + " merge-worthy"
              : t.functions + " fns"));
          }
        }
        b.addEventListener("click", function () { D.go(S.sel === t ? "#atlas" : hrefOf(t.name)); });
        b.addEventListener("mouseenter", function () { showTip(b, t); });
        b.addEventListener("focus", function () { showTip(b, t); });
        b.addEventListener("mouseleave", hideTip);
        b.addEventListener("blur", hideTip);
        S.tiles[t.name] = { button: b, t: t };
        dom.map.appendChild(b);
      });
      dom.map.appendChild(dom.tip);
      dom.tip.hidden = true;
      S.drawnW = W; S.drawnMode = S.mode;
    }
    dom.map.setAttribute("aria-label", "Treemap of " + md.nouns + ", sized by function count, tinted by " + tintById(S.tint).label);
    Object.keys(S.tiles).forEach(function (k) {
      var tile = S.tiles[k], on = S.sel === tile.t;
      var cls = "at-tile " + stepClass(tile.t) + (on ? " at-on" : "");
      if (tile.button.className !== cls) tile.button.className = cls;
      tile.button.setAttribute("aria-label", label(k) + ": " + numbersOf(tile.t, S.mode).join(", ") + (on ? " (selected)" : ""));
      if (on) tile.button.setAttribute("aria-current", "true"); else tile.button.removeAttribute("aria-current");
    });
    var note = "Area is a " + md.noun + "'s share of the functions; tint is " + tintById(S.tint).note + ". " +
      "Small territories carry no label — hover or focus one for its numbers, or use “Go to”. " +
      "Pairs that cross a " + md.noun + " line are listed under Seams rather than drawn on the map.";
    if (S.mode === "dir") {
      note += " Functions are grouped by the directory their file lives in, so a package name " +
        "used in several directories (every main) splits into one territory per directory. Habitat norms are modelled per package, so that tint is unavailable here.";
    }
    dom.mapNote.textContent = note;
  }

  function showTip(b, t) {
    D.clear(dom.tip);
    dom.tip.appendChild(el("strong", "mono", label(t.name)));
    dom.tip.appendChild(el("span", null, numbersOf(t, S.mode).join(" · ")));
    dom.tip.hidden = false;
    var W = dom.map.clientWidth, H = dom.map.clientHeight;
    var bx = b.offsetLeft, by = b.offsetTop, bh = b.offsetHeight;
    var tw = dom.tip.offsetWidth, th = dom.tip.offsetHeight;
    var x = Math.max(4, Math.min(W - tw - 4, bx));
    var y = by + bh + 6;
    if (y + th > H) y = Math.max(4, by - th - 6);
    if (by - th - 6 < 0 && by + bh + 6 + th > H) y = Math.max(4, by + 4); // no room either side: overlay
    dom.tip.style.left = x + "px";
    dom.tip.style.top = y + "px";
  }
  function hideTip() { dom.tip.hidden = true; }

  function stats(t) {
    var g = el("dl", "at-stats");
    function cell(k, v) { var d = el("div", "at-stat"); append(d, el("dt", null, k), el("dd", "mono", v)); g.appendChild(d); }
    cell("merge-worthy / fn", rate(t.withinMerge / t.functions));
    cell("pairs inside", D.comma(t.within));
    cell("crossing pairs", D.comma(t.cross));
    cell("misfits", D.comma(t.misfits));
    cell("families touching", D.comma(t.families));
    if (S.mode === "pkg") cell("habitat norm", t.norm >= 0 ? D.fixed(t.norm) : "no model");
    return g;
  }

  function drawRail(m, md) {
    var r = D.clear(dom.rail);
    if (S.missing !== null) {
      append(r, el("div", "at-kicker", "Not found"),
        el("p", null, "No package or directory named “" + S.missing + "” in this run."),
        D.link("#atlas", "btn", "Clear selection"));
      return;
    }
    var t = S.sel;
    if (!t) { drawRailEmpty(r, m, md); return; }

    var head = el("div", "at-rail-head");
    append(head, el("div", "at-kicker", "Selected " + md.noun),
      el("h3", "at-rail-name mono", label(t.name)),
      el("div", "at-rail-sub", D.plural(t.functions, "function") + " · " + D.comma(t.withinMerge) + " merge-worthy " +
        (t.withinMerge === 1 ? "pair" : "pairs") + " inside"));
    r.appendChild(head);
    r.appendChild(stats(t));

    // Ranked here
    var ranked = el("section", "at-block");
    var rh = el("div", "at-block-head");
    append(rh, el("h4", "at-kicker", "Ranked here"),
      el("span", "muted at-small", t.within ? "by rank, of " + D.plural(t.within, "pair") + " inside" : ""));
    ranked.appendChild(rh);
    if (!t.within) {
      ranked.appendChild(el("p", "muted at-small", "No pair on this page has both functions in this " + md.noun + "."));
    } else {
      ranked.appendChild(D.scoreKey());
      var list = el("div", "at-pairs");
      var shown = 0;
      var more = el("button", "btn at-more");
      more.type = "button";
      var showMore = function () {
        var to = Math.min(t.withinEdges.length, shown + 8);
        for (; shown < to; shown++) list.appendChild(pairRow(t.withinEdges[shown], shown + 1, null, S.mode === "pkg" ? t.name : undefined));
        var left = t.withinEdges.length - shown;
        more.hidden = left <= 0;
        more.textContent = "Show " + Math.min(8, left) + " more (" + D.comma(left) + " left)";
      };
      more.addEventListener("click", showMore);
      showMore();
      append(ranked, list, more);
    }
    r.appendChild(ranked);

    // Its seams
    var sb = el("section", "at-block");
    var sbh = el("div", "at-block-head");
    append(sbh, el("h4", "at-kicker", "Its seams"),
      el("span", "muted at-small", t.seams.length ? "merge-worthy / all pairs crossing" : ""));
    sb.appendChild(sbh);
    if (!t.seams.length) {
      sb.appendChild(el("p", "muted at-small", "No pair crosses out of this " + md.noun + "."));
    } else {
      var ul = el("ul", "at-rail-seams");
      var sShown = 0;
      var sMore = el("button", "btn at-more");
      sMore.type = "button";
      var showSeams = function () {
        var to = Math.min(t.seams.length, sShown + 8);
        for (; sShown < to; sShown++) {
          var s = t.seams[sShown], other = s.a === t.name ? s.b : s.a;
          var li = el("li");
          var b = el("button", "at-seam-btn");
          b.type = "button";
          b.setAttribute("aria-label", "Select " + label(other) + ": " + s.merge + " merge-worthy of " + D.plural(s.pairs, "pair") + " crossing");
          append(b, el("span", "mono at-seam-other", label(other)), el("span", "mono at-seam-n", s.merge + " / " + s.pairs));
          (function (o) { b.addEventListener("click", function () { D.go(hrefOf(o)); }); })(other);
          li.appendChild(b);
          ul.appendChild(li);
        }
        var left = t.seams.length - sShown;
        sMore.hidden = left <= 0;
        sMore.textContent = "Show " + Math.min(8, left) + " more (" + D.comma(left) + " left)";
      };
      sMore.addEventListener("click", showSeams);
      showSeams();
      append(sb, ul, sMore);
    }
    r.appendChild(sb);

    var actions = el("div", "at-actions");
    if (S.mode === "pkg") actions.appendChild(D.link(D.triagePkgHref(t.name), "btn btn-primary", "Triage this package →"));
    actions.appendChild(D.link("#atlas", "btn", "Clear selection"));
    r.appendChild(actions);
  }

  function drawRailEmpty(r, m, md) {
    append(r, el("div", "at-kicker", "Reading the atlas"),
      el("p", "at-rail-p", "Each tile is a " + md.noun + ", sized by how many functions it holds and tinted by the measure chosen above. " +
        "Select a tile (click, or Tab to it and press Enter) to see its numbers, the pairs ranked inside it, and the seams " +
        "that join it to other " + md.nouns + ". Select it again to clear."),
      el("p", "at-rail-p muted", "Nothing is selected yet."));
    if (!m.list.length) return;
    var most = m.list.slice().sort(function (x, y) {
      return (y.withinMerge - x.withinMerge) || (y.functions - x.functions) || D.cmp(x.name, y.name);
    })[0];
    var largest = m.list[0];
    var hints = el("div", "at-hints");
    hints.appendChild(el("div", "at-kicker", "Places to start"));
    function hint(t, why) {
      var b = el("button", "btn at-hint");
      b.type = "button";
      append(b, el("span", "mono", label(t.name)), el("span", "muted", why));
      b.addEventListener("click", function () { D.go(hrefOf(t.name)); });
      hints.appendChild(b);
    }
    if (most.withinMerge > 0) hint(most, "most merge-worthy pairs inside (" + D.comma(most.withinMerge) + ")");
    if (largest !== most) hint(largest, "largest (" + D.plural(largest.functions, "function") + ")");
    if (m.seams.length) {
      var s = m.seams[0];
      var p = el("p", "at-rail-p muted at-small", "Heaviest seam: " + label(s.a) + " ↔ " + label(s.b) + ", " +
        s.merge + " merge-worthy of " + D.plural(s.pairs, "pair") + ".");
      hints.appendChild(p);
    }
    r.appendChild(hints);
  }

  function drawSeams() {
    var m = model(S.mode), md = modeById(S.mode);
    D.clear(dom.seamList);
    var n = Math.min(S.seamShown, m.seams.length);
    dom.seamSub.textContent = m.seams.length
      ? "showing " + n + " of " + D.plural(m.seams.length, "seam") + " between " + md.nouns +
        ", by merge-worthy pairs crossing, then all pairs; the bar is " +
        (m.seams[0].merge > 0 ? "merge-worthy pairs" : "all pairs") + " against the top seam's"
      : "no pair crosses a " + md.noun + " line";
    var top = m.seams.length ? Math.max(m.seams[0].merge, 1) : 1;
    var byMerge = m.seams.length && m.seams[0].merge > 0;
    var maxPairs = 1;
    m.seams.forEach(function (s) { if (s.pairs > maxPairs) maxPairs = s.pairs; });
    for (var i = 0; i < n; i++) dom.seamList.appendChild(seamRow(m.seams[i], i, byMerge ? top : maxPairs, byMerge));
    var left = m.seams.length - n;
    dom.seamMore.hidden = left <= 0;
    dom.seamMore.textContent = "Show " + Math.min(10, left) + " more seams (" + D.comma(left) + " left)";
  }

  function seamRow(s, i, scale, byMerge) {
    var li = el("li", "at-seam");
    var b = el("button", "at-seam-row");
    b.type = "button";
    b.setAttribute("aria-expanded", "false");
    var names = el("span", "at-seam-names mono");
    append(names, label(s.a), el("span", "muted", " ↔ "), label(s.b));
    var v = (byMerge ? s.merge : s.pairs) / scale;
    var bar = D.bar(v, "neutral", "");
    bar.classList.add("at-seam-bar");
    append(b, el("span", "rank-n", String(i + 1)), names, bar,
      el("span", "at-seam-count mono", s.merge + " / " + D.plural(s.pairs, "pair")));
    b.setAttribute("aria-label", label(s.a) + " and " + label(s.b) + ": " + s.merge + " merge-worthy of " + D.plural(s.pairs, "pair") + " crossing");
    var panel = el("div", "at-seam-panel");
    panel.hidden = true;
    b.addEventListener("click", function () {
      var open = panel.hidden;
      panel.hidden = !open;
      b.setAttribute("aria-expanded", open ? "true" : "false");
      if (open && !panel.firstChild) fillSeam(panel, s);
    });
    append(li, b, panel);
    return li;
  }

  function fillSeam(panel, s) {
    var walk = function () { D.setCaseList(s.edges); };
    var list = el("div", "at-pairs");
    s.edges.slice(0, 5).forEach(function (i, k) { list.appendChild(pairRow(i, k + 1, walk)); });
    panel.appendChild(list);
    var foot = el("div", "at-seam-foot");
    foot.appendChild(el("span", "muted at-small", s.edges.length > 5
      ? "top 5 by rank of " + D.plural(s.edges.length, "pair") + " on this seam; the case file steps through all of them"
      : "all " + D.plural(s.edges.length, "pair") + " on this seam, by rank"));
    [s.a, s.b].forEach(function (name) {
      var b = el("button", "btn", "Select " + label(name));
      b.type = "button";
      b.addEventListener("click", function () { D.go(hrefOf(name)); });
      foot.appendChild(b);
    });
    panel.appendChild(foot);
  }

  /* ── Routing ──────────────────────────────────────────────────────────── */

  function resolve(params) {
    S.sel = null; S.missing = null;
    if (!params.length || (params.length === 1 && params[0] === "")) return;
    var name = params.join("/");
    var m = model(S.mode);
    if (m.by[name]) { S.sel = m.by[name]; return; }
    var other = S.mode === "pkg" ? "dir" : "pkg";
    var mo = model(other);
    if (mo.by[name]) {
      S.mode = other;
      if (S.mode === "dir" && S.tint === "norm") S.tint = "merge";
      S.sel = mo.by[name];
      return;
    }
    S.missing = name;
  }

  D.screens["atlas"] = {
    render: function (host, params) {
      if (!S.built || !host.firstChild) build(host);
      resolve(params || []);
      update();
    }
  };
})(DOPPEL);
