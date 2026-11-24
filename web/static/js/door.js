/* Türchen-Seite: Öffnungsanimation, Audio-Player, Slideshows.
   Vanilla ES5-kompatibel (ältere iOS-Safaris), keine Abhängigkeiten. */
(function () {
  'use strict';
  var doc = document;
  var root = doc.documentElement;
  var reduceMotion = false;
  try { reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches; } catch (e) {}

  function each(list, fn) { for (var i = 0; i < list.length; i++) fn(list[i], i); }
  function on(el, ev, fn, opts) { el.addEventListener(ev, fn, opts || false); }

  /* ---------- Öffnungsanimation ---------- */
  function initIntro() {
    var day = root.getAttribute('data-day');
    try { window.localStorage.setItem('sa-door-' + day, '1'); } catch (e) {}
    if (!/\bintro\b/.test(root.className)) return;
    var overlay = doc.querySelector('[data-intro]');
    var done = false;
    function finish() {
      if (done) return;
      done = true;
      root.className = root.className.replace(/\s*\bintro\b/, '');
    }
    // CSS spielt die Animation allein ab; JS räumt nur auf und erlaubt Überspringen.
    setTimeout(finish, 2800);
    if (overlay) {
      on(overlay, 'click', finish);
      on(overlay, 'animationend', function (e) { if (e.target === overlay) finish(); });
    }
  }

  /* ---------- Audio-Player ---------- */
  function fmt(t) {
    if (!isFinite(t) || t < 0) return '–:––';
    t = Math.floor(t);
    var m = Math.floor(t / 60), s = t % 60;
    return m + ':' + (s < 10 ? '0' : '') + s;
  }

  function initPlayer(el) {
    var audio = el.querySelector('audio');
    var ui = el.querySelector('.player__ui');
    var btn = el.querySelector('[data-play]');
    var seek = el.querySelector('[data-seek]');
    var cur = el.querySelector('[data-cur]');
    var dur = el.querySelector('[data-dur]');
    var err = el.querySelector('[data-error]');
    if (!audio || !ui || !btn || !seek) return;
    audio.removeAttribute('controls');
    audio.hidden = true;
    ui.hidden = false;
    var dragging = false;

    function setProgress(frac) {
      if (!isFinite(frac) || frac < 0) frac = 0;
      if (frac > 1) frac = 1;
      seek.value = String(Math.round(frac * 1000));
      seek.style.setProperty('--p', (frac * 100).toFixed(2) + '%');
    }
    function update() {
      if (!dragging) setProgress(audio.currentTime / audio.duration);
      cur.textContent = fmt(audio.currentTime);
    }
    function setPlaying(p) {
      el.className = el.className.replace(/\s*\bis-playing\b/, '') + (p ? ' is-playing' : '');
      btn.setAttribute('aria-label', p ? 'Pause' : 'Abspielen');
    }
    on(btn, 'click', function () {
      if (audio.paused || audio.ended) {
        var pr = audio.play();
        if (pr && pr.catch) pr.catch(function () { setPlaying(false); });
      } else {
        audio.pause();
      }
    });
    on(audio, 'play', function () { setPlaying(true); });
    on(audio, 'pause', function () { setPlaying(false); });
    on(audio, 'ended', function () { setPlaying(false); setProgress(1); });
    on(audio, 'timeupdate', update);
    on(audio, 'loadedmetadata', function () { dur.textContent = fmt(audio.duration); update(); });
    on(audio, 'durationchange', function () { dur.textContent = fmt(audio.duration); });
    on(audio, 'error', function () { if (err) err.hidden = false; btn.disabled = true; });
    on(seek, 'input', function () {
      dragging = true;
      var frac = seek.value / 1000;
      seek.style.setProperty('--p', (frac * 100).toFixed(2) + '%');
      if (isFinite(audio.duration)) cur.textContent = fmt(frac * audio.duration);
    });
    on(seek, 'change', function () {
      dragging = false;
      if (isFinite(audio.duration)) audio.currentTime = (seek.value / 1000) * audio.duration;
      update();
    });
  }

  // Immer nur ein Medium gleichzeitig.
  on(doc, 'play', function (e) {
    each(doc.querySelectorAll('audio, video'), function (m) {
      if (m !== e.target && !m.paused) m.pause();
    });
  }, true);

  /* ---------- Slideshow ---------- */
  function initSlideshow(el) {
    var track = el.querySelector('[data-track]');
    if (!track) return;
    var slides = track.children;
    var n = slides.length;
    if (n < 2) return;
    var prev = el.querySelector('[data-prev]');
    var next = el.querySelector('[data-next]');
    var dotsBox = el.querySelector('[data-dots]');
    var count = el.querySelector('[data-count]');
    var dots = [];
    var index = 0;
    var userTook = false;

    each(slides, function (s, i) {
      if (!dotsBox) return;
      var b = doc.createElement('button');
      b.type = 'button';
      b.className = 'slides__dot';
      b.setAttribute('aria-label', 'Foto ' + (i + 1) + ' von ' + n);
      on(b, 'click', function () { userTook = true; go(i); });
      dotsBox.appendChild(b);
      dots.push(b);
    });
    // Die Pfeile liegen über dem Foto, sind aber keine Kinder des Tracks: ein
    // Wisch, der auf einem Pfeil beginnt, würde sonst nichts bewegen. Deshalb
    // horizontale Wische auf den Pfeilen selbst erkennen und weiterblättern.
    var swipedAt = 0;
    function swipeable(btn) {
      var sx = 0, sy = 0, active = false;
      on(btn, 'touchstart', function (e) {
        var t = e.touches && e.touches[0];
        if (!t || e.touches.length > 1) { active = false; return; }
        sx = t.clientX; sy = t.clientY; active = true; userTook = true;
      }, { passive: true });
      on(btn, 'touchend', function (e) {
        var t = e.changedTouches && e.changedTouches[0];
        if (!active || !t) return;
        active = false;
        var dx = t.clientX - sx, dy = t.clientY - sy;
        if (Math.abs(dx) > 24 && Math.abs(dx) > Math.abs(dy) * 1.2) {
          swipedAt = Date.now();
          go(index + (dx < 0 ? 1 : -1));
        }
      }, { passive: true });
      on(btn, 'touchcancel', function () { active = false; }, { passive: true });
    }
    function step(d) {
      return function () {
        if (Date.now() - swipedAt < 600) return; // Klick nach Wisch ignorieren
        userTook = true;
        go(index + d);
      };
    }
    if (prev) { prev.hidden = false; on(prev, 'click', step(-1)); swipeable(prev); }
    if (next) { next.hidden = false; on(next, 'click', step(1)); swipeable(next); }

    function preload(i) {
      var s = slides[i];
      if (!s) return;
      var img = s.querySelector('img');
      if (img && img.getAttribute('loading') === 'lazy') {
        img.setAttribute('loading', 'eager');
        var pre = new Image();
        pre.src = img.currentSrc || img.src;
      }
    }
    function render() {
      each(dots, function (d, i) { d.setAttribute('aria-current', i === index ? 'true' : 'false'); });
      if (count) count.textContent = (index + 1) + ' / ' + n;
      if (prev) prev.disabled = index === 0;
      if (next) next.disabled = index === n - 1;
      preload(index + 1);
    }
    function go(i) {
      if (i < 0) i = 0;
      if (i > n - 1) i = n - 1;
      var left = i * track.clientWidth;
      if (track.scrollTo && !reduceMotion) {
        try { track.scrollTo({ left: left, behavior: 'smooth' }); } catch (e) { track.scrollLeft = left; }
      } else {
        track.scrollLeft = left;
      }
      index = i;
      render();
    }
    var ticking = false;
    on(track, 'scroll', function () {
      if (ticking) return;
      ticking = true;
      window.requestAnimationFrame(function () {
        ticking = false;
        var w = track.clientWidth || 1;
        var i = Math.round(track.scrollLeft / w);
        if (i !== index && i >= 0 && i < n) { index = i; render(); }
      });
    }, { passive: true });
    on(track, 'keydown', function (e) {
      if (e.key === 'ArrowLeft') { userTook = true; go(index - 1); e.preventDefault(); }
      if (e.key === 'ArrowRight') { userTook = true; go(index + 1); e.preventDefault(); }
    });
    on(track, 'touchstart', function () { userTook = true; }, { passive: true });
    on(track, 'pointerdown', function () { userTook = true; });
    on(window, 'resize', function () { track.scrollLeft = index * track.clientWidth; });

    var interval = parseInt(el.getAttribute('data-interval'), 10) || 0;
    if (interval >= 1500 && !reduceMotion) {
      setInterval(function () {
        if (userTook || doc.hidden) return;
        go(index === n - 1 ? 0 : index + 1);
      }, interval);
    }
    render();
  }

  initIntro();
  each(doc.querySelectorAll('[data-player]'), initPlayer);
  each(doc.querySelectorAll('[data-slideshow]'), initSlideshow);
})();
