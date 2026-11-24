/* Countdown bis zum Öffnen. Nutzt die Serverzeit (ADVENT_NOW im Dev-Modus)
   über einen Offset zur Geräteuhr und lädt bei 0 neu. */
(function () {
  'use strict';
  var doc = document;
  var dataEl = doc.getElementById('cd-data');
  if (!dataEl) return;
  var data;
  try { data = JSON.parse(dataEl.textContent); } catch (e) { return; }
  var offset = data.now - Date.now();
  var els = {
    d: doc.querySelector('[data-d]'), h: doc.querySelector('[data-h]'),
    m: doc.querySelector('[data-m]'), s: doc.querySelector('[data-s]'),
    dl: doc.querySelector('[data-dl]'), hl: doc.querySelector('[data-hl]'),
    ml: doc.querySelector('[data-ml]'), sl: doc.querySelector('[data-sl]')
  };
  var reloading = false;
  var timer = 0; // genau eine laufende Schleife (visibilitychange ruft tick() erneut)
  function pad(n) { return (n < 10 ? '0' : '') + n; }
  function set(el, v) { if (el && el.textContent !== v) el.textContent = v; }
  // „1 Tag“, „2 Tage“, „0 Stunden“ … (gleiche Regel wie advent.Plural in Go)
  function word(n, one, many) { return n === 1 ? one : many; }
  function show(d, h, m, s) {
    set(els.d, String(d)); set(els.h, pad(h)); set(els.m, pad(m)); set(els.s, pad(s));
    set(els.dl, word(d, 'Tag', 'Tage')); set(els.hl, word(h, 'Stunde', 'Stunden'));
    set(els.ml, word(m, 'Minute', 'Minuten')); set(els.sl, word(s, 'Sekunde', 'Sekunden'));
  }

  function tick() {
    clearTimeout(timer);
    timer = 0;
    var rem = data.target - (Date.now() + offset);
    if (rem <= 0) {
      show(0, 0, 0, 0);
      if (data.reload && !reloading) {
        reloading = true;
        setTimeout(function () { window.location.reload(); }, 700);
      }
      return;
    }
    var secs = Math.floor(rem / 1000);
    show(Math.floor(secs / 86400), Math.floor(secs % 86400 / 3600), Math.floor(secs % 3600 / 60), secs % 60);
    timer = setTimeout(tick, (rem % 1000) + 20);
  }
  tick();
  // Nach dem Aufwachen des Handys sofort neu rechnen.
  doc.addEventListener('visibilitychange', function () { if (!doc.hidden) tick(); });

  /* Am Türchen rütteln: wackelt und zeigt einen neuen Spruch. */
  var door = doc.querySelector('[data-door]');
  var line = doc.querySelector('[data-line]');
  var lines = data.lines || [];
  if (door) {
    door.addEventListener('click', function () {
      door.className = door.className.replace(/\s*\bis-shaking\b/, '');
      void door.offsetWidth; // Animation neu starten
      door.className += ' is-shaking';
      if (navigator.vibrate) { try { navigator.vibrate(40); } catch (e) {} }
      if (line && lines.length > 1) {
        var current = line.textContent;
        var next = current;
        for (var i = 0; i < 6 && next === current; i++) {
          next = '„' + lines[Math.floor(Math.random() * lines.length)] + '“';
        }
        line.className += ' is-swapping';
        setTimeout(function () {
          line.textContent = next;
          line.className = line.className.replace(/\s*\bis-swapping\b/, '');
        }, 220);
      }
    });
    door.addEventListener('animationend', function () {
      door.className = door.className.replace(/\s*\bis-shaking\b/, '');
    });
  }
})();
