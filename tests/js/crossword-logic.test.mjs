// Tests für web/static/js/crossword-logic.js – ausführen mit `make test` bzw. `node --test tests/js/`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';

const require = createRequire(import.meta.url);
const X = require('../../web/static/js/crossword-logic.js');

// K A T Z E · ·
// U · · · · · ·
// S · G R Ü ß E
// S · · · · · ·
const _ = null;
const small = {
  width: 7, height: 4,
  cells: [
    ['K', 'A', 'T', 'Z', 'E', _, _],
    ['U', _, _, _, _, _, _],
    ['S', _, 'G', 'R', 'Ü', 'ß', 'E'],
    ['S', _, _, _, _, _, _],
  ],
  entries: [
    { number: 1, direction: 'across', row: 0, col: 0, answer: 'KATZE', clue: 'Miau' },
    { number: 1, direction: 'down', row: 0, col: 0, answer: 'KUSS', clue: 'Schmatz' },
    { number: 2, direction: 'across', row: 2, col: 2, answer: 'GRÜßE', clue: 'Jeden Morgen' },
  ],
};
const idx = (m, r, c) => r * m.W + c;

test('normalize / lettersOf: Umlaute, ß, NFD', () => {
  assert.equal(X.normalize('grüße'), 'GRÜßE');
  assert.equal(X.normalize('STRAẞE'), 'STRAßE');
  assert.equal(X.normalize('Grüße'), 'GRÜßE');
  assert.deepEqual(X.lettersOf('ä'), ['Ä']);
  assert.deepEqual(X.lettersOf('ä'), ['Ä']);
  assert.deepEqual(X.lettersOf('ß'), ['ß']);
  assert.deepEqual(X.lettersOf(' 1-x'), ['X']);
});

test('model: Nummern, Felder, Reihenfolge waagerecht vor senkrecht', () => {
  const m = X.model(small);
  assert.equal(m.entries.length, 3);
  assert.deepEqual(m.entries.map(e => e.dir + e.number), ['across1', 'across2', 'down1']);
  assert.equal(m.num[0], 1);
  assert.equal(m.num[idx(m, 2, 2)], 2);
  assert.deepEqual(m.entries[1].cells, [16, 17, 18, 19, 20]);
  assert.equal(m.entries[1].answer, 'GRÜßE');
  assert.equal(X.entryOf(m, 0, 'down'), 2);
});

test('select: Tippen wählt Wort, gleiches Feld wechselt Richtung', () => {
  const m = X.model(small), st = X.state(m);
  assert.equal(st.cell, 0); assert.equal(st.dir, 'across');
  X.select(m, st, 0);
  assert.equal(st.dir, 'down', 'gleiches Feld → senkrecht');
  X.select(m, st, 0);
  assert.equal(st.dir, 'across');
  X.select(m, st, idx(m, 2, 0)); // S in KUSS: nur senkrecht
  assert.equal(st.dir, 'down');
  X.select(m, st, idx(m, 2, 0));
  assert.equal(st.dir, 'down', 'keine waagerechte Alternative → bleibt');
  assert.equal(X.select(m, st, 1 * 7 + 1), false, 'Block nicht wählbar');
});

test('input: eintragen, weiterspringen, am Ende zum nächsten Wort mit Lücken', () => {
  const m = X.model(small), st = X.state(m);
  for (const ch of 'katze') X.input(m, st, ch);
  assert.deepEqual(st.letters.slice(0, 5), ['K', 'A', 'T', 'Z', 'E']);
  // KATZE voll → nächstes Wort mit Lücken: GRÜßE
  assert.equal(st.dir, 'across');
  assert.equal(st.cell, idx(m, 2, 2));
  for (const ch of ['g', 'r', 'ü', 'ß', 'e']) X.input(m, st, ch);
  assert.equal(st.letters[idx(m, 2, 5)], 'ß');
  // weiter zu KUSS, erstes leeres Feld (U)
  assert.equal(st.dir, 'down');
  assert.equal(st.cell, idx(m, 1, 0));
  assert.equal(X.input(m, st, '1'), false, 'Ziffern werden ignoriert');
  X.input(m, st, 'u'); X.input(m, st, 's'); X.input(m, st, 's');
  assert.ok(X.isFilled(m, st));
  assert.ok(X.isSolved(m, st));
  assert.equal(X.solvedEntries(m, st), 3);
});

test('backspace: volles Feld leeren, leeres Feld → zurück und leeren', () => {
  const m = X.model(small), st = X.state(m);
  X.input(m, st, 'K'); X.input(m, st, 'A');
  assert.equal(st.cell, 2);
  X.backspace(m, st); // Feld 2 leer → zurück auf 1 und leeren
  assert.equal(st.cell, 1);
  assert.equal(st.letters[1], '');
  X.input(m, st, 'X');
  X.select(m, st, 1);
  X.select(m, st, 1); // Richtung bleibt across (kein senkrechtes Wort)
  X.backspace(m, st); // voll → leeren, bleibt stehen
  assert.equal(st.cell, 1);
  assert.equal(st.letters[1], '');
  X.select(m, st, 0); X.backspace(m, st); X.backspace(m, st);
  assert.equal(st.cell, 0, 'am Wortanfang bleibt die Auswahl');
});

test('move: Pfeiltasten wechseln erst die Richtung und überspringen Blöcke', () => {
  const m = X.model(small), st = X.state(m);
  X.move(m, st, 1, 0); // ↓ auf K: erst Richtung
  assert.equal(st.dir, 'down'); assert.equal(st.cell, 0);
  X.move(m, st, 1, 0);
  assert.equal(st.cell, 7);
  X.move(m, st, 0, 1); // → auf U: rechts nur Blöcke → bleibt
  assert.equal(st.cell, 7);
  X.select(m, st, idx(m, 2, 0));
  X.move(m, st, 0, 1); // → über den Block zu G
  assert.equal(st.cell, idx(m, 2, 2));
  assert.equal(st.dir, 'across');
});

test('nextEntry: zyklisch, bevorzugt Wörter mit Lücken', () => {
  const m = X.model(small), st = X.state(m);
  X.nextEntry(m, st, 1);
  assert.equal(X.current(m, st).answer, 'GRÜßE');
  X.nextEntry(m, st, 1);
  assert.equal(X.current(m, st).answer, 'KUSS');
  X.nextEntry(m, st, 1);
  assert.equal(X.current(m, st).answer, 'KATZE');
  X.nextEntry(m, st, -1);
  assert.equal(X.current(m, st).answer, 'KUSS');
  X.selectEntry(m, st, 0); for (const ch of 'KATZE') X.input(m, st, ch);
  X.selectEntry(m, st, 2); X.nextEntry(m, st, 1); // KATZE voll → überspringen
  assert.equal(X.current(m, st).answer, 'GRÜßE');
});

test('check, reveal, Fortschritt speichern/laden', () => {
  const m = X.model(small), st = X.state(m);
  for (const ch of 'KOTZE') X.input(m, st, ch);
  assert.deepEqual(X.check(m, st), [1]);
  assert.equal(st.wrong[1], true);
  X.select(m, st, 1);
  X.input(m, st, 'A');
  assert.equal(st.wrong[1], false, 'neuer Buchstabe hebt Markierung auf');
  X.select(m, st, idx(m, 2, 4));
  X.reveal(m, st);
  assert.equal(st.letters[idx(m, 2, 4)], 'Ü');
  X.input(m, st, 'X'); // aufgedeckt bleibt
  assert.equal(st.letters[idx(m, 2, 4)], 'Ü');
  const saved = JSON.parse(JSON.stringify(X.serialize(st)));
  const st2 = X.state(m, saved);
  assert.deepEqual(st2.letters, st.letters);
  assert.equal(st2.revealed[idx(m, 2, 4)], true);
  // falsche Länge (anderes Raster) → frisch
  assert.equal(X.state(m, { letters: ['K'] }).letters[0], '');
  assert.deepEqual(X.counts(m, st), { open: 13, filled: 6, right: 6 });
});

test('diffValue: Einfügen, Löschen, Komposition, Ersetzen', () => {
  assert.deepEqual(X.diffValue(' ', ' A'), { del: 0, ins: 'A' });
  assert.deepEqual(X.diffValue(' ', ''), { del: 1, ins: '' });
  assert.deepEqual(X.diffValue(' A', ' AB'), { del: 0, ins: 'B' }); // Android-Komposition wächst
  assert.deepEqual(X.diffValue(' AB', ' A'), { del: 1, ins: '' });
  assert.deepEqual(X.diffValue(' AB', ' Ab'), { del: 1, ins: 'b' }); // Ersetzung
  assert.deepEqual(X.diffValue(' ', ' ÄÖ'), { del: 0, ins: 'ÄÖ' }); // insertText mit mehreren Zeichen
  assert.deepEqual(X.diffValue('', ''), { del: 0, ins: '' });
});

test('Beispiel-Raster (doors/assets-example/kreuzwort.json) ist lösbar', () => {
  const data = JSON.parse(readFileSync(new URL('../../doors/assets-example/kreuzwort.json', import.meta.url)));
  const m = X.model(data), st = X.state(m);
  for (let i = 0; i < m.entries.length; i++) {
    X.selectEntry(m, st, i);
    const e = m.entries[i];
    st.cell = e.cells[0];
    for (const ch of e.answer) X.input(m, st, ch);
  }
  assert.ok(X.isSolved(m, st));
  assert.equal(X.solvedEntries(m, st), m.entries.length);
});
