/* doppel review page.
 *
 * A classic script inlined into a file:// page, no library — the contract
 * timeline.js and the dashboard screens hold. Everything in the payload was
 * settled by internal/identity against two snapshots: the classes, scores,
 * verdicts and explanation sentences are read back, never recomputed, and this
 * page has neither body nor bag. What it decides is order and wording.
 *
 * The order is a reviewer's: merge-worthy pairs the change introduced, then the
 * ones it removed, then what happened to each function. Pairs no classified
 * change explains are counted apart and never presented as the change's doing.
 */
(function () {
  "use strict";

  var DATA = JSON.parse(document.getElementById("doppel-data").textContent);
  var U = DOPPEL_UI;
  var $ = U.$, el = U.el, add = U.append, fixed = U.fixed, comma = U.comma, plural = U.plural, chip = U.chip;

  var ch = DATA.change || {};
  var created = ch.created || [];
  var dissolved = ch.dissolved || [];
  var changes = ch.changes || [];
  var counts = ch.counts || [];
  var bounds = DATA.bounds || {};

  var where = {};
  (DATA.units || []).forEach(function (u) { where[u.key] = u; });
  function loc(key) { var u = where[key]; return u ? u.file + ":" + u.line : ""; }

  /* identity's report order and the one colour each class keeps on the page. */
  var CLASS_ORDER = ["split", "merged", "moved", "renamed", "edited", "new", "deleted"];
  var CLASS_INK = {
    split: "var(--color-accent-700)", merged: "var(--color-accent-700)",
    moved: "var(--color-text)", renamed: "var(--shape-ink)", edited: "var(--ink-soft)",
    "new": "var(--overlap-ink)", deleted: "var(--ink-faint)"
  };

  /* A classified side is one whose class is anything but unchanged — the
     rule identity.PairChange.Attributable states. */
  function classified(c) { return !!c && c !== "unchanged"; }
  function causeOf(p) {
    var parts = [];
    if (classified(p.aClass)) parts.push(p.aClass + " " + p.a);
    if (classified(p.bClass)) parts.push(p.bClass + " " + p.b);
    return parts.length ? parts.join(", ") : "no change on either side explains it — retrieval re-ranking";
  }

  function copy(text, btn) {
    var done = function () { var t = btn.textContent; btn.textContent = "Copied"; setTimeout(function () { btn.textContent = t; }, 1200); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done, function () { fallback(); });
    } else fallback();
    function fallback() {
      var ta = el("textarea"); ta.value = text; ta.style.position = "fixed"; ta.style.opacity = "0";
      document.body.appendChild(ta); ta.select();
      try { document.execCommand("copy"); done(); } catch (e) { /* nothing to copy into */ }
      document.body.removeChild(ta);
    }
  }

  /* A finding as a review comment: what it is, where both sides live, and the
     stored sentence, in markdown a forge renders. */
  function reviewComment(p, dir) {
    var lines = [
      "**doppel:** " + (dir === "created" ? "new near-duplicate" : "near-duplicate removed") +
        (p.mergeWorthy ? " (merge-worthy)" : "") + " — code-shape " + fixed(p.score) + ", overlap " + fixed(p.overlap),
      "- `" + p.a + "`" + (loc(p.a) ? " — " + loc(p.a) : ""),
      "- `" + p.b + "`" + (loc(p.b) ? " — " + loc(p.b) : "")
    ];
    if (p.explain) lines.push("", "_" + p.explain + "_");
    if (dir === "created") lines.push("", "Reuse or extend the existing one, or say in one line why this copy is deliberate.");
    return lines.join("\n");
  }

  /* ── Masthead ─────────────────────────────────────────────────────────── */

  $("au-sub").textContent = DATA.base + " → " + DATA.head + " · " +
    comma(DATA.oldFunctions) + " → " + comma(DATA.newFunctions) + " functions · " + DATA.params;

  var root = $("au");

  /* ── Verdict ──────────────────────────────────────────────────────────── */

  var reclassified = 0;
  counts.forEach(function (c) { if (c.class !== "unchanged") reclassified += c.count; });
  var mwUp = DATA.createdMergeWorthy || 0, mwDown = DATA.dissolvedMergeWorthy || 0;

  function verdictLine() {
    if (!mwUp && !mwDown && !reclassified) return "Nothing changed: no function was reclassified and no pair appeared or vanished.";
    var s;
    if (mwUp && mwDown) s = "This change introduced " + plural(mwUp, "merge-worthy pair") + " and removed " + comma(mwDown) + ".";
    else if (mwUp) s = "This change introduced " + plural(mwUp, "merge-worthy pair") + " and removed none.";
    else if (mwDown) s = "This change removed " + plural(mwDown, "merge-worthy pair") + " and introduced none.";
    else s = "No merge-worthy pair appeared or vanished.";
    var newOnes = created.filter(function (p) { return p.mergeWorthy && (p.aClass === "new" || p.bClass === "new"); }).length;
    if (newOnes) s += " Of those introduced, " + (newOnes === 1 ? "one involves" : comma(newOnes) + " involve") + " a function that did not exist before.";
    return s;
  }

  var verdict = el("section", "au-verdict");
  verdict.setAttribute("aria-label", "Net effect");
  var vt = add(el("div", "au-verdict-text"), el("div", "au-verdict-kicker", "Net effect"), el("p", "au-verdict-line", verdictLine()));
  var tallies = el("div", "au-tallies");
  function tally(n, cls, label) {
    return add(el("div", "au-tally"), el("span", "au-tally-n " + (cls || ""), n), el("span", "au-tally-l", label));
  }
  add(tallies,
    tally((mwUp ? "+" : "") + comma(mwUp), mwUp ? "up" : "", "merge-worthy introduced"),
    tally((mwDown ? "−" : "") + comma(mwDown), mwDown ? "down" : "", "merge-worthy removed"),
    tally(comma(reclassified), "", "functions reclassified"));
  add(root, add(verdict, vt, tallies));

  var grid = el("div", "au-grid");
  var main = el("div", "au-main");
  var side = el("aside", "au-side");
  side.setAttribute("aria-label", "Context");
  add(root, add(grid, main, side));

  /* ── Pair sections ────────────────────────────────────────────────────── */

  function pairSection(id, title, sub, list, total, dir) {
    var sec = el("section", "au-section");
    sec.id = id;
    var head = add(el("div", "au-section-head"), el("h2", "section-title", title));
    head.appendChild(U.scoreKey(id === "au-introduced"));
    sec.appendChild(head);
    sec.appendChild(el("p", "au-note", sub));

    var worthy = list.filter(function (p) { return p.mergeWorthy; });
    var onlyWorthy = { on: worthy.length > 0 };
    var toggle = el("label", "au-toggle");
    var box = el("input"); box.type = "checkbox"; box.checked = onlyWorthy.on;
    add(toggle, box, " merge-worthy only (" + comma(worthy.length) + " of " + comma(list.length) + " listed)");
    if (list.length) sec.appendChild(toggle);

    var ol = el("ol", "au-list");
    var more = el("div", "au-more");
    sec.appendChild(ol);
    sec.appendChild(more);

    function draw() {
      ol.textContent = "";
      var rows = onlyWorthy.on ? worthy : list;
      if (!rows.length) {
        ol.appendChild(el("li", "empty", list.length ? "None is merge-worthy — untick the filter to see the rest." : "None."));
      }
      rows.forEach(function (p, i) { ol.appendChild(pairRow(p, dir, id + "-" + i)); });
      var held = total - list.length;
      more.textContent = held > 0 ? "The payload lists the first " + comma(list.length) + " of " + comma(total) +
        " (bounded at " + comma(bounds.maxPairs) + "); `doppel diff` prints them all." : "";
    }
    box.addEventListener("change", function () { onlyWorthy.on = box.checked; draw(); });
    draw();
    return sec;
  }

  function pairRow(p, dir, id) {
    var li = el("li");
    var btn = el("button", "au-row" + (dir === "dissolved" ? " gone" : ""));
    btn.type = "button";
    btn.setAttribute("aria-expanded", "false");
    btn.setAttribute("aria-controls", id);
    var edge = el("span", "au-edge " + (p.attributable ? dir : "churn"));
    var who = add(el("span", "au-who"),
      el("span", "au-fn", p.a), el("span", "au-fn", p.b), el("span", "au-cause", causeOf(p)));
    var tags = el("span", "au-tags");
    if (p.mergeWorthy) tags.appendChild(chip("merge-worthy", "solid"));
    if (!p.attributable) tags.appendChild(chip("not this change's doing", "quiet"));
    if (tags.firstChild) who.appendChild(tags);
    var scores = add(el("span", "au-scores"), U.bar(p.score, "shape"), U.bar(p.overlap, "overlap"));
    add(btn, edge, who, scores);

    var detail = el("div", "au-detail");
    detail.id = id;
    detail.hidden = true;
    if (p.explain) add(detail, add(el("div"), el("span", "muted", "explain: "), p.explain));
    var locs = [p.a, p.b].map(function (k) { return loc(k) ? k + " — " + loc(k) : ""; }).filter(Boolean);
    if (locs.length) detail.appendChild(el("div", "au-loc", locs.join("  ·  ")));
    if (dir === "created" && p.attributable) {
      detail.appendChild(el("div", null, p.mergeWorthy
        ? "Resolve before merging: reuse or extend the existing function, or record in one line why this copy is deliberate."
        : "Below the merge verdict: a candidate to glance at, not a finding."));
    }
    var actions = el("div", "au-actions");
    var c = el("button", "btn btn-primary", "Copy as review comment");
    c.type = "button";
    c.addEventListener("click", function () { copy(reviewComment(p, dir), c); });
    actions.appendChild(c);
    detail.appendChild(actions);

    btn.addEventListener("click", function () {
      var open = btn.getAttribute("aria-expanded") !== "true";
      btn.setAttribute("aria-expanded", open ? "true" : "false");
      detail.hidden = !open;
    });
    return add(li, btn, detail);
  }

  main.appendChild(pairSection("au-introduced", "Introduced — resolve before you merge",
    "Near-duplicate pairs present after the change and not before, attributed to a classified change first.",
    created, ch.createdTotal || created.length, "created"));
  main.appendChild(pairSection("au-removed", "Removed — credit the change",
    "Pairs present before and gone after. A rename re-keys every pair its function held, so it shows here and above as the same fact restated.",
    dissolved, ch.dissolvedTotal || dissolved.length, "dissolved"));

  /* ── Functions ────────────────────────────────────────────────────────── */

  var fsec = el("section", "au-section");
  fsec.id = "au-functions";
  add(fsec, add(el("div", "au-section-head"), el("h2", "section-title", "What happened to each function"),
    el("span", "card-sub", "matched by body, not name — a rename is one line, not a delete and an add")));

  var countOf = {};
  counts.forEach(function (c) { countOf[c.class] = c.count; });
  var filter = { cls: "" };
  var chips = el("div", "au-classes");
  var list = el("div", "au-changes");
  var fmore = el("div", "au-more");
  var chipEls = [];
  CLASS_ORDER.forEach(function (cls) {
    var b = el("button", "au-class", cls + " " + comma(countOf[cls] || 0));
    b.type = "button";
    b.disabled = !countOf[cls];
    b.setAttribute("aria-pressed", "false");
    b.addEventListener("click", function () {
      filter.cls = filter.cls === cls ? "" : cls;
      chipEls.forEach(function (x) { x.b.setAttribute("aria-pressed", x.cls === filter.cls ? "true" : "false"); });
      drawChanges();
    });
    chipEls.push({ b: b, cls: cls });
    chips.appendChild(b);
  });
  chips.appendChild(el("span", "au-more", comma(countOf.unchanged || 0) + " unchanged, counted and not listed"));

  function describe(c) {
    var o = (c.old || []).join(" + "), n = (c["new"] || []).join(" + ");
    if (c.class === "new") return n;
    if (c.class === "deleted") return o;
    if (o === n) return n;
    return o + " → " + n;
  }
  function evidence(c) {
    if (c.class === "new" || c.class === "deleted") return "—";
    if (c.class === "split" || c.class === "merged") return "by containment";
    var s = "jaccard " + fixed(c.jaccard);
    if (c.digestEqual) s += " · digest =";
    return s;
  }
  function secondary(c) {
    var bits = [];
    if (c.class === "moved" && c.nameChanged) bits.push("also renamed");
    if (c.class !== "edited" && c.class !== "new" && c.class !== "deleted" && !c.digestEqual &&
        c.class !== "split" && c.class !== "merged") bits.push("body edited");
    return bits.join("; ");
  }
  function drawChanges() {
    list.textContent = "";
    var rows = changes.filter(function (c) { return !filter.cls || c.class === filter.cls; });
    if (!rows.length) list.appendChild(el("div", "empty", "No function was reclassified."));
    rows.forEach(function (c) {
      var row = el("div", "au-change");
      var badge = el("span", "au-badge", c.class);
      badge.style.color = CLASS_INK[c.class] || "var(--ink-soft)";
      var what = el("span", "au-what", describe(c));
      var extra = [secondary(c), c.file ? c.file + ":" + c.line : ""].filter(Boolean).join(" · ");
      if (extra) what.appendChild(el("small", null, extra));
      add(list, add(row, badge, what, el("span", "au-ev", evidence(c))));
    });
    var held = (ch.changesTotal || 0) - changes.length;
    fmore.textContent = held > 0 ? "The payload lists the first " + comma(changes.length) + " of " +
      comma(ch.changesTotal) + " (bounded at " + comma(bounds.maxChanges) + "); `doppel diff` prints them all." : "";
  }
  drawChanges();
  add(main, add(fsec, chips, list, fmore));

  /* ── Side column ──────────────────────────────────────────────────────── */

  var attrCreated = ch.attributableNew || 0;
  var createdTotal = ch.createdTotal || created.length;
  var dissolvedTotal = ch.dissolvedTotal || dissolved.length;
  var attrDissolved = DATA.dissolvedAttributable || 0;
  var churn = (createdTotal - attrCreated) + (dissolvedTotal - attrDissolved);
  var shareMax = Math.max(1, attrCreated, attrDissolved, churn);

  function share(label, n, kind) {
    var row = add(el("div", "au-share"), add(el("div", "au-share-row"), el("span", null, label), el("span", "mono", comma(n))));
    var bar = U.bar(n / shareMax, kind, "");
    return add(row, bar);
  }
  var attr = add(el("section", "card"),
    el("h3", "card-title", "How much of this is the change's doing"),
    share("pairs created by a classified change", attrCreated, "overlap"),
    share("pairs dissolved by a classified change", attrDissolved, "shape"),
    share("pairs no change explains", churn, "neutral"),
    el("p", "au-note", "Retrieval keeps a bounded top-K per function, so code elsewhere can move a pair in or out without either body changing. Those are counted here and never blamed on the change."));
  side.appendChild(attr);

  var trust = add(el("section", "card"), el("h3", "card-title", "Reading the comparison"));
  if (bounds.reportTop || bounds.reportMaxPerFunc) {
    trust.appendChild(el("p", "au-note warn",
      "These snapshots store the ranked report list (--top " + bounds.reportTop + ", --max-per-func " +
      bounds.reportMaxPerFunc + "), not the full candidate set, so a pair can appear or vanish here because a cutoff moved. " +
      "Re-run analyze with --top 0 --max-per-func 0 for the whole set."));
  }
  (DATA.notes || []).forEach(function (n) { trust.appendChild(el("p", "au-note warn", "Note: " + n)); });
  trust.appendChild(el("p", "au-note",
    "Functions are matched by body: an identical fingerprint digest is the same body, and the corpus-weighted overlap of the two bodies decides everything else. Every line carries the evidence that produced its class."));
  trust.appendChild(el("p", "au-note",
    "Scores and explanation sentences are those the run holding each pair recorded; nothing on this page is recomputed."));
  side.appendChild(trust);

  $("colophon").textContent = "doppel diff " + DATA.base + " " + DATA.head + " · " + DATA.params +
    " · same input, same bytes: this page is deterministic.";
})();
