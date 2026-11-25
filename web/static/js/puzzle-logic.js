/* Puzzle-Logik ohne DOM: Schiebepuzzle (lösbares Mischen über Inversionen,
   Züge) und Jigsaw (Nasen/Löcher der Kanten, Einrasten).
   Läuft im Browser (window.PuzzleLogic) und in Node (Tests in tests/js/). */
(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.PuzzleLogic = api;
})(this, function () {
  'use strict';

  /* ---------- Schiebepuzzle ----------
     tiles[pos] = Kachel-ID; ID n*n-1 ist die Lücke. Gelöst: tiles[i] === i. */
  function solved(n) {
    var a = [];
    for (var i = 0; i < n * n; i++) a.push(i);
    return a;
  }

  function isSolved(tiles) {
    for (var i = 0; i < tiles.length; i++) if (tiles[i] !== i) return false;
    return true;
  }

  function blankIndex(tiles) { return tiles.indexOf(tiles.length - 1); }

  function inversions(tiles) {
    var blank = tiles.length - 1, inv = 0;
    for (var i = 0; i < tiles.length; i++) {
      if (tiles[i] === blank) continue;
      for (var j = i + 1; j < tiles.length; j++) {
        if (tiles[j] !== blank && tiles[i] > tiles[j]) inv++;
      }
    }
    return inv;
  }

  // Ungerade Breite: lösbar ⇔ Inversionen gerade.
  // Gerade Breite: lösbar ⇔ Inversionen + Zeile der Lücke (von unten, ab 1) ungerade.
  function isSolvable(tiles, n) {
    var inv = inversions(tiles);
    if (n % 2 === 1) return inv % 2 === 0;
    var rowFromBottom = n - Math.floor(blankIndex(tiles) / n);
    return (inv + rowFromBottom) % 2 === 1;
  }

  function misplaced(tiles) {
    var m = 0;
    for (var i = 0; i < tiles.length; i++) if (tiles[i] !== i && tiles[i] !== tiles.length - 1) m++;
    return m;
  }

  // Zufällige, lösbare Anordnung, die nicht (fast) schon gelöst ist.
  function shuffleSolvable(n, rnd) {
    rnd = rnd || Math.random;
    var minWrong = Math.ceil((n * n - 1) * 0.6);
    for (var tries = 0; tries < 1000; tries++) {
      var t = solved(n);
      for (var i = t.length - 1; i > 0; i--) {
        var j = Math.floor(rnd() * (i + 1));
        var x = t[i]; t[i] = t[j]; t[j] = x;
      }
      if (!isSolvable(t, n)) {
        // Tausch zweier Kacheln (nicht der Lücke) kippt die Parität.
        var a = -1, b = -1;
        for (var k = 0; k < t.length && b < 0; k++) {
          if (t[k] === t.length - 1) continue;
          if (a < 0) a = k; else b = k;
        }
        var y = t[a]; t[a] = t[b]; t[b] = y;
      }
      if (!isSolved(t) && misplaced(t) >= minWrong) return t;
    }
    return null; // praktisch unerreichbar
  }

  // Kachel an idx (gleiche Zeile/Spalte wie die Lücke) Richtung Lücke schieben;
  // dazwischenliegende Kacheln rutschen mit. Liefert ein neues Array oder null.
  function slide(tiles, n, idx) {
    var b = blankIndex(tiles);
    if (idx === b || idx < 0 || idx >= tiles.length) return null;
    var br = Math.floor(b / n), bc = b % n, r = Math.floor(idx / n), c = idx % n;
    var step;
    if (r === br) step = c < bc ? -1 : 1;
    else if (c === bc) step = r < br ? -n : n;
    else return null;
    var t = tiles.slice();
    var pos = b;
    while (pos !== idx) {
      t[pos] = t[pos + step];
      pos += step;
    }
    t[idx] = tiles.length - 1;
    return t;
  }

  // Wisch-Richtung → Index der Kachel, die in die Lücke rutscht (oder -1).
  // 'left' heißt: die Kachel rechts neben der Lücke wandert nach links.
  function swipeTarget(tiles, n, dir) {
    var b = blankIndex(tiles), r = Math.floor(b / n), c = b % n;
    if (dir === 'left') return c < n - 1 ? b + 1 : -1;
    if (dir === 'right') return c > 0 ? b - 1 : -1;
    if (dir === 'up') return r < n - 1 ? b + n : -1;
    if (dir === 'down') return r > 0 ? b - n : -1;
    return -1;
  }

  /* ---------- Jigsaw ----------
     edges.h[r][c]: Kante unter Teil (r,c) (r < n-1), edges.v[r][c]: Kante rechts von (r,c) (c < n-1).
     +1 = Teil oben/links hat dort eine Nase, das Nachbarteil das passende Loch. */
  function makeEdges(n, rnd) {
    rnd = rnd || Math.random;
    var hz = [], vt = [], r, c;
    for (r = 0; r < n; r++) {
      hz.push([]); vt.push([]);
      for (c = 0; c < n; c++) {
        hz[r].push(r < n - 1 ? (rnd() < 0.5 ? 1 : -1) : 0);
        vt[r].push(c < n - 1 ? (rnd() < 0.5 ? 1 : -1) : 0);
      }
    }
    return { h: hz, v: vt };
  }

  // Seiten eines Teils: +1 Nase nach außen, -1 Loch, 0 gerader Rand.
  function pieceEdges(edges, n, r, c) {
    return {
      top: r > 0 ? -edges.h[r - 1][c] : 0,
      right: c < n - 1 ? edges.v[r][c] : 0,
      bottom: r < n - 1 ? edges.h[r][c] : 0,
      left: c > 0 ? -edges.v[r][c - 1] : 0
    };
  }

  function validEdges(edges, n) {
    if (!edges || !edges.h || !edges.v || edges.h.length !== n || edges.v.length !== n) return false;
    for (var r = 0; r < n; r++) {
      if (!edges.h[r] || !edges.v[r] || edges.h[r].length !== n || edges.v[r].length !== n) return false;
      for (var c = 0; c < n; c++) {
        var hv = edges.h[r][c], vv = edges.v[r][c];
        if ((r < n - 1) !== (hv === 1 || hv === -1) || (c < n - 1) !== (vv === 1 || vv === -1)) return false;
      }
    }
    return true;
  }

  function snaps(x, y, tx, ty, tol) {
    var dx = x - tx, dy = y - ty;
    return dx * dx + dy * dy <= tol * tol;
  }

  return {
    solved: solved, isSolved: isSolved, blankIndex: blankIndex, inversions: inversions,
    isSolvable: isSolvable, shuffleSolvable: shuffleSolvable, slide: slide, swipeTarget: swipeTarget,
    makeEdges: makeEdges, pieceEdges: pieceEdges, validEdges: validEdges, snaps: snaps
  };
});
