// Tests für web/static/js/wrapped-logic.js – ausführen mit `make test` bzw. `node --test tests/js/`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';

const require = createRequire(import.meta.url);
const W = require('../../web/static/js/wrapped-logic.js');
const sp = (s) => s.split(String.fromCharCode(160)).join(' ').split(String.fromCharCode(0x202f)).join(' ');

const stats = JSON.parse(readFileSync(new URL('../../doors/assets-example/stats.json', import.meta.url)));
const ALL = ['intro', 'days', 'messages', 'emojis', 'goodnight', 'hours', 'streak', 'busiest', 'words', 'first', 'response', 'outro'];
const ids = (list) => list.map(s => s.id);

test('Zahlen, Daten, Plural', () => {
  assert.equal(W.fmtNum(1234567), '1.234.567');
  assert.equal(W.fmtNum(999), '999');
  assert.equal(W.fmtNum(0), '0');
  assert.equal(sp(W.count(1, 'Tag', 'Tage')), '1 Tag');
  assert.equal(sp(W.count(2, 'Tag', 'Tage')), '2 Tage');
  assert.equal(sp(W.count(1042, 'Tag', 'Tage')), '1.042 Tage');
  assert.equal(sp(W.fmtDate('2024-02-14')), '14. Februar 2024');
  assert.equal(sp(W.fmtRange('2026-07-16', '2026-08-26')), 'vom 16. Juli bis 26. August 2026');
  assert.equal(sp(W.fmtRange('2025-12-30', '2026-01-02')), 'vom 30. Dezember 2025 bis 2. Januar 2026');
  assert.equal(sp(W.fmtRange('2026-07-16', '2026-07-16')), 'am 16. Juli 2026');
  assert.equal(sp(W.pct(0.517)), '52 %');
  assert.equal(sp(W.minutesText(0.4)), 'unter 1 Minute');
  assert.equal(sp(W.minutesText(1)), '1 Minute');
  assert.equal(sp(W.minutesText(61)), '1 Stunde 1 Minute');
  assert.equal(sp(W.minutesText(178.5)), '2 Stunden 59 Minuten');
  assert.equal(sp(W.minutesText(120)), '2 Stunden');
});

test('buildSlides: Standardreihenfolge mit Beispiel-Stats, Texte pluralisiert', () => {
  const list = W.buildSlides({ title: 'Schatzo Wrapped 2026', outro_text: 'Danke', stats,
    slides: ['intro', 'days', 'messages', 'emojis', 'goodnight', 'hours', 'streak', 'busiest', 'words', 'first', 'outro'].map(id => ({ id })) });
  assert.deepEqual(ids(list), ['intro', 'days', 'messages', 'emojis', 'goodnight', 'hours', 'streak', 'busiest', 'words', 'first', 'outro']);
  const by = Object.fromEntries(list.map(s => [s.id, s]));
  assert.equal(by.intro.title, 'Schatzo Wrapped 2026');
  assert.equal(by.days.value, stats.daysTogether);
  assert.equal(by.days.unit, 'Tagen zusammen');
  assert.equal(by.messages.bars.length, 2);
  assert.equal(by.streak.unit, 'Tage am Stück');
  assert.match(sp(by.goodnight.text), /^An \d+ von \d+ Abenden war (Sandy|Jakob) zuerst\.$/);
  assert.equal(by.hours.hist.length, 24);
  assert.equal(by.hours.week.length, 7);
  assert.equal(by.outro.text, 'Danke');
  for (const s of list) assert.ok(s.ms >= 5000, s.id);
});

test('buildSlides: Singular, fehlende Daten, Überschreibungen', () => {
  const st = { since: '2026-12-21', daysTogether: 1, totalMessages: 2,
    persons: [{ name: 'Sandy', messages: 1, words: 1, share: 0.5 }, { name: 'Jakob', messages: 1, words: 0, share: 0.5, avgResponseMinutes: 1 }],
    topEmojis: [], topWords: [], goodNightFirst: [{ name: 'Sandy', days: 1 }, { name: 'Jakob', days: 0 }], goodNightNights: 1,
    longestStreak: { days: 1, from: '2026-12-21', to: '2026-12-21' }, activeDays: 1, busiestDay: { date: '2026-12-21', count: 1 },
    firstMessage: { date: '2026-12-21', time: '10:00', name: 'Sandy', text: 'Hi' } };
  const list = W.buildSlides({ stats: st, slides: ALL.map(id => ({ id })).concat([{ id: 'nope' }]) });
  assert.deepEqual(ids(list), ['intro', 'days', 'messages', 'goodnight', 'streak', 'busiest', 'first', 'response', 'outro'], 'ohne Emojis/Wörter/Stunden');
  const by = Object.fromEntries(list.map(s => [s.id, s]));
  assert.equal(by.days.unit, 'Tag zusammen');
  assert.equal(by.messages.unit, 'Nachrichten');
  assert.equal(by.messages.text, 'Exakt halbe-halbe. Wie es sich gehört.');
  assert.equal(sp(by.messages.extra), 'Zusammen 1 Wort.');
  assert.equal(by.streak.unit, 'Tag am Stück');
  assert.equal(sp(by.streak.text), 'Am 21. Dezember 2026.');
  assert.equal(sp(by.streak.extra), 'Insgesamt haben wir an 1 Tag geschrieben.');
  assert.equal(by.busiest.unit, 'Nachricht an einem Tag');
  assert.equal(sp(by.goodnight.text), 'An 1 von 1 Abend war Sandy zuerst.');
  assert.equal(sp(by.first.text), 'Danach kam noch 1 Nachricht.');
  assert.equal(by.response.winner, 'Jakob');
  const over = W.buildSlides({ stats: st, slides: [{ id: 'intro' }, { id: 'days', title: 'So lange schon', text: 'Und noch viel länger.' }, { id: 'outro' }] });
  assert.equal(over[1].title, 'So lange schon');
  assert.equal(over[1].text, 'Und noch viel länger.');
  // tie in good nights
  st.goodNightFirst = [{ name: 'Sandy', days: 3 }, { name: 'Jakob', days: 3 }];
  st.goodNightNights = 6;
  const gn = W.buildSlides({ stats: st, slides: [{ id: 'goodnight' }] })[0];
  assert.equal(gn.winner, 'Unentschieden!');
});

test('player: Auto-Weiter, Pause (mehrere Gründe), vor/zurück, Ende', () => {
  const p = W.player([1000, 2000, 500]);
  assert.equal(p.tick(400), null);
  assert.equal(p.progress(0), 0.4);
  p.pause('hold');
  p.pause('hidden');
  assert.equal(p.tick(5000), null, 'pausiert läuft nichts');
  p.resume('hold');
  assert.ok(p.isPaused(), 'noch hidden');
  p.resume('hidden');
  assert.equal(p.tick(600), 'advance');
  assert.equal(p.index, 1);
  assert.equal(p.progress(0), 1);
  assert.equal(p.progress(2), 0);
  assert.equal(p.tick(1999), null);
  assert.equal(p.tick(1), 'advance');
  assert.equal(p.tick(500), 'end');
  assert.ok(p.ended);
  assert.equal(p.tick(1000), null, 'nach dem Ende passiert nichts mehr');
  assert.equal(p.progress(2), 1);
  assert.equal(p.next(), false);
  p.prev();
  assert.equal(p.index, 1);
  assert.equal(p.ended, false);
  p.goTo(0);
  p.prev();
  assert.equal(p.index, 0, 'zurück auf der ersten Slide bleibt dort');
  assert.equal(p.toggle('key'), true);
  assert.equal(p.toggle('key'), false);
  p.tick(0); p.tick(-5); p.tick(NaN);
  assert.equal(p.elapsed, 0);
});

test('gesture: Tippen links/rechts, Wischen, Halten, Runterwischen', () => {
  const g = (o) => W.gesture(Object.assign({ dx: 0, dy: 0, ms: 80, x: 300, width: 390 }, o));
  assert.equal(g({}), 'next');
  assert.equal(g({ x: 60 }), 'prev');
  assert.equal(g({ ms: 600 }), 'hold');
  assert.equal(g({ dx: -120, dy: 10 }), 'next');
  assert.equal(g({ dx: 120, dy: -20 }), 'prev');
  assert.equal(g({ dy: 160, dx: 10 }), 'close');
  assert.equal(g({ dx: 30, dy: 30 }), null);
});
