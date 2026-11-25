// Tests für web/static/js/puzzle-logic.js – ausführen mit `make test` bzw. `node --test tests/js/`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const P = require('../../web/static/js/puzzle-logic.js');

// deterministischer Zufall
function rng(seed) {
  let s = seed >>> 0;
  return () => ((s = (s * 1664525 + 1013904223) >>> 0) / 4294967296);
}

// alle vom gelösten Zustand erreichbaren Anordnungen (Breitensuche)
function reachable(n) {
  const start = P.solved(n).join(',');
  const seen = new Set([start]);
  const queue = [P.solved(n)];
  while (queue.length) {
    const t = queue.pop();
    for (let i = 0; i < t.length; i++) {
      const next = P.slide(t, n, i);
      if (!next) continue;
      const key = next.join(',');
      if (!seen.has(key)) { seen.add(key); queue.push(next); }
    }
  }
  return seen;
}

function permutations(a) {
  if (a.length <= 1) return [a];
  return a.flatMap((x, i) => permutations([...a.slice(0, i), ...a.slice(i + 1)]).map((p) => [x, ...p]));
}

test('isSolvable stimmt für 2×2 mit der Breitensuche überein (alle 24 Anordnungen)', () => {
  const r = reachable(2);
  assert.equal(r.size, 12);
  for (const p of permutations([0, 1, 2, 3])) {
    assert.equal(P.isSolvable(p, 2), r.has(p.join(',')), p.join(','));
  }
});

test('isSolvable stimmt für 3×3 mit der Breitensuche überein', () => {
  const r = reachable(3);
  assert.equal(r.size, 181440); // 9!/2
  const rand = rng(7);
  for (let k = 0; k < 3000; k++) {
    const t = P.solved(3);
    for (let i = t.length - 1; i > 0; i--) { const j = Math.floor(rand() * (i + 1)); [t[i], t[j]] = [t[j], t[i]]; }
    assert.equal(P.isSolvable(t, 3), r.has(t.join(',')), t.join(','));
  }
});

test('isSolvable für gerade Breite 4: Zufallszüge lösbar, ein Tausch unlösbar', () => {
  const rand = rng(42);
  for (let k = 0; k < 200; k++) {
    let t = P.solved(4);
    for (let m = 0; m < 200; m++) {
      const next = P.slide(t, 4, Math.floor(rand() * 16));
      if (next) t = next;
    }
    assert.ok(P.isSolvable(t, 4), 'per Zügen erreicht → lösbar');
    const swapped = t.slice();
    const idx = [...swapped.keys()].filter((i) => swapped[i] !== 15).slice(0, 2);
    [swapped[idx[0]], swapped[idx[1]]] = [swapped[idx[1]], swapped[idx[0]]];
    assert.ok(!P.isSolvable(swapped, 4), 'Tausch zweier Teile → unlösbar');
  }
});

test('shuffleSolvable: lösbar, nicht gelöst, gültige Permutation (3–6)', () => {
  const rand = rng(1);
  for (const n of [3, 4, 5, 6]) {
    for (let k = 0; k < 300; k++) {
      const t = P.shuffleSolvable(n, rand);
      assert.equal(t.length, n * n);
      assert.deepEqual([...t].sort((a, b) => a - b), P.solved(n));
      assert.ok(P.isSolvable(t, n));
      assert.ok(!P.isSolved(t));
    }
  }
  const r3 = reachable(3);
  for (let k = 0; k < 300; k++) assert.ok(r3.has(P.shuffleSolvable(3, rand).join(',')));
});

test('slide: einzelne und mehrere Kacheln, ungültige Züge', () => {
  const s = P.solved(3); // Lücke unten rechts (Index 8)
  assert.deepEqual(P.slide(s, 3, 7), [0, 1, 2, 3, 4, 5, 6, 8, 7]);
  assert.deepEqual(P.slide(s, 3, 6), [0, 1, 2, 3, 4, 5, 8, 6, 7]); // zwei Kacheln rutschen mit
  assert.deepEqual(P.slide(s, 3, 2), [0, 1, 8, 3, 4, 2, 6, 7, 5]);
  assert.equal(P.slide(s, 3, 0), null); // diagonal
  assert.equal(P.slide(s, 3, 8), null); // Lücke selbst
  assert.ok(P.isSolved(P.slide(P.slide(s, 3, 7), 3, 8)));
});

test('swipeTarget', () => {
  const s = P.solved(3);
  assert.equal(P.swipeTarget(s, 3, 'right'), 7);
  assert.equal(P.swipeTarget(s, 3, 'down'), 5);
  assert.equal(P.swipeTarget(s, 3, 'left'), -1);
  assert.equal(P.swipeTarget(s, 3, 'up'), -1);
});

test('Jigsaw-Kanten passen zusammen, Rand ist gerade', () => {
  const rand = rng(3);
  for (const n of [3, 4, 5, 6]) {
    const e = P.makeEdges(n, rand);
    assert.ok(P.validEdges(e, n));
    for (let r = 0; r < n; r++) {
      for (let c = 0; c < n; c++) {
        const p = P.pieceEdges(e, n, r, c);
        if (r === 0) assert.equal(p.top, 0);
        if (c === 0) assert.equal(p.left, 0);
        if (r === n - 1) assert.equal(p.bottom, 0);
        if (c === n - 1) assert.equal(p.right, 0);
        if (c < n - 1) assert.equal(p.right, -P.pieceEdges(e, n, r, c + 1).left);
        if (r < n - 1) assert.equal(p.bottom, -P.pieceEdges(e, n, r + 1, c).top);
        if (r < n - 1) assert.ok(Math.abs(p.bottom) === 1);
      }
    }
  }
  assert.ok(!P.validEdges({ h: [[1]], v: [[1]] }, 1));
  assert.ok(!P.validEdges(null, 3));
});

test('snaps', () => {
  assert.ok(P.snaps(10, 10, 12, 13, 5));
  assert.ok(!P.snaps(10, 10, 20, 10, 5));
});
