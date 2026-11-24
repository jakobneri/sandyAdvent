/* Gemeinsame Bausteine der Spiele-Türchen (quiz, memory, wordle, puzzle):
   DOM-Helfer, Fortschritt in localStorage (ohne Speicher lauffähig),
   Pluralisierung, Konfetti/Schnee-Burst, Gewinn-Panel mit „Nochmal spielen“.
   Vanilla ES5-kompatibel, keine Abhängigkeiten. */
(function (w) {
  'use strict';
  var doc = document;
  var reduceMotion = false;
  try { reduceMotion = w.matchMedia('(prefers-reduced-motion: reduce)').matches; } catch (e) {}

  /* ---------- DOM ---------- */
  // h('button', {class: 'x', text: 'Hallo', onclick: fn, 'aria-label': '…'}, [kinder])
  function h(tag, attrs, kids) {
    var el = doc.createElement(tag);
    var k;
    if (attrs) {
      for (k in attrs) {
        if (!attrs.hasOwnProperty(k) || attrs[k] === null || attrs[k] === undefined || attrs[k] === false) continue;
        var v = attrs[k];
        if (k === 'class') el.className = v;
        else if (k === 'text') el.textContent = v;
        else if (k.slice(0, 2) === 'on' && typeof v === 'function') el.addEventListener(k.slice(2), v, false);
        else if (k === 'hidden') el.hidden = true;
        else if (k === 'disabled') el.disabled = true;
        else el.setAttribute(k, v === true ? '' : String(v));
      }
    }
    if (kids) {
      if (!(kids instanceof Array)) kids = [kids];
      for (var i = 0; i < kids.length; i++) {
        if (kids[i] === null || kids[i] === undefined || kids[i] === false) continue;
        el.appendChild(typeof kids[i] === 'string' ? doc.createTextNode(kids[i]) : kids[i]);
      }
    }
    return el;
  }
  function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); }
  function hasClass(el, c) { return (' ' + el.className + ' ').indexOf(' ' + c + ' ') >= 0; }
  function addClass(el, c) { if (!hasClass(el, c)) el.className = (el.className ? el.className + ' ' : '') + c; }
  function removeClass(el, c) {
    el.className = (' ' + el.className + ' ').replace(' ' + c + ' ', ' ').replace(/^\s+|\s+$/g, '');
  }
  function toggleClass(el, c, on) { if (on) addClass(el, c); else removeClass(el, c); }
  // Animation neu starten (z. B. Wackeln bei falscher Eingabe).
  function replayClass(el, c) {
    removeClass(el, c);
    void el.offsetWidth;
    addClass(el, c);
  }

  // Text mit Leerzeile = Absatz, Zeilenumbruch = <br> (wie im media-Template).
  function paragraphs(text) {
    var out = [];
    if (!text) return out;
    var parts = String(text).replace(/\r\n/g, '\n').split(/\n\s*\n/);
    for (var i = 0; i < parts.length; i++) {
      var p = parts[i].replace(/^\s+|\s+$/g, '');
      if (!p) continue;
      var el = doc.createElement('p');
      var lines = p.split('\n');
      for (var j = 0; j < lines.length; j++) {
        if (j) el.appendChild(doc.createElement('br'));
        el.appendChild(doc.createTextNode(lines[j]));
      }
      out.push(el);
    }
    return out;
  }

  /* ---------- Sprache ---------- */
  function plural(n, one, many) { return (n === 1 || n === -1) ? one : many; }
  function count(n, one, many) { return n + ' ' + plural(n, one, many); }
  function fmtTime(ms) {
    var s = Math.max(0, Math.floor(ms / 1000));
    var m = Math.floor(s / 60);
    s = s % 60;
    return m + ':' + (s < 10 ? '0' : '') + s;
  }
  // „1 Minute 5 Sekunden“ – für Screenreader und das Gewinn-Panel.
  function durationText(ms) {
    var s = Math.max(0, Math.round(ms / 1000));
    var m = Math.floor(s / 60);
    s = s % 60;
    if (!m) return count(s, 'Sekunde', 'Sekunden');
    return count(m, 'Minute', 'Minuten') + (s ? ' ' + count(s, 'Sekunde', 'Sekunden') : '');
  }

  /* ---------- Zufall ---------- */
  function shuffle(a, rnd) {
    rnd = rnd || Math.random;
    for (var i = a.length - 1; i > 0; i--) {
      var j = Math.floor(rnd() * (i + 1));
      var t = a[i]; a[i] = a[j]; a[j] = t;
    }
    return a;
  }
  function range(n) { var a = []; for (var i = 0; i < n; i++) a.push(i); return a; }

  /* ---------- Fortschritt (localStorage, optional) ---------- */
  function store(key) {
    return {
      load: function () {
        try {
          var raw = w.localStorage.getItem(key);
          return raw ? JSON.parse(raw) : null;
        } catch (e) { return null; }
      },
      save: function (obj) {
        try { w.localStorage.setItem(key, JSON.stringify(obj)); } catch (e) {}
      },
      clear: function () {
        try { w.localStorage.removeItem(key); } catch (e) {}
      }
    };
  }

  function haptic(ms) {
    if (navigator.vibrate) { try { navigator.vibrate(ms || 15); } catch (e) {} }
  }

  /* ---------- Konfetti + Schnee ---------- */
  var COLORS = ['#f0c66e', '#e3ad4c', '#c23447', '#a11f36', '#fffaf1', '#2a6553', '#f9e3b0', '#ff8a3d'];
  function celebrate(anchor, big) {
    var target = anchor && anchor.getBoundingClientRect ? anchor : null;
    if (reduceMotion || !w.requestAnimationFrame) {
      if (target) replayClass(target, 'is-glowing');
      return;
    }
    var c = doc.createElement('canvas');
    var ctx = c.getContext && c.getContext('2d');
    if (!ctx) return;
    var W = w.innerWidth, H = w.innerHeight;
    var dpr = Math.min(w.devicePixelRatio || 1, 2);
    c.width = Math.round(W * dpr);
    c.height = Math.round(H * dpr);
    c.className = 'confetti';
    c.setAttribute('aria-hidden', 'true');
    doc.body.appendChild(c);
    ctx.scale(dpr, dpr);
    var ox = W / 2, oy = H * 0.42;
    if (target) {
      var r = target.getBoundingClientRect();
      ox = Math.min(Math.max(r.left + r.width / 2, 40), W - 40);
      oy = Math.min(Math.max(r.top + Math.min(r.height / 2, 120), 60), H * 0.7);
    }
    var parts = [];
    var n = big ? 170 : 95;
    for (var i = 0; i < n; i++) {
      var a = Math.random() * Math.PI * 2;
      var sp = 4 + Math.random() * (big ? 9 : 7);
      var flake = Math.random() < 0.28;
      parts.push({
        x: ox, y: oy,
        vx: Math.cos(a) * sp, vy: Math.sin(a) * sp - (big ? 6 : 4),
        r: Math.random() * Math.PI, vr: (Math.random() - 0.5) * 0.35,
        w: flake ? 2 + Math.random() * 2.5 : 5 + Math.random() * 5,
        h: flake ? 0 : 3 + Math.random() * 4,
        flake: flake,
        color: flake ? '#ffffff' : COLORS[i % COLORS.length],
        wob: Math.random() * 10
      });
    }
    var start = 0, last = 0, DUR = big ? 3200 : 2500;
    function frame(t) {
      if (!start) { start = t; last = t; }
      var dt = Math.min((t - last) / 16.7, 3);
      last = t;
      var el = t - start;
      ctx.clearRect(0, 0, W, H);
      var alpha = el > DUR - 700 ? Math.max(0, (DUR - el) / 700) : 1;
      for (var i = 0; i < parts.length; i++) {
        var p = parts[i];
        p.vx *= Math.pow(0.985, dt);
        p.vy = p.vy * Math.pow(0.985, dt) + (p.flake ? 0.08 : 0.22) * dt;
        if (p.flake && p.vy > 1.6) p.vy = 1.6;
        p.wob += 0.08 * dt;
        p.x += (p.vx + (p.flake ? Math.sin(p.wob) * 0.6 : 0)) * dt;
        p.y += p.vy * dt;
        p.r += p.vr * dt;
        ctx.globalAlpha = alpha;
        ctx.fillStyle = p.color;
        if (p.flake) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, p.w, 0, Math.PI * 2);
          ctx.fill();
        } else {
          ctx.save();
          ctx.translate(p.x, p.y);
          ctx.rotate(p.r);
          ctx.scale(1, Math.abs(Math.cos(p.wob)) * 0.8 + 0.2);
          ctx.fillRect(-p.w / 2, -p.h / 2, p.w, p.h);
          ctx.restore();
        }
      }
      if (el < DUR) w.requestAnimationFrame(frame);
      else if (c.parentNode) c.parentNode.removeChild(c);
    }
    w.requestAnimationFrame(frame);
  }

  /* ---------- Buttons + Gewinn-Panel ---------- */
  function button(text, opts) {
    opts = opts || {};
    return h('button', {
      type: 'button',
      class: 'btn' + (opts.variant ? ' btn--' + opts.variant : ''),
      onclick: opts.onclick,
      'aria-label': opts.label
    }, text);
  }

  var STAR = '<svg viewBox="0 0 48 48" aria-hidden="true" focusable="false"><path d="M24 3.5l6.2 12.6 13.9 2-10 9.8 2.4 13.8L24 35.2l-12.5 6.5 2.4-13.8-10-9.8 13.9-2z"/></svg>';

  // winPanel({title, stats: ['12 Züge', …], text, extra: Element, onReplay})
  function winPanel(o) {
    var badge = h('div', { class: 'win__badge', 'aria-hidden': 'true' });
    badge.innerHTML = STAR; // statisches SVG
    var stats = null;
    if (o.stats && o.stats.length) {
      stats = h('p', { class: 'win__stats' });
      for (var i = 0; i < o.stats.length; i++) {
        stats.appendChild(h('span', { class: 'win__stat' }, o.stats[i]));
      }
    }
    var text = o.text ? h('div', { class: 'win__text' }, paragraphs(o.text)) : null;
    var replay = o.onReplay ? button(o.replayLabel || 'Nochmal spielen', { variant: 'ghost', onclick: o.onReplay }) : null;
    var panel = h('section', { class: 'win', role: 'status', tabindex: '-1' }, [
      badge,
      h('p', { class: 'win__kicker', text: o.kicker || 'Türchen geknackt' }),
      h('h2', { class: 'win__title', text: o.title || 'Geschafft!' }),
      stats,
      o.extra || null,
      text,
      replay ? h('div', { class: 'win__actions' }, replay) : null
    ]);
    return panel;
  }

  function scrollIntoView(el) {
    if (!el || !el.getBoundingClientRect) return;
    var r = el.getBoundingClientRect();
    if (r.top >= 0 && r.top < w.innerHeight * 0.6) return;
    try { el.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'start' }); } catch (e) { el.scrollIntoView(); }
  }

  /* ---------- Start ---------- */
  // SAGame.start('memory', function (ctx) { … }) – ctx: root, data, store, section
  function start(kind, init) {
    function run() {
      var section = doc.querySelector('[data-game="' + kind + '"]');
      var dataEl = doc.getElementById('game-data');
      if (!section || !dataEl) return;
      var root = section.querySelector('[data-game-root]');
      var data;
      try { data = JSON.parse(dataEl.textContent); } catch (e) { data = null; }
      clear(root);
      if (!data) {
        root.appendChild(h('p', { class: 'game__error', text: 'Das Spiel konnte nicht geladen werden.' }));
        return;
      }
      try {
        init({ root: root, data: data, store: store(section.getAttribute('data-key') || 'sa-game-' + kind), section: section });
      } catch (err) {
        clear(root);
        root.appendChild(h('p', { class: 'game__error', text: 'Ups – hier hakt gerade etwas. Lade die Seite bitte neu.' }));
        if (w.console && console.error) console.error(err);
      }
    }
    if (doc.readyState === 'loading') doc.addEventListener('DOMContentLoaded', run, false);
    else run();
  }

  w.SAGame = {
    h: h, clear: clear, addClass: addClass, removeClass: removeClass, hasClass: hasClass,
    toggleClass: toggleClass, replayClass: replayClass, paragraphs: paragraphs,
    plural: plural, count: count, fmtTime: fmtTime, durationText: durationText,
    shuffle: shuffle, range: range, store: store, haptic: haptic, celebrate: celebrate,
    button: button, winPanel: winPanel, scrollIntoView: scrollIntoView, start: start,
    reduceMotion: reduceMotion
  };
})(window);
