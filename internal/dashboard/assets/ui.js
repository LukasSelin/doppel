/* doppel pages — shared formatting and DOM helpers.
 *
 * Everything here is independent of any payload, which is why it is its own
 * file: the single-run dashboard (core.js re-exports it on DOPPEL) and the
 * two-run review page (audit.js) both load it, rather than each carrying a
 * copy doppel would find as a clone of the other. Nothing here may read the
 * page's data, the clock or object-key order.
 */
var DOPPEL_UI = (function () {
  "use strict";

  var SVGNS = "http://www.w3.org/2000/svg";

  /* ── Formatting ──────────────────────────────────────────────────────── */

  function fixed(v, n) { return (v === undefined || v === null || v < 0) ? "—" : Number(v).toFixed(n === undefined ? 2 : n); }
  /* A missing count reads as 0, not "undefined": payloads omit zero fields. */
  function comma(n) { return String(n === undefined || n === null ? 0 : n).replace(/\B(?=(\d{3})+(?!\d))/g, ","); }
  function pct(n, d) { return d > 0 ? Math.round((100 * n) / d) : 0; }
  function plural(n, one, many) { return comma(n) + " " + (n === 1 ? one : (many || one + "s")); }
  function cmp(a, b) { return a < b ? -1 : a > b ? 1 : 0; }

  /* ── DOM ─────────────────────────────────────────────────────────────── */

  function $(id) { return document.getElementById(id); }
  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined && text !== null) n.textContent = text;
    return n;
  }
  function svg(tag, attrs) {
    var n = document.createElementNS(SVGNS, tag);
    Object.keys(attrs || {}).forEach(function (k) { n.setAttribute(k, attrs[k]); });
    return n;
  }
  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); return node; }
  function append(parent) {
    for (var i = 1; i < arguments.length; i++) {
      var c = arguments[i];
      if (c === null || c === undefined || c === false) continue;
      parent.appendChild(typeof c === "string" ? document.createTextNode(c) : c);
    }
    return parent;
  }
  function link(href, cls, text) { var a = el("a", cls, text); a.href = href; return a; }

  /* A pill. variant: "solid" (the merge-worthy verdict), "outline" (a kind or
     a scope), "quiet" (a filtered-out kind still shown on request). */
  function chip(text, variant) { return el("span", "chip chip-" + (variant || "outline"), text); }

  /* One score as a track and a fill. kind is "shape", "overlap" or "neutral";
     the colour belongs to the quantity, never to the screen, so code-shape is
     the same blue everywhere and overlap the same magenta. */
  function bar(value, kind, label) {
    var row = el("div", "bar bar-" + (kind || "neutral"));
    var track = el("span", "bar-track");
    var fill = el("span", "bar-fill");
    fill.style.width = Math.max(0, Math.min(1, value || 0)) * 100 + "%";
    track.appendChild(fill);
    row.appendChild(track);
    row.appendChild(el("span", "bar-value mono", label !== undefined ? label : fixed(value)));
    return row;
  }

  /* The two-colour legend, said once per view that shows both scores.
     verbose adds what each score means, for the first place a reader meets
     them. One definition, because four screens each wrote their own. */
  function scoreKey(verbose) {
    var k = el("div", "score-key");
    append(k,
      append(el("span"), el("i", "k-shape"), verbose ? "code-shape — how alike the bodies are" : "code-shape"),
      append(el("span"), el("i", "k-overlap"), verbose ? "overlap — how much context they share" : "overlap"));
    return k;
  }

  /* A big number with a label and a note, the unit of every headline strip. */
  function stat(value, label, note, unit) {
    var s = el("div", "stat");
    s.appendChild(el("div", "stat-label", label));
    var v = el("div", "stat-value-row");
    v.appendChild(el("span", "stat-value", value));
    if (unit) v.appendChild(el("span", "stat-unit mono", unit));
    s.appendChild(v);
    if (note) s.appendChild(el("div", "stat-note", note));
    return s;
  }

  return {
    fixed: fixed, comma: comma, pct: pct, plural: plural, cmp: cmp,
    $: $, el: el, svg: svg, clear: clear, append: append, link: link,
    chip: chip, bar: bar, stat: stat, scoreKey: scoreKey
  };
})();
