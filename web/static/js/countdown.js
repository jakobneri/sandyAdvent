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
    days: doc.querySelector('[data-days]'), dl: doc.querySelector('[data-dl]')
  };
  var reloading = false;
  var timer = 0; // genau eine laufende Schleife (visibilitychange ruft tick() erneut)
  function pad(n) { return (n < 10 ? '0' : '') + n; }
  function set(el, v) { if (el && el.textContent !== v) el.textContent = v; }
  // „1 Tag“ / „15 Tage“; bei 0 Tagen nur noch die Uhrzeit.
  function setDays(n) {
    set(els.d, String(n));
    set(els.dl, n === 1 ? 'Tag' : 'Tage');
    if (els.days && els.days.hidden !== (n === 0)) els.days.hidden = n === 0;
  }

  function tick() {
    clearTimeout(timer);
    timer = 0;
    var rem = data.target - (Date.now() + offset);
    if (rem <= 0) {
      setDays(0); set(els.h, '00'); set(els.m, '00'); set(els.s, '00');
      if (data.reload && !reloading) {
        reloading = true;
        setTimeout(function () { window.location.reload(); }, 700);
      }
      return;
    }
    var secs = Math.floor(rem / 1000);
    setDays(Math.floor(secs / 86400));
    set(els.h, pad(Math.floor(secs % 86400 / 3600)));
    set(els.m, pad(Math.floor(secs % 3600 / 60)));
    set(els.s, pad(secs % 60));
    timer = setTimeout(tick, (rem % 1000) + 20);
  }
  tick();
  // Nach dem Aufwachen des Handys sofort neu rechnen.
  doc.addEventListener('visibilitychange', function () { if (!doc.hidden) tick(); });

  /* Am Türchen rütteln: wackelt, der Anhänger schwingt, neuer Spruch. */
  var door = doc.querySelector('[data-door]');
  var tag = doc.querySelector('[data-tag]');
  var line = doc.querySelector('[data-line]');
  var lines = data.lines || [];
  function drop(el, cls) { el.className = el.className.replace(new RegExp('\\s*\\b' + cls + '\\b', 'g'), ''); }
  function restart(el, cls) {
    drop(el, cls);
    void el.offsetWidth; // Animation neu starten
    el.className += ' ' + cls;
  }
  if (tag) tag.addEventListener('animationend', function (e) { if (e.target === tag) drop(tag, 'is-swinging'); });
  if (door) {
    door.addEventListener('click', function () {
      restart(door, 'is-shaking');
      if (tag) restart(tag, 'is-swinging');
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
    door.addEventListener('animationend', function (e) { if (e.target === door) drop(door, 'is-shaking'); });
  }
})();
