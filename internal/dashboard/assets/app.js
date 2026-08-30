/* doppel dashboard.
 *
 * A classic script: it is inlined into a file:// page and module semantics buy
 * nothing here. There is no library — the map is a power diagram drawn as
 * plain SVG, which is the geometry below plus a few hundred lines of render.
 *
 * The contract with Go is that this file makes the presentation decisions and
 * Go makes none. Everything arriving in the payload is a raw score, count or
 * identifier; every percentage, colour, area and label is decided here. The one
 * deliberate exception is Edge.rank — analyzer.RankKey is the single definition
 * of corroborated evidence in this repo, and reimplementing it here would let
 * the two drift apart silently.
 */
(function () {
  "use strict";

  var DATA = JSON.parse(document.getElementById("doppel-data").textContent);

  /* The LCS diff is O(n·m). Bodies longer than this are shown undiffed. */
  var DIFF_MAX_LINES = 600;

  var $ = function (id) { return document.getElementById(id); };
  var SVGNS = "http://www.w3.org/2000/svg";

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined && text !== null) n.textContent = String(text);
    return n;
  }
  function svg(tag, attrs) {
    var n = document.createElementNS(SVGNS, tag);
    if (attrs) Object.keys(attrs).forEach(function (k) { n.setAttribute(k, attrs[k]); });
    return n;
  }
  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); }
  function fixed(v, n) { return (v === undefined || v === null) ? "—" : Number(v).toFixed(n === undefined ? 2 : n); }
  function comma(n) { return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ","); }
  function pct(n, d) { return d > 0 ? Math.round((100 * n) / d) : 0; }
  function plural(n, one, many) { return n + " " + (n === 1 ? one : (many || one + "s")); }

  /* ── Indexes ─────────────────────────────────────────────────────────── */

  var units = DATA.units || [];
  var edges = DATA.edges || [];
  var facts = DATA.facts || {};

  var bodyByUnit = Object.create(null);
  (DATA.bodies || []).forEach(function (b) { bodyByUnit[b.unit] = b.text; });

  /* Edges are delivered rank-descending, so every per-unit list inherits that
     order without a second sort. */
  var edgesByUnit = Object.create(null);
  edges.forEach(function (e, i) {
    (edgesByUnit[e.a] || (edgesByUnit[e.a] = [])).push(i);
    (edgesByUnit[e.b] || (edgesByUnit[e.b] = [])).push(i);
  });

  var conceptRows = DATA.concepts || [];

  /* The vocabulary is learned, so its size is a property of the corpus rather
     than a constant this page can plan for. Cycling the palette would give two
     unrelated concepts the same hue and say nothing about it, so instead the
     concepts that colour the most functions take the palette and the rest share
     one neutral, counted honestly in the legend.
     
     Payload order is (dominant desc, carried desc, id asc), so a concept keeps
     its colour across runs of an unchanged tree. */
  var PALETTE = [
    "#0088b0", "#d6006c", "#edbb00", "#1186ac", "#ff458e", "#006786",
    "#7d7979", "#38a6cf", "#aa0b56", "#62c5ee", "#790e3d", "#004961",
    "#0a303e"
  ];
  var OTHER = "#bab6b6";
  var conceptColor = Object.create(null);
  conceptRows.forEach(function (row, i) {
    if (i < PALETTE.length && row.dominant > 0) conceptColor[row.id] = PALETTE[i];
  });
  function colorFor(id) { return (id && conceptColor[id]) || OTHER; }

  /* ── Facts header ────────────────────────────────────────────────────── */

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
    host.appendChild(fact(comma(facts.functions || 0), "functions",
      comma(facts.packages || 0) + " packages · tests " + (facts.testsMode || "excluded")));
    host.appendChild(fact(comma(facts.pairs || 0), "scored pairs",
      "of " + comma(facts.candidatePairs || 0) + " retrieved"));
    host.appendChild(fact(comma(facts.familyCount || 0), "families",
      comma(facts.familyFuncs || 0) + " functions · largest " + (facts.familyLargest || 0)));
    host.appendChild(fact(comma(facts.misfits || 0), "misfits",
      comma(facts.misfitsExcused || 0) + " excused by subsystem"));
    host.appendChild(fact(
      pct(facts.onlyCall || 0, facts.candidatePairs || 0) + "%", "call-only",
      pct(facts.onlyConcept || 0, facts.candidatePairs || 0) + "% concept-only"));
    host.appendChild(fact(fixed(facts.threshold), "threshold",
      "struct-min " + fixed(facts.structMin) + (facts.calibrate > 0 ? " · calibrated" : "")));
    /* How repetitive the corpus is at all, and how close a typical function
       already sits to its nearest scored neighbour. Neither number moved a
       pair; they say what the pair list was drawn out of. The percentile is
       over the functions retrieval actually paired, which is why the label
       says so rather than implying every function was measured. */
    host.appendChild(fact(fixed(facts.compression, 2) + "×", "compression",
      "canonical nodes per distinct subtree"));
    host.appendChild(fact(fixed(facts.nnP50), "median nearest",
      "p90 " + fixed(facts.nnP90) + " · over " + comma(facts.nnScored || 0) + " paired"));
  }

  function renderColophon() {
    var host = $("colophon");
    var bits = [];
    bits.push("This page draws the full comparator-scored, struct-min-filtered pair set — " +
      comma(facts.pairs || 0) + " pairs. The text report showed " + comma(facts.reported || 0) +
      " after --top and --max-per-func" +
      (facts.suppressed ? ", holding back " + comma(facts.suppressed) + " to the per-function cap" : "") +
      "; that cap is a presentation device, so it does not bound this page.");
    bits.push("The map is a cartogram: region area is a share of the function count, not a measured " +
      "size, and regions are packed so related ones tend to adjoin. Adjacency is therefore a " +
      "tendency and not a claim — a planar map cannot realise every adjacency a corpus asks for, " +
      "which is what the arcs are for.");
    if (facts.bodiesOmitted) {
      bits.push("Source is inlined for the highest-ranked pairs only: " + comma(facts.bodiesOmitted) +
        " function bodies were left out to keep the file a sane size.");
    }
    if (!facts.debug) {
      bits.push("Shared structure lists the top 3 patterns per pair; run with --debug for 20.");
    }
    bits.push("A region is tinted by the concept most of its functions carry. Habitat fit, roles " +
      "and concept typicality are all corpus-relative — they move when unrelated code moves.");
    clear(host);
    host.appendChild(el("p", null, bits.join(" ")));
  }

  /* ── Tabs ────────────────────────────────────────────────────────────── */

  var screens = { map: $("screen-map"), neighbourhood: $("screen-neighbourhood") };
  var current = "map";

  function show(name) {
    if (!screens[name]) name = "map";
    var was = current;
    current = name;
    Object.keys(screens).forEach(function (k) { screens[k].hidden = k !== name; });
    Array.prototype.forEach.call($("tabs").children, function (b) {
      b.setAttribute("aria-selected", b.dataset.screen === name ? "true" : "false");
    });
    /* The map measures its own container, which has no size while hidden. */
    if (name === "map" && was !== "map") drawMap();
  }
  Array.prototype.forEach.call($("tabs").children, function (b) {
    b.addEventListener("click", function () { show(b.dataset.screen); writeHash(); });
  });

  /* ── Geometry: polygons and the power diagram ────────────────────────── */

  function area(poly) {
    var a = 0;
    for (var i = 0, n = poly.length; i < n; i++) {
      var p = poly[i], q = poly[(i + 1) % n];
      a += p[0] * q[1] - q[0] * p[1];
    }
    return a / 2;
  }

  function centroid(poly) {
    var a = 0, cx = 0, cy = 0;
    for (var i = 0, n = poly.length; i < n; i++) {
      var p = poly[i], q = poly[(i + 1) % n];
      var f = p[0] * q[1] - q[0] * p[1];
      a += f; cx += (p[0] + q[0]) * f; cy += (p[1] + q[1]) * f;
    }
    if (Math.abs(a) < 1e-9) return poly.length ? poly[0].slice() : [0, 0];
    a *= 3;
    return [cx / a, cy / a];
  }

  /* Clip a tagged polygon by the half-plane a·x + b·y <= c.
   *
   * Tagged: each edge carries the index of the site that produced it, so the
   * finished cell knows which neighbour every one of its borders faces. That is
   * what lets the map paint a border with the duplication that crosses it —
   * recovering adjacency afterwards by comparing coordinates would be both
   * slower and fuzzier than recording it at the moment it is created.
   *
   * Sutherland–Hodgman, with the tag rule: an edge kept whole keeps its tag, an
   * edge entering the half-plane keeps its tag from the crossing point, and the
   * edge that runs along the clip line itself is tagged with the clipper.
   */
  function clipHalfPlane(verts, tags, a, b, c, clipper) {
    var outV = [], outT = [], n = verts.length;
    if (!n) return { verts: outV, tags: outT };
    var dist = new Array(n);
    for (var i = 0; i < n; i++) dist[i] = a * verts[i][0] + b * verts[i][1] - c;

    for (i = 0; i < n; i++) {
      var j = (i + 1) % n;
      var din = dist[i] <= 0, djn = dist[j] <= 0;
      if (din) { outV.push(verts[i]); outT.push(tags[i]); }
      if (din !== djn) {
        var t = dist[i] / (dist[i] - dist[j]);
        var x = [verts[i][0] + t * (verts[j][0] - verts[i][0]),
                 verts[i][1] + t * (verts[j][1] - verts[i][1])];
        outV.push(x);
        outT.push(din ? clipper : tags[i]);
      }
    }
    return { verts: outV, tags: outT };
  }

  /* The power (Laguerre) diagram of weighted sites, clipped to a rectangle.
   *
   * A site's cell is the set of points where |x−pᵢ|² − wᵢ is least. The
   * bisector between two sites is still a straight line — only shifted by the
   * weight difference — so a cell is still convex and still falls out of
   * successive half-plane clips. That is the whole reason weighted Voronoi is
   * usable here: the area targeting below needs weights, and weights cost
   * nothing but a constant in the clip.
   *
   * O(n²) clips. n is a package count — 18 on this repo, ~170 on moby — so the
   * quadratic is irrelevant and an incremental Delaunay would be ceremony.
   */
  function powerDiagram(sites, weights, bounds) {
    var x0 = bounds[0], y0 = bounds[1], x1 = bounds[2], y1 = bounds[3];
    var cells = [];
    for (var i = 0; i < sites.length; i++) {
      var verts = [[x0, y0], [x1, y0], [x1, y1], [x0, y1]];
      var tags = [-1, -1, -1, -1]; // -1 is the frame, not a neighbour
      var pi = sites[i], wi = weights[i];
      for (var j = 0; j < sites.length && verts.length; j++) {
        if (j === i) continue;
        var pj = sites[j], wj = weights[j];
        var a = 2 * (pj[0] - pi[0]);
        var b = 2 * (pj[1] - pi[1]);
        var c = (pj[0] * pj[0] + pj[1] * pj[1] - wj) - (pi[0] * pi[0] + pi[1] * pi[1] - wi);
        if (a === 0 && b === 0) continue;
        var r = clipHalfPlane(verts, tags, a, b, c, j);
        verts = r.verts; tags = r.tags;
      }
      cells.push({ poly: verts, tags: tags, area: verts.length ? Math.abs(area(verts)) : 0 });
    }
    return cells;
  }

  /* The distance from site i to its nearest other site. */
  function nearestGap(sites, i) {
    var best = Infinity;
    for (var j = 0; j < sites.length; j++) {
      if (i === j) continue;
      var dx = sites[i][0] - sites[j][0], dy = sites[i][1] - sites[j][1];
      var d = Math.sqrt(dx * dx + dy * dy);
      if (d < best) best = d;
    }
    return best === Infinity ? 1e6 : best;
  }

  /* Fit the cells to target area shares — this is what makes the map a
   * cartogram rather than a diagram.
   *
   * Two alternating steps, after Nocaj & Brandes. Positions move toward their
   * cell's centroid, which keeps the tiling regular instead of letting it shear
   * into slivers. Weights then trade area between neighbours until each cell
   * holds its share of the functions.
   *
   * Both steps are expressed in *radii*, and that is the whole trick. A weight
   * is a squared radius, so the weight step is multiplicative — a cell 4× too
   * small has its radius scaled by a damped power of the ratio — and both steps
   * clamp against the distance to the nearest neighbouring site: a move never
   * jumps past a neighbour, and a radius never exceeds the gap to one. Written
   * additively on raw area instead, with the clamp expressed as
   * wᵢ − wⱼ ≤ |pᵢ − pⱼ|², it does not converge: that constraint binds hardest
   * exactly where two sites are close, so a large region wedged beside a small
   * one cannot grow. Measured on this repo, that version left `parser` at 0.02%
   * of the canvas against a 4.9% target while a one-function package took 5×
   * its share. This one lands the weighted mean area error at 0.2%.
   *
   * The residual error is concentrated in regions too small to draw honestly
   * anyway — a one-function package is 0.2% of the map, and being 30% wrong
   * about it is a few pixels.
   *
   * Deterministic by construction: fixed iteration count, fixed order, no
   * randomness anywhere, so the same payload always draws the same map.
   */
  function fitAreas(sites, shares, bounds, iterations) {
    var n = sites.length;
    var total = (bounds[2] - bounds[0]) * (bounds[3] - bounds[1]);
    // Start each site holding a circle of its target area.
    var weights = shares.map(function (s) {
      var r = Math.sqrt((total * s) / Math.PI);
      return r * r;
    });
    var cells, i;

    for (var it = 0; it < iterations; it++) {
      cells = powerDiagram(sites, weights, bounds);
      for (i = 0; i < n; i++) {
        if (cells[i].area <= 0) continue;
        var c = centroid(cells[i].poly);
        var dx = c[0] - sites[i][0], dy = c[1] - sites[i][1];
        var d = Math.sqrt(dx * dx + dy * dy), cap = 0.9 * nearestGap(sites, i);
        if (d > cap && d > 0) { dx *= cap / d; dy *= cap / d; }
        sites[i][0] += dx; sites[i][1] += dy;
      }

      cells = powerDiagram(sites, weights, bounds);
      for (i = 0; i < n; i++) {
        var target = total * shares[i];
        var got = cells[i].area > 0 ? cells[i].area : total * 1e-6;
        var r = Math.sqrt(Math.max(weights[i], 1e-9)) * Math.pow(target / got, 0.15);
        var gap = nearestGap(sites, i);
        if (r > gap) r = gap; // a site must stay inside its own cell
        weights[i] = r * r;
      }
    }
    return { cells: powerDiagram(sites, weights, bounds), sites: sites, weights: weights };
  }

  /* Seed the sites so related regions start near each other.
   *
   * A plain spring embedding of the region coupling graph: coupled regions
   * attract, everything repels, run to a fixed iteration count. It decides
   * which regions end up adjoining, because the area fitting above moves sites
   * only locally afterwards.
   *
   * Started on a circle in size order rather than at random — this is the one
   * place a layout would normally reach for a seeded RNG, and a deterministic
   * ring costs nothing and removes the question.
   */
  function seedSites(n, links, bounds) {
    var pos = new Array(n);
    for (var i = 0; i < n; i++) {
      var t = (2 * Math.PI * i) / n;
      pos[i] = [Math.cos(t), Math.sin(t)];
    }
    if (n < 3) {
      return pos.map(function (p) {
        return [(bounds[0] + bounds[2]) / 2 + p[0] * 30, (bounds[1] + bounds[3]) / 2 + p[1] * 30];
      });
    }

    var maxW = 0;
    links.forEach(function (l) { if (l.mass > maxW) maxW = l.mass; });

    for (var iter = 0; iter < 260; iter++) {
      var fx = new Array(n).fill(0), fy = new Array(n).fill(0);
      for (i = 0; i < n; i++) {
        for (var j = i + 1; j < n; j++) {
          var dx = pos[j][0] - pos[i][0], dy = pos[j][1] - pos[i][1];
          var d = Math.sqrt(dx * dx + dy * dy) || 1e-6;
          var rep = 0.06 / (d * d);
          fx[i] -= (dx / d) * rep; fy[i] -= (dy / d) * rep;
          fx[j] += (dx / d) * rep; fy[j] += (dy / d) * rep;
        }
      }
      links.forEach(function (l) {
        var dx = pos[l.b][0] - pos[l.a][0], dy = pos[l.b][1] - pos[l.a][1];
        var d = Math.sqrt(dx * dx + dy * dy) || 1e-6;
        var att = 0.9 * (maxW ? l.mass / maxW : 0) * d;
        fx[l.a] += (dx / d) * att; fy[l.a] += (dy / d) * att;
        fx[l.b] -= (dx / d) * att; fy[l.b] -= (dy / d) * att;
      });
      var step = 0.08 * (1 - iter / 260);
      for (i = 0; i < n; i++) { pos[i][0] += fx[i] * step; pos[i][1] += fy[i] * step; }
    }

    // Normalise into the frame, inset so no site starts on an edge.
    var minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    pos.forEach(function (p) {
      if (p[0] < minX) minX = p[0]; if (p[0] > maxX) maxX = p[0];
      if (p[1] < minY) minY = p[1]; if (p[1] > maxY) maxY = p[1];
    });
    var spanX = (maxX - minX) || 1, spanY = (maxY - minY) || 1;
    var padX = (bounds[2] - bounds[0]) * 0.08, padY = (bounds[3] - bounds[1]) * 0.08;
    return pos.map(function (p) {
      return [
        bounds[0] + padX + ((p[0] - minX) / spanX) * (bounds[2] - bounds[0] - 2 * padX),
        bounds[1] + padY + ((p[1] - minY) / spanY) * (bounds[3] - bounds[1] - 2 * padY)
      ];
    });
  }

  /* ── The region model ────────────────────────────────────────────────── */

  var partitionEl = $("partition"), metricEl = $("metric"), cutEl = $("cut"), cutLabel = $("cut-label");
  var mergeOnly = $("merge-only"), showArcs = $("show-arcs");

  function regionKeyOf(u) {
    if (partitionEl.value === "concept") return u.concept || "untagged";
    return u.package || "(none)";
  }

  function metricOf(e) {
    var m = metricEl.value;
    return m === "overlap" ? e.overlap : (m === "rank" ? e.rank : e.shape);
  }

  function metricMax() {
    if (metricEl.value !== "rank") return 1;
    var max = 0;
    edges.forEach(function (e) { if (e.rank > max) max = e.rank; });
    return max > 0 ? max : 1;
  }

  function syncCutRange() {
    var max = metricMax();
    cutEl.max = String(max);
    cutEl.step = String(max / 100);
    var start = metricEl.value === "shape" ? Math.min(facts.threshold || 0, max) : 0;
    cutEl.value = String(start);
  }

  /* Regions, their members, and the duplication that runs between them.
   *
   * Ordered by size then name: the seed ring is walked in this order, so a
   * stable order here is what makes the map itself stable. */
  function buildRegions() {
    var byKey = Object.create(null);
    units.forEach(function (u) {
      var k = regionKeyOf(u);
      var r = byKey[k] || (byKey[k] = { key: k, members: [], concepts: Object.create(null), misfits: 0 });
      r.members.push(u.id);
      r.concepts[u.concept || ""] = (r.concepts[u.concept || ""] || 0) + 1;
      if (u.misfit) r.misfits++;
    });

    var regions = Object.keys(byKey).sort(function (x, y) {
      return byKey[y].members.length - byKey[x].members.length || (x < y ? -1 : x > y ? 1 : 0);
    }).map(function (k) { return byKey[k]; });

    var indexOf = Object.create(null);
    regions.forEach(function (r, i) {
      indexOf[r.key] = i;
      r.index = i;
      r.internal = { pairs: 0, merge: 0, mass: 0 };
      // The region's own colour is the concept most of its members carry.
      var best = "", bestN = 0;
      Object.keys(r.concepts).sort().forEach(function (c) {
        if (c && r.concepts[c] > bestN) { best = c; bestN = r.concepts[c]; }
      });
      r.concept = best;
    });

    var cut = parseFloat(cutEl.value) || 0;
    var linkMap = Object.create(null);
    var links = [];
    var shown = 0;
    edges.forEach(function (e, ei) {
      if (metricOf(e) < cut) return;
      if (mergeOnly.checked && !e.merge) return;
      shown++;
      var a = indexOf[regionKeyOf(units[e.a])], b = indexOf[regionKeyOf(units[e.b])];
      if (a === b) {
        var r = regions[a];
        r.internal.pairs++; r.internal.mass += e.total;
        if (e.merge) r.internal.merge++;
        return;
      }
      if (a > b) { var t = a; a = b; b = t; }
      var key = a + ":" + b;
      var l = linkMap[key];
      if (!l) { l = linkMap[key] = { a: a, b: b, pairs: 0, merge: 0, mass: 0, edges: [] }; links.push(l); }
      l.pairs++; l.mass += e.total; l.edges.push(ei);
      if (e.merge) l.merge++;
    });
    links.sort(function (x, y) { return y.mass - x.mass || x.a - y.a || x.b - y.b; });
    return { regions: regions, links: links, linkMap: linkMap, indexOf: indexOf, shownEdges: shown };
  }

  /* ── Map rendering ───────────────────────────────────────────────────── */

  var BOUNDS = [0, 0, 1000, 640];
  var mapState = null;
  var selection = null; // {kind:"region"|"border", ...}

  function drawMap() {
    var host = $("map");
    var model = buildRegions();
    var regions = model.regions;

    $("edge-count").textContent = comma(model.shownEdges) + " pairs · " +
      plural(regions.length, "region") + " · " + plural(model.links.length, "link");

    var empty = $("map-empty");
    if (!regions.length) {
      empty.hidden = false;
      empty.textContent = "Nothing to draw.";
      clear(host);
      return;
    }
    empty.hidden = true;

    var totalMembers = 0;
    regions.forEach(function (r) { totalMembers += r.members.length; });
    var shares = regions.map(function (r) { return r.members.length / totalMembers; });

    var sites = seedSites(regions.length, model.links, BOUNDS);
    var fit = fitAreas(sites, shares, BOUNDS, regions.length > 70 ? 160 : 260);
    mapState = { model: model, cells: fit.cells, sites: fit.sites };

    clear(host);
    host.setAttribute("viewBox", BOUNDS.join(" "));
    host.setAttribute("preserveAspectRatio", "xMidYMid meet");

    var view = svg("g", { id: "map-view" });
    var gFill = svg("g"), gBorder = svg("g"), gArc = svg("g"), gLabel = svg("g");
    view.appendChild(gFill); view.appendChild(gBorder); view.appendChild(gArc);
    view.appendChild(gLabel);
    host.appendChild(view);

    drawTerritories(gFill, gLabel, model, fit.cells);
    drawBorders(gBorder, model, fit.cells);
    if (showArcs.checked) drawArcs(gArc, model, fit.cells);

    attachPanZoom(host, view);
    renderSelection();
  }

  function pointsOf(poly) {
    return poly.map(function (p) { return p[0].toFixed(2) + "," + p[1].toFixed(2); }).join(" ");
  }

  function drawTerritories(gFill, gLabel, model, cells) {
    model.regions.forEach(function (r, i) {
      var cell = cells[i];
      if (!cell.poly.length) return;
      var tint = colorFor(r.concept);
      var poly = svg("polygon", {
        points: pointsOf(cell.poly),
        fill: tint,
        "fill-opacity": r.internal.merge > 0 ? 0.20 : 0.10,
        stroke: "none",
        class: "region"
      });
      poly.addEventListener("click", function (ev) { ev.stopPropagation(); selectRegion(i); });
      var title = svg("title");
      title.textContent = r.key + " — " + plural(r.members.length, "function") +
        (r.internal.merge ? " · " + plural(r.internal.merge, "merge-worthy pair") + " inside" : "");
      poly.appendChild(title);
      gFill.appendChild(poly);

      // A label only where its own region can hold it.
      var c = centroid(cell.poly);
      var size = Math.max(8, Math.min(20, Math.sqrt(cell.area) / 7));
      if (cell.area > 900) {
        var label = svg("text", {
          x: c[0].toFixed(1), y: c[1].toFixed(1), "text-anchor": "middle",
          class: "region-label", "font-size": size.toFixed(1)
        });
        label.textContent = r.key;
        gLabel.appendChild(label);
        if (cell.area > 4200) {
          var sub = svg("text", {
            x: c[0].toFixed(1), y: (c[1] + size * 1.05).toFixed(1), "text-anchor": "middle",
            class: "region-sub", "font-size": (size * 0.62).toFixed(1)
          });
          sub.textContent = r.members.length;
          gLabel.appendChild(sub);
        }
      }
    });
  }

  /* Every border is drawn. The ones two related regions share are painted by
     how much duplication crosses them; the rest are hairlines. That split is
     the map's whole claim: the border exists because of the packing, and its
     weight is the only part of it that is evidence. */
  function drawBorders(gBorder, model, cells) {
    var maxMass = 0;
    model.links.forEach(function (l) { if (l.mass > maxMass) maxMass = l.mass; });

    model.regions.forEach(function (r, i) {
      var cell = cells[i];
      for (var k = 0; k < cell.poly.length; k++) {
        var j = cell.tags[k];
        if (j <= i) continue; // frame edges are -1; each shared edge drawn once
        var p = cell.poly[k], q = cell.poly[(k + 1) % cell.poly.length];
        var dx = q[0] - p[0], dy = q[1] - p[1];
        if (dx * dx + dy * dy < 4) continue; // a corner touch is not a border

        var l = model.linkMap[i + ":" + j];
        var hot = l && l.mass > 0;
        var line = svg("line", {
          x1: p[0].toFixed(2), y1: p[1].toFixed(2), x2: q[0].toFixed(2), y2: q[1].toFixed(2),
          class: hot ? (l.merge ? "border hot merge" : "border hot") : "border",
          "stroke-width": hot ? (1.2 + 5 * Math.sqrt(l.mass / maxMass)).toFixed(2) : "0.8"
        });
        if (hot) {
          var t = svg("title");
          t.textContent = model.regions[i].key + " ↔ " + model.regions[j].key + " — " +
            plural(l.pairs, "pair") + ", " + l.merge + " merge-worthy, evidence " + fixed(l.mass, 0);
          line.appendChild(t);
          line.addEventListener("click", function (ev) { ev.stopPropagation(); selectBorder(l); });
        }
        gBorder.appendChild(line);
      }
    });
  }

  /* Related regions the packing could not seat next to each other.
   *
   * A planar map cannot realise every adjacency a corpus asks for, so these
   * are the relationships that would otherwise silently disappear into the
   * layout. Drawn as arcs, and counted in the selection panel, so the map is
   * never quietly lossy. */
  function drawArcs(gArc, model, cells) {
    var adjacent = Object.create(null);
    model.regions.forEach(function (r, i) {
      cells[i].tags.forEach(function (j) {
        if (j >= 0) adjacent[Math.min(i, j) + ":" + Math.max(i, j)] = true;
      });
    });
    var maxMass = 0;
    model.links.forEach(function (l) { if (l.mass > maxMass) maxMass = l.mass; });

    model.links.forEach(function (l) {
      if (adjacent[l.a + ":" + l.b]) return;
      var ca = centroid(cells[l.a].poly), cb = centroid(cells[l.b].poly);
      var mx = (ca[0] + cb[0]) / 2, my = (ca[1] + cb[1]) / 2;
      var dx = cb[0] - ca[0], dy = cb[1] - ca[1];
      var len = Math.sqrt(dx * dx + dy * dy) || 1;
      var bow = Math.min(len * 0.18, 70);
      var qx = mx - (dy / len) * bow, qy = my + (dx / len) * bow;
      var path = svg("path", {
        d: "M" + ca[0].toFixed(1) + "," + ca[1].toFixed(1) +
           " Q" + qx.toFixed(1) + "," + qy.toFixed(1) +
           " " + cb[0].toFixed(1) + "," + cb[1].toFixed(1),
        class: l.merge ? "arc merge" : "arc",
        "stroke-width": (0.7 + 3 * Math.sqrt(l.mass / maxMass)).toFixed(2)
      });
      var t = svg("title");
      t.textContent = model.regions[l.a].key + " ↔ " + model.regions[l.b].key +
        " — not adjacent · " + plural(l.pairs, "pair") + ", " + l.merge + " merge-worthy";
      path.appendChild(t);
      path.addEventListener("click", function (ev) { ev.stopPropagation(); selectBorder(l); });
      gArc.appendChild(path);
    });
  }

  function attachPanZoom(host, view) {
    var k = 1, tx = 0, ty = 0, dragging = false, lx = 0, ly = 0;
    function apply() { view.setAttribute("transform", "translate(" + tx + "," + ty + ") scale(" + k + ")"); }
    host.addEventListener("wheel", function (ev) {
      ev.preventDefault();
      var pt = host.createSVGPoint(); pt.x = ev.clientX; pt.y = ev.clientY;
      var loc = pt.matrixTransform(host.getScreenCTM().inverse());
      var f = Math.exp(-ev.deltaY * 0.0016), nk = Math.max(0.4, Math.min(14, k * f));
      f = nk / k;
      tx = loc.x - (loc.x - tx) * f; ty = loc.y - (loc.y - ty) * f; k = nk;
      apply();
    }, { passive: false });
    host.addEventListener("mousedown", function (ev) { dragging = true; lx = ev.clientX; ly = ev.clientY; });
    window.addEventListener("mouseup", function () { dragging = false; });
    host.addEventListener("mousemove", function (ev) {
      if (!dragging) return;
      var m = host.getScreenCTM();
      tx += (ev.clientX - lx) / m.a; ty += (ev.clientY - ly) / m.d;
      lx = ev.clientX; ly = ev.clientY;
      apply();
    });
    host.addEventListener("click", function () { selection = null; renderSelection(); });
  }

  /* ── Selection panel ─────────────────────────────────────────────────── */

  function selectRegion(i) { selection = { kind: "region", i: i }; renderSelection(); }
  function selectBorder(l) { selection = { kind: "border", link: l }; renderSelection(); }

  function renderSelection() {
    var host = $("selection-panel");
    clear(host);
    if (!mapState) return;
    var model = mapState.model;

    if (!selection) {
      host.appendChild(el("div", "control-label", "Selection"));
      host.appendChild(el("p", "control-note", "Click a region for what it holds, or a painted border for the pairs that cross it."));
      return;
    }

    if (selection.kind === "region") {
      var r = model.regions[selection.i];
      host.appendChild(el("div", "control-label", partitionEl.value === "concept" ? "Concept" : "Package"));
      host.appendChild(el("div", "sel-title mono", r.key));
      host.appendChild(el("p", "control-note",
        plural(r.members.length, "function") + " · " + plural(r.misfits, "misfit") + " · " +
        (r.internal.pairs ? plural(r.internal.pairs, "pair") + " inside it, " + r.internal.merge + " merge-worthy"
          : "no pairs inside it")));
      var linked = model.links.filter(function (l) { return l.a === selection.i || l.b === selection.i; });
      if (linked.length) {
        host.appendChild(el("div", "control-label", "Shares code with"));
        var ul = el("ul", "sel-list");
        linked.slice(0, 8).forEach(function (l) {
          var other = model.regions[l.a === selection.i ? l.b : l.a];
          var li = el("li");
          li.appendChild(el("span", "mono", other.key));
          li.appendChild(el("span", "sel-meta", plural(l.pairs, "pair") + " · " + l.merge + " merge-worthy"));
          li.addEventListener("click", function () { selectBorder(l); });
          ul.appendChild(li);
        });
        host.appendChild(ul);
        if (linked.length > 8) host.appendChild(el("p", "control-note", (linked.length - 8) + " more not listed."));
      }
      return;
    }

    var l = selection.link;
    host.appendChild(el("div", "control-label", "Border"));
    host.appendChild(el("div", "sel-title mono", model.regions[l.a].key + " ↔ " + model.regions[l.b].key));
    host.appendChild(el("p", "control-note",
      plural(l.pairs, "pair") + " cross it · " + l.merge + " merge-worthy · evidence " + fixed(l.mass, 0)));
    var ol = el("ul", "sel-list");
    l.edges.slice(0, 10).forEach(function (ei) {
      var e = edges[ei];
      var li = el("li");
      li.appendChild(el("span", "mono", units[e.a].key + " ↔ " + units[e.b].key));
      li.appendChild(el("span", "sel-meta", "shape " + fixed(e.shape) + " · overlap " + fixed(e.overlap) +
        (e.merge ? " · merge-worthy" : "")));
      li.addEventListener("click", function () {
        selectUnit(e.a, e.b); show("neighbourhood"); writeHash();
      });
      ol.appendChild(li);
    });
    host.appendChild(ol);
    if (l.edges.length > 10) host.appendChild(el("p", "control-note", (l.edges.length - 10) + " more not listed."));
  }

  function renderLegend() {
    var host = $("legend");
    clear(host);
    var pooled = 0, pooledConcepts = 0;
    conceptRows.forEach(function (row) {
      if (conceptColor[row.id]) {
        var line = el("div", "legend-row");
        var dot = el("span", "legend-dot");
        dot.style.background = conceptColor[row.id];
        line.appendChild(dot);
        line.appendChild(el("span", "legend-id", row.id));
        line.appendChild(el("span", "legend-n mono", comma(row.dominant)));
        line.title = row.id + " — leads " + plural(row.dominant, "function") +
          ", carried by " + comma(row.carried);
        host.appendChild(line);
      } else if (row.dominant > 0) {
        pooled += row.dominant;
        pooledConcepts++;
      }
    });

    var uncoloured = 0;
    units.forEach(function (u) { if (!u.concept) uncoloured++; });
    if (pooled || uncoloured) {
      var other = el("div", "legend-row");
      var d = el("span", "legend-dot");
      d.style.background = OTHER;
      other.appendChild(d);
      other.appendChild(el("span", "legend-id",
        pooledConcepts ? plural(pooledConcepts, "other concept") + " and unassigned" : "no concept"));
      other.appendChild(el("span", "legend-n mono", comma(pooled + uncoloured)));
      host.appendChild(other);
    }
    host.appendChild(el("p", "control-note",
      plural(conceptRows.length, "concept") + " learned from this corpus — the vocabulary is " +
      "derived from what the code does, not read off a fixed list."));
  }

  var redrawTimer = null;
  function scheduleDraw() {
    clearTimeout(redrawTimer);
    redrawTimer = setTimeout(function () { selection = null; drawMap(); }, 140);
  }

  cutEl.addEventListener("input", function () {
    cutLabel.textContent = fixed(parseFloat(cutEl.value), metricEl.value === "rank" ? 1 : 2);
    scheduleDraw();
  });
  metricEl.addEventListener("change", function () {
    syncCutRange();
    cutLabel.textContent = fixed(parseFloat(cutEl.value), metricEl.value === "rank" ? 1 : 2);
    scheduleDraw();
  });
  [partitionEl, mergeOnly, showArcs].forEach(function (n) {
    n.addEventListener("change", scheduleDraw);
  });

  /* ── Neighbourhood ───────────────────────────────────────────────────── */

  var selUnit = null, selOther = null;

  /* Units worth listing are the ones with a neighbour; a function in no pair
     has no neighbourhood to show. */
  var pairedUnits = units.filter(function (u) { return edgesByUnit[u.id]; })
    .sort(function (a, b) {
      var ea = edgesByUnit[a.id], eb = edgesByUnit[b.id];
      return edges[eb[0]].rank - edges[ea[0]].rank || (a.key < b.key ? -1 : 1);
    });

  /* The picker is not the corpus: it is the units that landed in at least one
     scored pair, in the order of each one's single best-corroborated pair. Both
     facts are invisible from the list itself, so the note states them. */
  function renderPickerNote() {
    $("nb-picker-note").textContent =
      plural(pairedUnits.length, "function") + " of " + comma(units.length) +
      " appear in a scored pair. Listed best-corroborated pair first.";
  }

  function renderUnitList() {
    var host = $("nb-units");
    var q = $("search").value.trim().toLowerCase();
    clear(host);
    var shown = 0;
    pairedUnits.forEach(function (u) {
      if (q && u.key.toLowerCase().indexOf(q) < 0 && u.file.toLowerCase().indexOf(q) < 0) return;
      if (++shown > 400) return;
      var li = el("li");
      li.setAttribute("aria-selected", u.id === selUnit ? "true" : "false");
      li.appendChild(el("span", "u-name mono", u.key));
      li.appendChild(el("span", "u-meta", (edgesByUnit[u.id] || []).length + " neighbours · " + u.package));
      li.addEventListener("click", function () { selectUnit(u.id, null); writeHash(); });
      host.appendChild(li);
    });
    if (shown === 0) host.appendChild(el("li", "empty", "No function matches."));
    else if (shown > 400) host.appendChild(el("li", "empty", "Showing the first 400 — narrow the filter."));
  }
  $("search").addEventListener("input", renderUnitList);

  function selectUnit(id, other) {
    selUnit = id;
    var list = edgesByUnit[id] || [];
    selOther = null;
    if (other !== null && other !== undefined) {
      for (var i = 0; i < list.length; i++) {
        var e = edges[list[i]];
        if (e.a === other || e.b === other) { selOther = other; break; }
      }
    }
    if (selOther === null && list.length) {
      var first = edges[list[0]];
      selOther = first.a === id ? first.b : first.a;
    }
    renderUnitList();
    renderNeighbourhood();
  }

  function renderNeighbourhood() {
    var head = $("nb-head"), listHost = $("nb-list"), detail = $("nb-detail"), bodies = $("nb-bodies");
    clear(head); clear(listHost); clear(detail); clear(bodies);

    if (selUnit === null || !units[selUnit]) {
      head.appendChild(el("p", "empty", "Pick a function on the left."));
      return;
    }
    var u = units[selUnit];
    head.appendChild(el("h2", "mono", u.key));
    var meta = el("div", "fact-s");
    meta.textContent = u.file + ":" + u.line + " · role " + u.role +
      " · fan-in " + u.fanIn + " / fan-out " + u.fanOut + " · " + u.nodes + " nodes" +
      (u.fit >= 0 ? " · habitat fit " + fixed(u.fit) : "") + (u.misfit ? " (misfit)" : "");
    head.appendChild(meta);
    if (u.concepts && u.concepts.length) {
      var tags = el("div", "tagset");
      u.concepts.forEach(function (c) {
        var chip = el("span", "chip" + (c.id === u.concept ? " on" : ""));
        chip.appendChild(el("span", "chip-id", c.id));
        /* The confidence is shown because it is the finding: a learned concept
           is carried by degree, so whether both sides of a pair really mean it
           is exactly what a reader is judging. */
        chip.appendChild(el("span", "chip-n mono", fixed(c.confidence)));
        var sw = el("span", "chip-dot");
        sw.style.background = colorFor(c.id);
        chip.insertBefore(sw, chip.firstChild);
        tags.appendChild(chip);
      });
      head.appendChild(tags);
    }

    var list = edgesByUnit[selUnit] || [];
    if (!list.length) {
      listHost.appendChild(el("li", "empty", "No scored neighbour."));
      return;
    }
    list.forEach(function (ei) {
      var e = edges[ei];
      var otherId = e.a === selUnit ? e.b : e.a;
      var o = units[otherId];
      var li = el("li", "nb-row");
      li.setAttribute("aria-selected", otherId === selOther ? "true" : "false");
      li.appendChild(el("span", "mono", o.key));
      /* Rank is what this list is sorted by, so it leads: without it a sorted
         list reads as an unsorted one. */
      var bits = "rank " + fixed(e.rank, 1) + " · shape " + fixed(e.shape) +
        " · overlap " + fixed(e.overlap) +
        " · evidence " + fixed(e.total, 1) + (e.merge ? " · merge-worthy" : "");
      if (e.kind) bits += " · " + e.kind;
      li.appendChild(el("span", "r-meta", bits));
      li.addEventListener("click", function () { selOther = otherId; renderNeighbourhood(); writeHash(); });
      listHost.appendChild(li);
    });

    renderPair(detail, bodies, selUnit, selOther);
  }

  function panel(title, note) {
    var p = el("div", "panel");
    p.appendChild(el("div", "panel-title", title));
    if (note) p.appendChild(el("p", "panel-note", note));
    return p;
  }

  function renderPair(host, bodyHost, aId, bId) {
    var e = null;
    (edgesByUnit[aId] || []).some(function (ei) {
      var c = edges[ei];
      if (c.a === bId || c.b === bId) { e = c; return true; }
      return false;
    });
    if (!e) { host.appendChild(el("p", "empty", "Select a neighbour.")); return; }

    /* The clicked function reads on the left whichever side of the pair it is,
       so moving down the neighbour list does not swap the panes underneath. */
    var left = aId, right = bId;

    /* Every tile carries what its number is measured in, and the two with a
       floor carry this run's floor — derived here rather than pinned, because a
       calibrated run moves both, which is the whole reason they belong on the
       tile rather than in the masthead alone. Being under a floor is muted, not
       flagged: --struct-min filters, but --threshold gates only the structural
       channel, so a concept- or call-retrieved pair sits under it legitimately. */
    var scores = el("div", "scores");
    /* Containment sits beside code-shape and is never blended into it: a
       symmetric Jaccard and "how much of the smaller side is inside the
       larger" are different findings about the same pair. It has no floor of
       its own — nothing gates on it — so it carries a reading, not a number. */
    [["code-shape", fixed(e.shape), "floor " + fixed(facts.threshold), e.shape < facts.threshold],
     ["containment", fixed(e.containment), "smaller side inside larger"],
     ["overlap", fixed(e.overlap), "gate " + fixed(facts.structMin), e.overlap < facts.structMin],
     ["evidence", fixed(e.total, 1), "nats"],
     ["trophic", fixed(e.trophic), "shared / total"],
     ["call-sim", fixed(e.callSim), "ranks test pairs"],
     ["rank", fixed(e.rank, 3), "evidence × shape × overlap × trophic²"]].forEach(function (s) {
      var b = el("div", "score");
      b.appendChild(el("div", "score-n mono" + (s[3] ? " below" : ""), s[1]));
      b.appendChild(el("div", "score-l", s[0]));
      if (s[2]) b.appendChild(el("div", "score-s", s[2]));
      scores.appendChild(b);
    });
    var mergeNote = "Merge-worthy asserts all three: overlap at or above the " +
      fixed(facts.structMin) + " gate, at least 2 of 5 architectural signals, and " +
      "code-shape at or above 0.40. It is a gate, not a verdict — the bodies below are.";
    var top = panel("This pair", e.kindNote ? mergeNote + " " + e.kindNote : mergeNote);
    top.appendChild(scores);
    if (e.channels && e.channels.length) {
      top.appendChild(el("p", "panel-note", "Retrieved via " + e.channels.join(" + ") +
        (e.merge ? " · merge-worthy" : " · not merge-worthy") +
        (e.cross ? " · different packages" : " · same package")));
    }
    host.appendChild(top);

    var comps = panel("Code-shape components",
      "The four weighted parts of the code-shape number above — wl 0.60, flow 0.20, " +
      "nesting 0.05, sig 0.15 — and two that are reported but never scored. Size, " +
      "because the WL Jaccard already penalises a size mismatch through its union; " +
      "containment, because it answers a different question, and a 40-line function " +
      "whose whole shape reappears inside a 400-line one is a finding no blended " +
      "number states.");
    var bars = el("div", "bars");
    ["wl", "flow", "nesting", "sig", "size", "containment"].forEach(function (name, i) {
      var v = e.breakdown[i];
      var row = el("div", "bar-row");
      row.appendChild(el("span", null, name));
      var track = el("div", "bar-track");
      var fill = el("div", "bar-fill" + (v >= 0.9 ? " hot" : ""));
      fill.style.width = Math.max(0, Math.min(100, v * 100)) + "%";
      track.appendChild(fill);
      row.appendChild(track);
      row.appendChild(el("span", "bar-n", fixed(v)));
      bars.appendChild(row);
    });
    comps.appendChild(bars);
    host.appendChild(comps);

    if (e.explain) {
      host.appendChild(panel("What the canonicalizer did", e.explain));
    }

    if (e.chains && e.chains.length) {
      var ch = panel("Shared structure",
        "The highest-energy labels both bodies carry — the evidence the score rests on. " +
        "A label is a hash of a whole subtree, so it can be named and counted but not " +
        "pointed at; they are listed rather than highlighted below.");
      var ol = el("ol", "chains");
      e.chains.forEach(function (c) {
        var li = el("li");
        li.appendChild(el("span", "c-energy", fixed(c.energy)));
        li.appendChild(el("span", "c-level", "h" + c.depth));
        li.appendChild(el("span", "c-render", c.render));
        ol.appendChild(li);
      });
      ch.appendChild(ol);
      host.appendChild(ch);
    }

    if (!(e.chains && e.chains.length) && !(e.reasons && e.reasons.length)) {
      host.appendChild(panel("Shared structure and context",
        "Not inlined for this pair — the page bounds how much per-pair detail it carries, and this " +
        "one ranks below the cut. Its scores above are complete. Run doppel on a narrower subtree " +
        "to get the full evidence for it."));
    }

    if (e.reasons && e.reasons.length) {
      var rs = panel("Shared context");
      var ul = el("ul", "reasons");
      e.reasons.forEach(function (r) { ul.appendChild(el("li", null, r)); });
      rs.appendChild(ul);
      host.appendChild(rs);
    }

    bodyHost.appendChild(renderBodies(left, right));
  }

  /* ── Bodies and the text diff ────────────────────────────────────────── */

  function renderBodies(aId, bId) {
    var a = units[aId], b = units[bId];
    var ta = bodyByUnit[aId], tb = bodyByUnit[bId];

    var p = panel("Both bodies",
      "A line-level text diff: tinted lines are present on one side only, grey rules mark where " +
      "the other side has lines this one does not. It is a textual comparison — not the structural " +
      "claim the score makes, which is what shared structure above reports.");

    var grid = el("div", "bodies");
    var la = ta ? ta.split("\n") : null;
    var lb = tb ? tb.split("\n") : null;
    var marks = { a: null, b: null };
    if (la && lb && la.length <= DIFF_MAX_LINES && lb.length <= DIFF_MAX_LINES) {
      marks = diffLines(la, lb);
    }
    grid.appendChild(bodyPane(a, la, marks.a));
    grid.appendChild(bodyPane(b, lb, marks.b));
    p.appendChild(grid);
    return p;
  }

  function bodyPane(u, lines, marks) {
    var pane = el("div", "body-pane");
    pane.appendChild(el("div", "body-head mono", u.key + " — " + u.file + ":" + u.line));
    if (!lines) {
      pane.appendChild(el("div", "body-missing",
        "Source not inlined for this function — the page bounds how much it carries. " +
        "It is at " + u.file + ":" + u.line + "."));
      return pane;
    }
    var pre = el("pre", "body-code");
    lines.forEach(function (line, i) {
      var span = el("span", "ln" + (marks && marks[i] ? " add" : ""), line === "" ? " " : line);
      pre.appendChild(span);
    });
    pane.appendChild(pre);
    return pane;
  }

  /* Longest common subsequence over lines. Bodies are tens to hundreds of
     lines, so the quadratic table is cheap; DIFF_MAX_LINES is the guard for
     the pathological case rather than an expected path. */
  function diffLines(a, b) {
    var n = a.length, m = b.length;
    var dp = new Uint32Array((n + 1) * (m + 1));
    var w = m + 1;
    for (var i = n - 1; i >= 0; i--) {
      for (var j = m - 1; j >= 0; j--) {
        dp[i * w + j] = a[i] === b[j]
          ? dp[(i + 1) * w + (j + 1)] + 1
          : Math.max(dp[(i + 1) * w + j], dp[i * w + (j + 1)]);
      }
    }
    var ma = new Array(n).fill(false), mb = new Array(m).fill(false);
    var x = 0, y = 0;
    while (x < n && y < m) {
      if (a[x] === b[y]) { x++; y++; }
      else if (dp[(x + 1) * w + y] >= dp[x * w + (y + 1)]) { ma[x] = true; x++; }
      else { mb[y] = true; y++; }
    }
    while (x < n) { ma[x++] = true; }
    while (y < m) { mb[y++] = true; }
    return { a: ma, b: mb };
  }

  /* ── Hash routing ────────────────────────────────────────────────────── */

  function writeHash() {
    var h = current === "map" ? "#map"
      : (selUnit === null ? "#neighbourhood"
        : (selOther === null ? "#unit/" + selUnit : "#pair/" + selUnit + "-" + selOther));
    if (location.hash !== h) history.replaceState(null, "", h);
  }

  function readHash() {
    var h = (location.hash || "").replace(/^#/, "");
    var m;
    if ((m = h.match(/^pair\/(\d+)-(\d+)$/))) { selectUnit(+m[1], +m[2]); show("neighbourhood"); return; }
    if ((m = h.match(/^unit\/(\d+)$/))) { selectUnit(+m[1], null); show("neighbourhood"); return; }
    if (h === "neighbourhood") { show("neighbourhood"); return; }
    show("map");
  }
  window.addEventListener("hashchange", readHash);


  /* ── Boot ────────────────────────────────────────────────────────────── */

  renderFacts();
  renderColophon();
  renderLegend();
  renderPickerNote();
  syncCutRange();
  cutLabel.textContent = fixed(parseFloat(cutEl.value), 2);
  if (pairedUnits.length) selectUnit(pairedUnits[0].id, null);
  renderUnitList();
  renderNeighbourhood();
  drawMap();
  readHash();

  /* The map is sized from its container, so a resize needs the whole diagram
     back — cheap, and debounced, because the fitting loop is not free. */
  var resizeTimer = null;
  window.addEventListener("resize", function () {
    if (current !== "map") return;
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(drawMap, 200);
  });
})();
