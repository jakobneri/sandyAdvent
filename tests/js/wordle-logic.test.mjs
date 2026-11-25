// Tests für web/static/js/wordle-logic.js – ausführen mit `make test` bzw. `node --test tests/js/`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const W = require('../../web/static/js/wordle-logic.js');

const C = 'correct', P = 'present', A = 'absent';

test('normalize: Großbuchstaben, ß bleibt ß, NFC', () => {
  assert.equal(W.normalize('kekse'), 'KEKSE');
  assert.equal(W.normalize(' Grüße '), 'GRÜßE');
  assert.equal(W.normalize('straße'), 'STRAßE');
  assert.equal(W.normalize('STRAẞE'), 'STRAßE'); // großes ẞ → ß
  assert.equal(W.normalize('Grüße'), 'GRÜßE'); // zerlegtes ü (NFD)
  assert.equal(W.normalize('äöü'), 'ÄÖÜ');
  assert.equal(W.letters('Grüße').length, 5);
});

test('isLetter', () => {
  for (const ch of 'AZÄÖÜß') assert.ok(W.isLetter(ch), ch);
  for (const ch of ['a', 'ä', '1', ' ', '-', 'SS', 'É', '']) assert.ok(!W.isLetter(ch), JSON.stringify(ch));
});

test('score: einfache Fälle', () => {
  assert.deepEqual(W.score('KEKSE', 'KEKSE'), [C, C, C, C, C]);
  assert.deepEqual(W.score('MAUSI', 'KEKSE'), [A, A, A, C, A]);
  assert.deepEqual(W.score('SAUNA', 'KEKSE'), [P, A, A, A, A]);
  assert.ok(W.isWin(W.score('kekse', 'KEKSE')));
  assert.ok(!W.isWin(W.score('KEKSA', 'KEKSE')));
});

test('score: doppelte Buchstaben – erst grün, dann gelb nach Restanzahl', () => {
  // Lösung hat ein E an Position 1: zweites E im Versuch bleibt grau
  assert.deepEqual(W.score('EEXXX', 'XEXXX'), [A, C, C, C, C]);
  // ein gelbes E, das zweite E grau
  assert.deepEqual(W.score('EEAAA', 'BBBBE'), [P, A, A, A, A]);
  // Lösung mit zwei E: beide gelb
  assert.deepEqual(W.score('EEAAA', 'BBBEE'), [P, P, A, A, A]);
  // grün hat Vorrang, auch wenn das grüne E weiter hinten steht
  assert.deepEqual(W.score('EAAAE', 'BBBBE'), [A, A, A, A, C]);
  // klassisches Beispiel
  assert.deepEqual(W.score('SPEED', 'ABIDE'), [A, A, P, A, P]);
  assert.deepEqual(W.score('KEKSE', 'SEKTE'), [A, C, C, P, C]);
  assert.deepEqual(W.score('LLAMA', 'HELLO'), [P, P, A, A, A]);
});

test('score: Umlaute und ß sind eigene Buchstaben', () => {
  assert.deepEqual(W.score('GRUSE', 'GRÜßE'), [C, C, A, A, C]);
  assert.deepEqual(W.score('GRÜSSE', 'GRÜßEN'), [C, C, C, A, A, P]);
  assert.deepEqual(W.score('grüße', 'GRÜßE'), [C, C, C, C, C]);
  assert.deepEqual(W.score('ÄÖÜßA', 'AÄÖÜß'), [P, P, P, P, P]);
  assert.deepEqual(W.score('GLUHWEIN', 'GLÜHWEIN'), [C, C, A, C, C, C, C, C]);
  // NFD-Eingabe zählt wie NFC
  assert.deepEqual(W.score('GLÜHWEIN', 'GLÜHWEIN'), Array(8).fill(C));
});

test('score: lange Wörter bis 12 Buchstaben', () => {
  const w = 'WEIHNACHTSZE';
  assert.equal(w.length, 12);
  assert.ok(W.isWin(W.score(w, w)));
  assert.deepEqual(W.score('AAAAAAAAAAAA', w), [A, A, A, A, A, C, A, A, A, A, A, A]);
});

test('mergeKeys: grün > gelb > grau', () => {
  let k = {};
  k = W.mergeKeys(k, 'EEAAA', W.score('EEAAA', 'BBBBE'));
  assert.equal(k.E, P);
  assert.equal(k.A, A);
  k = W.mergeKeys(k, 'BBBBE', W.score('BBBBE', 'BBBBE'));
  assert.equal(k.E, C);
  k = W.mergeKeys(k, 'EXXXX', W.score('EXXXX', 'BBBBE'));
  assert.equal(k.E, C, 'grün darf nicht zurückgestuft werden');
});

test('check: Länge und Buchstaben', () => {
  assert.deepEqual(W.check('KEK', 5), { ok: false, reason: 'short', missing: 2 });
  assert.equal(W.check('KEKSEE', 5).reason, 'long');
  assert.equal(W.check('KEK5E', 5).reason, 'letters');
  assert.ok(W.check('grüße', 5).ok);
});
