/* Kreuzworträtsel-Logik ohne DOM: Raster-Modell, Auswahl (Feld + Richtung),
   Eingabe mit Weiterspringen, Löschen, Pfeiltasten, nächster Hinweis, Prüfen,
   Aufdecken, Fertig-Erkennung und der Werte-Vergleich für das versteckte
   Eingabefeld (Android-Komposition / keyCode 229).
   Läuft im Browser (window.CrosswordLogic) und in Node (Tests in
   tests/js/crossword-logic.test.mjs). Normalisierung wie crossword.Normalize in Go. */
(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.CrosswordLogic = api;
})(this, function () {
  'use strict';
  var LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÜß';
  var COMBINING_DIAERESIS = String.fromCharCode(0x308);
  var COMPOSE = { A: 'Ä', O: 'Ö', U: 'Ü', a: 'ä', o: 'ö', u: 'ü' };

  function normalize(s) {
    s = String(s === null || s === undefined ? '' : s);
    if (s.normalize) { try { s = s.normalize('NFC'); } catch (e) {} }
    var out = '';
    for (var i = 0; i < s.length; i++) {
      var ch = s.charAt(i);
      if (s.charAt(i + 1) === COMBINING_DIAERESIS && COMPOSE[ch]) { ch = COMPOSE[ch]; i++; }
      // 'ß'.toUpperCase() wäre "SS" – im Raster ist ß ein Kästchen.
      if (ch === 'ß' || ch === 'ẞ') out += 'ß';
      else out += ch.toUpperCase();
    }
    return out;
  }

  function isLetter(ch) { return typeof ch === 'string' && ch.length === 1 && LETTERS.indexOf(ch) >= 0; }

  // Buchstaben aus beliebigem getipptem Text (z. B. "ä", "Ä", "a" + Trema).
  function lettersOf(text) {
    var n = normalize(text), out = [];
    for (var i = 0; i < n.length; i++) if (isLetter(n.charAt(i))) out.push(n.charAt(i));
    return out;
  }

  /* ---------- Modell ---------- */
  function model(data) {
    var W = data.width, H = data.height, i, r, c;
    var m = { W: W, H: H, sol: [], num: [], entries: [], across: [], down: [] };
    for (r = 0; r < H; r++) {
      for (c = 0; c < W; c++) {
        var v = data.cells[r] && data.cells[r][c];
        m.sol.push(v ? normalize(v) : null);
        m.num.push(0);
        m.across.push(-1);
        m.down.push(-1);
      }
    }
    var list = data.entries.slice();
    // Reihenfolge der Hinweislisten: erst waagerecht, dann senkrecht, je nach Nummer.
    list.sort(function (a, b) {
      if (a.direction !== b.direction) return a.direction === 'across' ? -1 : 1;
      return a.number - b.number;
    });
    for (i = 0; i < list.length; i++) {
      var e = list[i], cells = [];
      var dr = e.direction === 'down' ? 1 : 0, dc = dr ? 0 : 1;
      var len = lettersOf(e.answer).length;
      for (var k = 0; k < len; k++) cells.push((e.row + dr * k) * W + e.col + dc * k);
      var entry = { i: i, number: e.number, dir: e.direction, row: e.row, col: e.col, answer: normalize(e.answer), clue: e.clue, cells: cells };
      m.entries.push(entry);
      m.num[cells[0]] = e.number;
      for (k = 0; k < cells.length; k++) (dr ? m.down : m.across)[cells[k]] = i;
    }
    return m;
  }

  function isOpen(m, idx) { return idx >= 0 && idx < m.sol.length && m.sol[idx] !== null; }
  function entryOf(m, idx, dir) { return idx < 0 ? -1 : (dir === 'down' ? m.down : m.across)[idx]; }
  function other(dir) { return dir === 'down' ? 'across' : 'down'; }

  /* ---------- Zustand ---------- */
  function state(m, saved) {
    var st = { letters: [], revealed: [], wrong: [], cell: -1, dir: 'across' };
    for (var i = 0; i < m.sol.length; i++) { st.letters.push(''); st.revealed.push(false); st.wrong.push(false); }
    if (saved && saved.letters instanceof Array && saved.letters.length === m.sol.length) {
      for (i = 0; i < m.sol.length; i++) {
        var l = saved.letters[i];
        if (isOpen(m, i) && isLetter(l)) st.letters[i] = l;
        if (isOpen(m, i) && saved.revealed && saved.revealed[i]) { st.revealed[i] = true; st.letters[i] = m.sol[i]; }
      }
    }
    if (m.entries.length) selectEntry(m, st, 0);
    return st;
  }

  function serialize(st) { return { letters: st.letters.slice(), revealed: st.revealed.slice() }; }

  function current(m, st) {
    var e = entryOf(m, st.cell, st.dir);
    return e >= 0 ? m.entries[e] : null;
  }

  // Tippen auf ein Feld: gleiches Feld = Richtung wechseln (wenn es dort ein Wort gibt).
  function select(m, st, idx) {
    if (!isOpen(m, idx)) return false;
    if (idx === st.cell) {
      if (entryOf(m, idx, other(st.dir)) >= 0) st.dir = other(st.dir);
      return true;
    }
    st.cell = idx;
    if (entryOf(m, idx, st.dir) < 0) st.dir = other(st.dir);
    return true;
  }

  function firstEmpty(m, st, e) {
    for (var k = 0; k < e.cells.length; k++) if (!st.letters[e.cells[k]]) return e.cells[k];
    return -1;
  }

  // Hinweis antippen / nächster Hinweis: zum ersten leeren Feld des Worts.
  function selectEntry(m, st, ei) {
    var e = m.entries[ei];
    if (!e) return false;
    st.dir = e.dir;
    var f = firstEmpty(m, st, e);
    st.cell = f >= 0 ? f : e.cells[0];
    return true;
  }

  function nextEntry(m, st, delta) {
    var n = m.entries.length;
    if (!n) return false;
    var cur = entryOf(m, st.cell, st.dir);
    var start = cur < 0 ? (delta > 0 ? -1 : 0) : cur;
    // bevorzugt Wörter, in denen noch etwas fehlt
    for (var step = 1; step <= n; step++) {
      var ei = ((start + delta * step) % n + n) % n;
      if (firstEmpty(m, st, m.entries[ei]) >= 0 || isFilled(m, st)) return selectEntry(m, st, ei);
    }
    return selectEntry(m, st, ((start + delta) % n + n) % n);
  }

  function setLetter(m, st, idx, ch) {
    if (st.revealed[idx]) return;
    st.letters[idx] = ch;
    st.wrong[idx] = false;
  }

  // Ein getippter Buchstabe: eintragen und weiter. Am Wortende geht es zum ersten
  // leeren Feld dieses Worts bzw. zum nächsten Wort mit Lücken.
  function input(m, st, ch) {
    ch = normalize(ch);
    if (!isLetter(ch) || !isOpen(m, st.cell)) return false;
    var e = current(m, st);
    setLetter(m, st, st.cell, ch);
    if (!e) return true;
    var pos = e.cells.indexOf(st.cell);
    if (pos < e.cells.length - 1) {
      st.cell = e.cells[pos + 1];
      return true;
    }
    var f = firstEmpty(m, st, e);
    if (f >= 0) st.cell = f;
    else if (!isFilled(m, st)) nextEntry(m, st, 1);
    return true;
  }

  // Löschen: volles Feld leeren; leeres Feld → ein Feld zurück und das leeren.
  function backspace(m, st) {
    if (!isOpen(m, st.cell)) return false;
    if (st.letters[st.cell] && !st.revealed[st.cell]) { setLetter(m, st, st.cell, ''); return true; }
    var e = current(m, st);
    if (!e) return false;
    var pos = e.cells.indexOf(st.cell);
    if (pos > 0) {
      st.cell = e.cells[pos - 1];
      setLetter(m, st, st.cell, '');
      return true;
    }
    return false;
  }

  // Pfeiltasten: erst die Richtung wechseln, dann Feld für Feld (Blöcke werden übersprungen).
  function move(m, st, dr, dc) {
    var dir = dc ? 'across' : 'down';
    if (dir !== st.dir && entryOf(m, st.cell, dir) >= 0) { st.dir = dir; return true; }
    var r = Math.floor(st.cell / m.W), c = st.cell % m.W;
    for (;;) {
      r += dr; c += dc;
      if (r < 0 || c < 0 || r >= m.H || c >= m.W) return false;
      var idx = r * m.W + c;
      if (isOpen(m, idx)) {
        st.cell = idx;
        if (entryOf(m, idx, st.dir) < 0) st.dir = other(st.dir);
        return true;
      }
    }
  }

  /* ---------- Prüfen / Fertig ---------- */
  function check(m, st) {
    var wrong = [];
    for (var i = 0; i < m.sol.length; i++) {
      st.wrong[i] = !!(isOpen(m, i) && st.letters[i] && st.letters[i] !== m.sol[i]);
      if (st.wrong[i]) wrong.push(i);
    }
    return wrong;
  }

  function reveal(m, st) {
    if (!isOpen(m, st.cell)) return false;
    st.letters[st.cell] = m.sol[st.cell];
    st.revealed[st.cell] = true;
    st.wrong[st.cell] = false;
    return true;
  }

  function counts(m, st) {
    var open = 0, filled = 0, right = 0;
    for (var i = 0; i < m.sol.length; i++) {
      if (!isOpen(m, i)) continue;
      open++;
      if (st.letters[i]) filled++;
      if (st.letters[i] === m.sol[i]) right++;
    }
    return { open: open, filled: filled, right: right };
  }
  function isFilled(m, st) { var c = counts(m, st); return c.filled === c.open; }
  function isSolved(m, st) { var c = counts(m, st); return c.right === c.open; }
  function entrySolved(m, st, ei) {
    var e = m.entries[ei];
    for (var k = 0; k < e.cells.length; k++) if (st.letters[e.cells[k]] !== m.sol[e.cells[k]]) return false;
    return true;
  }
  function solvedEntries(m, st) {
    var n = 0;
    for (var i = 0; i < m.entries.length; i++) if (entrySolved(m, st, i)) n++;
    return n;
  }

  /* ---------- verstecktes Eingabefeld ---------- */
  // Vergleicht den alten und neuen Wert des Eingabefelds. Liefert, wie viele
  // Zeichen gelöscht und welche eingefügt wurden – unabhängig davon, ob die
  // Tastatur einzelne Zeichen, ganze Kompositionen (Android) oder Ersetzungen schickt.
  function diffValue(prev, next) {
    prev = String(prev || ''); next = String(next || '');
    var p = 0;
    while (p < prev.length && p < next.length && prev.charAt(p) === next.charAt(p)) p++;
    return { del: prev.length - p, ins: next.slice(p) };
  }

  return {
    LETTERS: LETTERS, normalize: normalize, isLetter: isLetter, lettersOf: lettersOf,
    model: model, state: state, serialize: serialize, isOpen: isOpen, entryOf: entryOf, current: current,
    select: select, selectEntry: selectEntry, nextEntry: nextEntry, input: input, backspace: backspace, move: move,
    check: check, reveal: reveal, counts: counts, isFilled: isFilled, isSolved: isSolved,
    entrySolved: entrySolved, solvedEntries: solvedEntries, diffValue: diffValue
  };
});
