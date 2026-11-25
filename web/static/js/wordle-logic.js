/* Wordle-Logik ohne DOM: Normalisierung (NFC, Großbuchstaben, ß bleibt ß),
   Bewertung mit korrekter Behandlung doppelter Buchstaben, Tastenfarben.
   Läuft im Browser (window.WordleLogic) und in Node (module.exports, Tests in
   tests/js/wordle-logic.test.mjs). Gleiche Regeln wie doors.NormalizeWord in Go. */
(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.WordleLogic = api;
})(this, function () {
  'use strict';
  var LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÜß';
  var COMPOSE = { A: 'Ä', O: 'Ö', U: 'Ü', a: 'ä', o: 'ö', u: 'ü' };
  var RANK = { absent: 1, present: 2, correct: 3 };

  function normalize(s) {
    s = String(s === null || s === undefined ? '' : s).replace(/^\s+|\s+$/g, '');
    if (s.normalize) { try { s = s.normalize('NFC'); } catch (e) {} }
    var out = '';
    for (var i = 0; i < s.length; i++) {
      var ch = s.charAt(i);
      if (s.charAt(i + 1) === '̈' && COMPOSE[ch]) { ch = COMPOSE[ch]; i++; }
      // 'ß'.toUpperCase() wäre "SS" – ß ist hier aber ein eigener Buchstabe.
      if (ch === 'ß' || ch === 'ẞ') out += 'ß';
      else out += ch.toUpperCase();
    }
    return out;
  }

  function isLetter(ch) { return typeof ch === 'string' && ch.length === 1 && LETTERS.indexOf(ch) >= 0; }

  // Wort → Array einzelner Buchstaben (alle Buchstaben liegen in der BMP).
  function letters(word) { return normalize(word).split(''); }

  // Ergebnis je Position: 'correct' (grün), 'present' (gelb), 'absent' (grau).
  // Erst alle Treffer an der richtigen Stelle, dann gelb nur so oft, wie der
  // Buchstabe in der Lösung noch übrig ist.
  function score(guess, answer) {
    var g = letters(guess), a = letters(answer);
    var res = [], left = {}, i;
    for (i = 0; i < g.length; i++) res.push('absent');
    for (i = 0; i < a.length; i++) {
      if (g[i] === a[i]) res[i] = 'correct';
      else left[a[i]] = (left[a[i]] || 0) + 1;
    }
    for (i = 0; i < g.length; i++) {
      if (res[i] === 'correct') continue;
      if (left[g[i]] > 0) { res[i] = 'present'; left[g[i]]--; }
    }
    return res;
  }

  function isWin(result) {
    for (var i = 0; i < result.length; i++) if (result[i] !== 'correct') return false;
    return result.length > 0;
  }

  // Tastenfarben: der beste je gesehene Zustand gewinnt (grün > gelb > grau).
  function mergeKeys(keys, guess, result) {
    var out = {}, k;
    for (k in keys) if (keys.hasOwnProperty(k)) out[k] = keys[k];
    var g = letters(guess);
    for (var i = 0; i < g.length; i++) {
      var cur = out[g[i]];
      if (!cur || RANK[result[i]] > RANK[cur]) out[g[i]] = result[i];
    }
    return out;
  }

  // Prüft einen Versuch: {ok: true} oder {ok: false, reason: 'short'|'long'|'letters', missing: n}
  function check(guess, length) {
    var g = letters(guess);
    for (var i = 0; i < g.length; i++) if (!isLetter(g[i])) return { ok: false, reason: 'letters' };
    if (g.length < length) return { ok: false, reason: 'short', missing: length - g.length };
    if (g.length > length) return { ok: false, reason: 'long' };
    return { ok: true };
  }

  return { LETTERS: LETTERS, normalize: normalize, isLetter: isLetter, letters: letters, score: score, isWin: isWin, mergeKeys: mergeKeys, check: check };
});
