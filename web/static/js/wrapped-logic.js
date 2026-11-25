/* „Schatzo Wrapped“-Logik ohne DOM: deutsche Zahlen/Daten, Pluralisierung,
   Slide-Liste aus stats.json (+ Überschreibungen aus der Türchen-Config) und
   der Abspieler (Auto-Weiter, Pause aus mehreren Gründen, vor/zurück).
   Läuft im Browser (window.WrappedLogic) und in Node (tests/js/wrapped-logic.test.mjs). */
(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.WrappedLogic = api;
})(this, function () {
  'use strict';
  var NBSP = String.fromCharCode(160);
  var NNBSP = String.fromCharCode(0x202F);
  var MONTHS = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
  var WEEKDAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];
  var DEFAULT_MS = 6000;
  var DURATION = { intro: 5000, words: 7000, hours: 7000, messages: 6500 };

  /* ---------- Sprache ---------- */
  function plural(n, one, many) { return (n === 1 || n === -1) ? one : many; }
  function fmtNum(n) {
    n = Math.round(Number(n) || 0);
    var neg = n < 0, s = String(Math.abs(n)), out = '';
    while (s.length > 3) { out = '.' + s.slice(-3) + out; s = s.slice(0, -3); }
    return (neg ? '-' : '') + s + out;
  }
  function count(n, one, many) { return fmtNum(n) + NBSP + plural(n, one, many); }
  function parseISO(iso) {
    var m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(iso || ''));
    return m ? { y: +m[1], m: +m[2], d: +m[3] } : null;
  }
  function fmtDate(iso, noYear) {
    var p = parseISO(iso);
    if (!p) return '';
    return p.d + '.' + NBSP + MONTHS[p.m - 1] + (noYear ? '' : ' ' + p.y);
  }
  // „vom 16. Juli bis 26. August 2026“ (Jahr nur einmal, wenn gleich)
  function fmtRange(from, to) {
    var a = parseISO(from), b = parseISO(to);
    if (!a || !b) return '';
    if (from === to) return 'am ' + fmtDate(to);
    return 'vom ' + fmtDate(from, a.y === b.y) + ' bis ' + fmtDate(to);
  }
  function pct(share) { return Math.round((Number(share) || 0) * 100) + NNBSP + '%'; }
  function hourLabel(h) { return h + ' Uhr'; }
  function minutesText(min) {
    min = Number(min) || 0;
    if (min < 1) return 'unter 1' + NBSP + 'Minute';
    var m = Math.round(min);
    if (m < 60) return count(m, 'Minute', 'Minuten');
    var hrs = Math.floor(m / 60), rest = m % 60;
    return count(hrs, 'Stunde', 'Stunden') + (rest ? ' ' + count(rest, 'Minute', 'Minuten') : '');
  }

  /* ---------- Slides ---------- */
  function persons(st) { return (st && st.persons) || []; }

  // Jede Slide: {id, kicker, title, text, ms, …Daten für die Darstellung}.
  // Fehlen Daten, fällt die Slide weg (außer intro/outro).
  var BUILD = {
    intro: function (st, d) {
      return { kicker: 'Unser Jahr im Rückblick', title: d.title || 'Schatzo Wrapped',
        text: 'Unser Jahr in Nachrichten. Tippe rechts für weiter, links für zurück – gedrückt halten pausiert.' };
    },
    days: function (st) {
      if (!st.since || !(st.daysTogether > 0)) return null;
      var n = st.daysTogether;
      return { kicker: 'Tage zusammen', title: 'Wir sind seit', value: n, unit: plural(n, 'Tag', 'Tagen') + ' zusammen',
        text: 'Angefangen hat alles am ' + fmtDate(st.since) + '.' };
    },
    messages: function (st) {
      var ps = persons(st);
      if (!(st.totalMessages > 0) || !ps.length) return null;
      var words = 0;
      for (var i = 0; i < ps.length; i++) words += ps[i].words || 0;
      var text;
      if (ps.length === 2 && ps[0].messages === ps[1].messages) text = 'Exakt halbe-halbe. Wie es sich gehört.';
      else text = ps[0].name + ' schreibt ' + pct(ps[0].share) + ' davon.';
      return { kicker: 'Nachrichten', title: 'So viel haben wir uns geschrieben', value: st.totalMessages,
        unit: plural(st.totalMessages, 'Nachricht', 'Nachrichten'), bars: ps.map(function (p) {
          return { label: p.name, value: p.messages, note: fmtNum(p.messages) + ' · ' + pct(p.share) };
        }), text: text, extra: words ? 'Zusammen ' + count(words, 'Wort', 'Wörter') + '.' : '' };
    },
    emojis: function (st) {
      var e = st.topEmojis || [];
      if (!e.length) return null;
      return { kicker: 'Top-Emojis', title: 'Unsere Lieblings-Emojis', emojis: e.slice(0, 8),
        text: e[0].emoji + ' kam ' + fmtNum(e[0].count) + '-mal vor.' };
    },
    goodnight: function (st) {
      var g = st.goodNightFirst || [], nights = st.goodNightNights || 0;
      if (!g.length || !nights) return null;
      var tie = g.length > 1 && g[0].days === g[1].days;
      return { kicker: 'Gute Nacht', title: 'Wer sagt öfter zuerst Gute Nacht?', winner: tie ? 'Unentschieden!' : g[0].name,
        bars: g.map(function (p) { return { label: p.name, value: p.days, note: fmtNum(p.days) }; }),
        text: tie ? 'Ihr seid euch einig – ' + g[0].days + ':' + g[1].days + '.'
          : 'An ' + fmtNum(g[0].days) + ' von ' + count(nights, 'Abend', 'Abenden') + ' war ' + g[0].name + ' zuerst.' };
    },
    hours: function (st) {
      var h = st.mostActiveHour, wd = st.mostActiveWeekday;
      if (!h || !h.histogram || h.histogram.length !== 24 || !h.count) return null;
      var hr = h.hour, mood;
      if (hr >= 21 || hr < 4) mood = 'Echte Nachteulen. 🦉';
      else if (hr >= 4 && hr < 10) mood = 'Frühaufsteher! ☀️';
      else if (hr >= 11 && hr < 14) mood = 'Mittagspausen-Liebe. 🥪';
      else mood = 'Mitten am Tag. ☕';
      var out = { kicker: 'Lieblingszeit', title: 'Am meisten schreiben wir um', big: hourLabel(hr), hist: h.histogram, peak: hr, text: mood };
      if (wd && wd.histogram && wd.histogram.length === 7 && wd.count) {
        out.weekday = wd.weekday;
        out.week = wd.histogram;
        out.weekPeak = wd.index;
        out.weekLabels = WEEKDAYS_SHORT;
      }
      return out;
    },
    streak: function (st) {
      var s = st.longestStreak;
      if (!s || !(s.days > 0)) return null;
      return { kicker: 'Längste Serie', title: 'Kein Tag ohne Nachricht', value: s.days, unit: plural(s.days, 'Tag', 'Tage') + ' am Stück',
        text: s.from ? fmtRange(s.from, s.to).replace(/^./, function (c) { return c.toUpperCase(); }) + '.' : '',
        extra: st.activeDays ? 'Insgesamt haben wir an ' + count(st.activeDays, 'Tag', 'Tagen') + ' geschrieben.' : '' };
    },
    busiest: function (st) {
      var b = st.busiestDay;
      if (!b || !(b.count > 0) || !b.date) return null;
      return { kicker: 'Rekordtag', title: fmtDate(b.date), value: b.count, unit: plural(b.count, 'Nachricht', 'Nachrichten') + ' an einem Tag',
        text: 'Was war da bloß los? 😄' };
    },
    words: function (st) {
      var w = st.topWords || [];
      if (!w.length) return null;
      return { kicker: 'Top-Wörter', title: 'Unsere Wörter', words: w.slice(0, 12), text: '„' + w[0].word + '“ – ' + fmtNum(w[0].count) + '-mal.' };
    },
    first: function (st) {
      var f = st.firstMessage;
      if (!f || !f.text) return null;
      var rest = (st.totalMessages || 1) - 1;
      return { kicker: 'Ganz am Anfang', title: 'Die erste Nachricht', quote: f.text, meta: f.name + ' · ' + fmtDate(f.date) + (f.time ? ', ' + f.time + ' Uhr' : ''),
        text: rest === 1 ? 'Danach kam noch 1' + NBSP + 'Nachricht.' : rest > 1 ? 'Danach kamen noch ' + count(rest, 'Nachricht', 'Nachrichten') + '.' : '' };
    },
    response: function (st) {
      var ps = persons(st), rows = [];
      for (var i = 0; i < ps.length; i++) if (typeof ps[i].avgResponseMinutes === 'number') rows.push(ps[i]);
      if (!rows.length) return null;
      rows.sort(function (a, b) { return a.avgResponseMinutes - b.avgResponseMinutes; });
      return { kicker: 'Antwortzeit', title: 'Wer antwortet schneller?', winner: rows[0].name,
        bars: rows.map(function (p) { return { label: p.name, value: p.avgResponseMinutes, note: 'Ø ' + minutesText(p.avgResponseMinutes) }; }),
        invert: true, text: 'Im Schnitt nach ' + minutesText(rows[0].avgResponseMinutes) + '.' };
    },
    outro: function (st, d) {
      return { kicker: 'Frohe Weihnachten', title: 'Danke für dieses Jahr', text: d.outro_text || 'Auf noch ganz viele Nachrichten mehr!' };
    }
  };

  function buildSlides(data) {
    var st = data.stats || {}, list = data.slides || [], out = [];
    for (var i = 0; i < list.length; i++) {
      var o = list[i], fn = BUILD[o.id];
      if (!fn) continue;
      var s = fn(st, data);
      if (!s) continue;
      s.id = o.id;
      if (o.title) s.title = o.title;
      if (o.text) { s.text = o.text; s.extra = ''; }
      s.ms = DURATION[o.id] || DEFAULT_MS;
      out.push(s);
    }
    return out;
  }

  /* ---------- Abspieler ---------- */
  // tick(dt) liefert 'advance' (nächste Slide), 'end' (letzte Slide abgelaufen) oder null.
  function player(durations) {
    var p = { index: 0, elapsed: 0, ended: false, reasons: {} };
    var n = durations.length;
    function dur(i) { return durations[i] || DEFAULT_MS; }
    p.isPaused = function () { for (var k in p.reasons) if (p.reasons.hasOwnProperty(k) && p.reasons[k]) return true; return false; };
    p.pause = function (why) { p.reasons[why || 'user'] = true; };
    p.resume = function (why) { delete p.reasons[why || 'user']; };
    p.toggle = function (why) { if (p.reasons[why || 'user']) p.resume(why); else p.pause(why); return p.isPaused(); };
    p.goTo = function (i) {
      p.index = Math.max(0, Math.min(n - 1, i));
      p.elapsed = 0;
      p.ended = false;
      return p.index;
    };
    p.next = function () {
      if (p.index < n - 1) { p.goTo(p.index + 1); return true; }
      p.elapsed = dur(p.index);
      p.ended = true;
      return false;
    };
    p.prev = function () {
      if (p.index > 0) { p.goTo(p.index - 1); return true; }
      p.goTo(0); // erste Slide: von vorn
      return false;
    };
    p.tick = function (dt) {
      if (p.ended || p.isPaused() || !(dt > 0)) return null;
      p.elapsed += dt;
      if (p.elapsed < dur(p.index)) return null;
      if (p.index < n - 1) { p.goTo(p.index + 1); return 'advance'; }
      p.elapsed = dur(p.index);
      p.ended = true;
      return 'end';
    };
    p.progress = function (i) {
      if (i < p.index) return 1;
      if (i > p.index) return 0;
      return Math.min(1, p.elapsed / dur(i));
    };
    return p;
  }

  // Tippen/Wischen auswerten: {dx, dy, ms, x, width} → 'next' | 'prev' | 'close' | 'hold' | null
  function gesture(g) {
    var adx = Math.abs(g.dx), ady = Math.abs(g.dy);
    if (adx > 50 && adx > ady * 1.2) return g.dx < 0 ? 'next' : 'prev';
    if (g.dy > 90 && ady > adx * 1.2) return 'close';
    if (adx > 15 || ady > 15) return null;
    if (g.ms > 250) return 'hold';
    return g.x < g.width * 0.33 ? 'prev' : 'next';
  }

  return {
    plural: plural, count: count, fmtNum: fmtNum, fmtDate: fmtDate, fmtRange: fmtRange, pct: pct,
    hourLabel: hourLabel, minutesText: minutesText, buildSlides: buildSlides, player: player, gesture: gesture,
    DEFAULT_MS: DEFAULT_MS, WEEKDAYS_SHORT: WEEKDAYS_SHORT
  };
});
