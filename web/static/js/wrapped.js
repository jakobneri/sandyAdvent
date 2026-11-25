/* „Schatzo Wrapped“ (Türchen 22): Startkarte im Türchen, dann eine bildschirmfüllende
   Story mit animierten Slides aus stats.json. Rechts/links tippen = weiter/zurück,
   gedrückt halten = Pause, wischen, Auto-Weiter (~6 s, pausiert im Hintergrund),
   Pfeiltasten/Leertaste/Esc. „Bewegung reduzieren“ zeigt Endwerte ohne Animation. */
(function () {
  'use strict';
  var G = window.SAGame, L = window.WrappedLogic;
  if (!G || !L) return;
  var h = G.h;
  var w = window, doc = document;
  var reduce = G.reduceMotion;

  G.start('wrapped', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var slides = L.buildSlides(data);
    var seen = !!((store.load() || {}).seen);
    var overlay = null, player = null, slideEls = [], fills = [], raf = 0, lastT = 0, shown = -1, pushed = false;
    var startBtn, pausedEl, closeBtn;

    /* ---------- Startkarte ---------- */
    function buildStart() {
      G.clear(root);
      var secs = 0;
      for (var i = 0; i < slides.length; i++) secs += slides[i].ms / 1000;
      var mins = Math.max(1, Math.round(secs / 60));
      startBtn = G.button(seen ? 'Nochmal ansehen' : 'Abspielen', { onclick: open });
      startBtn.insertBefore(h('span', { class: 'wr-start__play', 'aria-hidden': 'true' }), startBtn.firstChild);
      var art = h('div', { class: 'wr-start__art', 'aria-hidden': 'true' }, [
        h('span', { class: 'wr-start__card wr-start__card--1' }),
        h('span', { class: 'wr-start__card wr-start__card--2' }),
        h('span', { class: 'wr-start__card wr-start__card--3' }, h('span', { class: 'wr-start__year', text: yearOf(data.title) }))
      ]);
      root.appendChild(h('section', { class: 'wr-start' }, [
        art,
        h('p', { class: 'wr-start__kicker', text: 'Unser Jahr im Rückblick' }),
        h('h2', { class: 'wr-start__title', text: data.title }),
        h('p', { class: 'wr-start__meta', text: G.count(slides.length, 'Slide', 'Slides') + ' · ca. ' + G.count(mins, 'Minute', 'Minuten') }),
        startBtn,
        h('p', { class: 'wr-start__hint', text: 'Rechts tippen: weiter · links: zurück · halten: Pause' })
      ]));
    }

    function yearOf(title) {
      var m = /(19|20)\d\d/.exec(title || '');
      return m ? m[0] : '♥';
    }

    /* ---------- Slides ---------- */
    function el(cls, text) { return h('div', { class: cls, text: text }); }

    function bars(list, invert) {
      var max = 0, i;
      for (i = 0; i < list.length; i++) max = Math.max(max, list[i].value);
      var box = h('div', { class: 'ws-bars' + (invert ? ' ws-bars--invert' : '') });
      for (i = 0; i < list.length; i++) {
        var v = max ? list[i].value / max : 0;
        if (invert && max) v = Math.max(0.12, Math.min(1, (list[0].value || 1) / (list[i].value || 1)));
        var fill = h('span', { class: 'ws-bar__fill' });
        fill.style.setProperty('--v', String(Math.max(0.04, v)));
        fill.style.setProperty('--i', String(i));
        box.appendChild(h('div', { class: 'ws-bar' }, [
          h('div', { class: 'ws-bar__head' }, [h('span', { class: 'ws-bar__label', text: list[i].label }), h('span', { class: 'ws-bar__note', text: list[i].note })]),
          h('div', { class: 'ws-bar__track' }, fill)
        ]));
      }
      return box;
    }

    function chart(values, peak, labels, cls) {
      var max = 0, i;
      for (i = 0; i < values.length; i++) max = Math.max(max, values[i]);
      var box = h('div', { class: 'ws-chart ' + cls, role: 'img' });
      for (i = 0; i < values.length; i++) {
        var col = h('span', { class: 'ws-chart__col' + (i === peak ? ' is-peak' : '') });
        col.style.setProperty('--v', String(max ? Math.max(0.03, values[i] / max) : 0.03));
        col.style.setProperty('--i', String(i));
        box.appendChild(col);
      }
      if (labels) {
        var lab = h('div', { class: 'ws-chart__labels', 'aria-hidden': 'true' });
        for (i = 0; i < labels.length; i++) lab.appendChild(h('span', { class: i === peak ? 'is-peak' : '', text: labels[i] }));
        return h('div', { class: 'ws-chartbox' }, [box, lab]);
      }
      return box;
    }

    function slideEl(s, idx) {
      var inner = h('div', { class: 'ws__inner' });
      var n = 0;
      function add(node) {
        if (!node) return;
        node.style.setProperty('--n', String(n++));
        G.addClass(node, 'ws-in');
        inner.appendChild(node);
      }
      add(el('ws__kicker', s.kicker));
      switch (s.id) {
        case 'intro':
          add(h('h2', { class: 'ws__title ws__title--hero', text: s.title }));
          add(el('ws__text', s.text));
          break;
        case 'days': case 'streak': case 'busiest': case 'messages':
          add(h('h2', { class: 'ws__title', text: s.title }));
          var num = h('div', { class: 'ws__num', 'data-count': String(s.value), text: reduce ? L.fmtNum(s.value) : '0' });
          num.setAttribute('aria-label', L.fmtNum(s.value));
          add(num);
          add(el('ws__unit', s.unit));
          if (s.bars) add(bars(s.bars));
          add(s.text ? el('ws__text', s.text) : null);
          add(s.extra ? el('ws__extra', s.extra) : null);
          break;
        case 'emojis':
          add(h('h2', { class: 'ws__title', text: s.title }));
          var grid = h('div', { class: 'ws-emojis' });
          for (var i = 0; i < s.emojis.length; i++) {
            var it = h('div', { class: 'ws-emoji' + (i === 0 ? ' ws-emoji--top' : '') }, [
              h('span', { class: 'ws-emoji__e', text: s.emojis[i].emoji }),
              h('span', { class: 'ws-emoji__c', text: L.fmtNum(s.emojis[i].count) + '×' })
            ]);
            it.style.setProperty('--i', String(i));
            grid.appendChild(it);
          }
          add(grid);
          add(el('ws__text', s.text));
          break;
        case 'goodnight': case 'response':
          add(h('h2', { class: 'ws__title', text: s.title }));
          add(el('ws__winner', s.winner));
          add(bars(s.bars, s.invert));
          add(el('ws__text', s.text));
          break;
        case 'hours':
          add(h('h2', { class: 'ws__title', text: s.title }));
          add(el('ws__big', s.big));
          var hl = [];
          for (var k = 0; k < 24; k++) hl.push(k % 6 === 0 ? String(k) : '');
          add(chart(s.hist, s.peak, hl, 'ws-chart--hours'));
          if (s.week) {
            add(el('ws__sub', 'Und am liebsten am ' + s.weekday + '.'));
            add(chart(s.week, s.weekPeak, s.weekLabels, 'ws-chart--week'));
          }
          add(el('ws__text', s.text));
          break;
        case 'words':
          add(h('h2', { class: 'ws__title', text: s.title }));
          var cloud = h('div', { class: 'ws-cloud' });
          var top = s.words[0].count || 1;
          for (var j = 0; j < s.words.length; j++) {
            var wd = h('span', { class: 'ws-word ws-word--c' + (j % 4), text: s.words[j].word });
            wd.style.setProperty('--s', String(0.55 + 0.45 * Math.sqrt(s.words[j].count / top)));
            wd.style.setProperty('--i', String(j));
            wd.style.setProperty('--r', String(j % 3 === 1 ? -4 : j % 3 === 2 ? 3 : 0));
            cloud.appendChild(wd);
          }
          add(cloud);
          add(el('ws__text', s.text));
          break;
        case 'first':
          add(h('h2', { class: 'ws__title', text: s.title }));
          add(h('figure', { class: 'ws-bubble' }, [
            h('blockquote', { class: 'ws-bubble__text', text: s.quote }),
            h('figcaption', { class: 'ws-bubble__meta', text: s.meta })
          ]));
          add(s.text ? el('ws__text', s.text) : null);
          break;
        case 'outro':
          add(h('h2', { class: 'ws__title ws__title--hero', text: s.title }));
          add(h('div', { class: 'ws__text ws__text--outro' }, G.paragraphs(s.text)));
          add(h('div', { class: 'ws__actions' }, [
            G.button('Nochmal ansehen', { onclick: function () { go(0); } }),
            G.button('Schließen', { variant: 'ghost', onclick: function () { close(); } })
          ]));
          break;
      }
      var deco = h('div', { class: 'ws__deco', 'aria-hidden': 'true' }, [h('span'), h('span'), h('span')]);
      if (s.id === 'outro' || s.id === 'intro') {
        for (var c = 0; c < 18; c++) {
          var flake = h('i', { class: 'ws-flake' });
          flake.style.setProperty('--x', String((c * 37) % 100));
          flake.style.setProperty('--d', String((c * 0.53) % 6));
          flake.style.setProperty('--z', String(0.6 + (c % 5) / 5));
          deco.appendChild(flake);
        }
      }
      return h('section', {
        class: 'ws ws--' + s.id, 'aria-roledescription': 'Slide',
        'aria-label': 'Slide ' + (idx + 1) + ' von ' + slides.length + ': ' + s.kicker, 'aria-hidden': 'true'
      }, [deco, inner]);
    }

    /* ---------- Zahlen hochzählen ---------- */
    function countUp(node) {
      var to = parseInt(node.getAttribute('data-count'), 10) || 0;
      if (reduce || !w.requestAnimationFrame) { node.textContent = L.fmtNum(to); return; }
      var start = 0, dur = Math.min(1600, 700 + to / 3);
      node.textContent = '0';
      function step(t) {
        if (!start) start = t;
        var p = Math.min(1, (t - start) / dur);
        var e = 1 - Math.pow(1 - p, 3);
        node.textContent = L.fmtNum(to * e);
        if (p < 1 && node.getAttribute('data-run') === '1') w.requestAnimationFrame(step);
      }
      node.setAttribute('data-run', '1');
      w.requestAnimationFrame(step);
    }

    /* ---------- Anzeigen ---------- */
    function show(i) {
      if (i === shown) return;
      for (var k = 0; k < slideEls.length; k++) {
        var s = slideEls[k];
        s.className = s.className.replace(/\s*is-(active|past|future)/g, '') + (k === i ? ' is-active' : k < i ? ' is-past' : ' is-future');
        s.setAttribute('aria-hidden', k === i ? 'false' : 'true');
        var nums = s.querySelectorAll('[data-count]');
        for (var j = 0; j < nums.length; j++) {
          if (k === i) countUp(nums[j]);
          else nums[j].setAttribute('data-run', '0');
        }
      }
      shown = i;
      overlay.setAttribute('data-slide', slides[i].id);
      if (i === slides.length - 1) {
        store.save({ seen: true });
        seen = true;
      }
    }

    function paintBars() {
      for (var i = 0; i < fills.length; i++) {
        var p = player.progress(i);
        fills[i].style.transform = 'scaleX(' + p.toFixed(4) + ')';
      }
    }

    function frame(t) {
      if (!overlay) return;
      var dt = lastT ? Math.min(t - lastT, 100) : 0;
      lastT = t;
      if (player.tick(dt) === 'advance') show(player.index);
      paintBars();
      raf = w.requestAnimationFrame(frame);
    }

    function go(i) { player.goTo(i); show(player.index); paintBars(); }
    function next() { if (player.next()) show(player.index); paintBars(); }
    function prev() { player.prev(); shown = -1; show(player.index); paintBars(); }

    function setPaused(why, on) {
      if (on) player.pause(why); else player.resume(why);
      if (overlay) {
        var p = player.isPaused();
        G.toggleClass(overlay, 'is-paused', p);
        pausedEl.hidden = !(p && (player.reasons.key || player.reasons.hold));
      }
    }

    /* ---------- Öffnen / Schließen ---------- */
    function open() {
      if (overlay || !slides.length) return;
      slideEls = [];
      fills = [];
      shown = -1;
      var barsBox = h('div', { class: 'wr__bars', 'aria-hidden': 'true' });
      for (var i = 0; i < slides.length; i++) {
        var f = h('span', { class: 'wr__fill' });
        fills.push(f);
        barsBox.appendChild(h('span', { class: 'wr__seg' }, f));
      }
      var stage = h('div', { class: 'wr__stage' });
      for (i = 0; i < slides.length; i++) {
        var s = slideEl(slides[i], i);
        slideEls.push(s);
        stage.appendChild(s);
      }
      closeBtn = h('button', { type: 'button', class: 'wr__close', 'aria-label': 'Schließen' });
      closeBtn.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M6 6l12 12M18 6L6 18"/></svg>'; // statisches SVG
      closeBtn.addEventListener('click', function () { close(); }, false);
      pausedEl = h('span', { class: 'wr__paused', hidden: true, text: 'Pause' });
      overlay = h('div', { class: 'wr', role: 'dialog', 'aria-modal': 'true', 'aria-label': data.title, tabindex: '-1' }, [
        stage,
        h('div', { class: 'wr__top' }, [barsBox, h('div', { class: 'wr__row' }, [
          h('span', { class: 'wr__brand', text: data.title }), pausedEl, closeBtn
        ])])
      ]);
      doc.body.appendChild(overlay);
      G.addClass(doc.documentElement, 'wr-open');
      player = L.player(slides.map(function (x) { return x.ms; }));
      bindGestures(stage);
      show(0);
      paintBars();
      try { overlay.focus({ preventScroll: true }); } catch (e) { overlay.focus(); }
      lastT = 0;
      raf = w.requestAnimationFrame(frame);
      if (doc.hidden) setPaused('hidden', true);
      // Zurück-Taste (Android) schließt die Story statt die Seite zu verlassen
      try { w.history.pushState({ sawrapped: 1 }, ''); pushed = true; } catch (e) { pushed = false; }
      w.requestAnimationFrame(function () { if (overlay) G.addClass(overlay, 'is-open'); });
    }

    function close(fromHistory) {
      if (!overlay) return;
      w.cancelAnimationFrame(raf);
      var o = overlay;
      overlay = null;
      G.removeClass(doc.documentElement, 'wr-open');
      if (o.parentNode) o.parentNode.removeChild(o);
      if (pushed && !fromHistory) { pushed = false; try { w.history.back(); } catch (e) {} }
      pushed = false;
      buildStart();
      try { startBtn.focus({ preventScroll: true }); } catch (e) {}
    }

    w.addEventListener('popstate', function () { if (overlay) { pushed = false; close(true); } }, false);
    doc.addEventListener('visibilitychange', function () { if (overlay) setPaused('hidden', doc.hidden); }, false);
    doc.addEventListener('keydown', function (e) {
      if (!overlay || e.ctrlKey || e.metaKey || e.altKey) return;
      var k = e.key;
      if (k === 'ArrowRight' || k === 'PageDown') { e.preventDefault(); next(); }
      else if (k === 'ArrowLeft' || k === 'PageUp') { e.preventDefault(); prev(); }
      else if (k === 'Escape' || k === 'Esc') { e.preventDefault(); close(); }
      else if ((k === ' ' || k === 'Spacebar') && !(e.target && e.target.tagName === 'BUTTON')) {
        e.preventDefault();
        setPaused('key', !player.reasons.key);
      }
    }, false);

    /* ---------- Gesten ---------- */
    function bindGestures(stage) {
      var down = null, holdT = 0, held = false;
      function isUI(t) {
        while (t && t !== stage) { if (t.tagName === 'BUTTON' || t.tagName === 'A') return true; t = t.parentNode; }
        return false;
      }
      function start(x, y, target) {
        if (isUI(target)) { down = null; return; }
        down = { x: x, y: y, t: Date.now() };
        held = false;
        w.clearTimeout(holdT);
        holdT = w.setTimeout(function () { held = true; setPaused('hold', true); G.addClass(overlay, 'is-holding'); }, 220);
      }
      function end(x, y) {
        w.clearTimeout(holdT);
        if (!down || !overlay) return;
        var r = stage.getBoundingClientRect();
        var act = L.gesture({ dx: x - down.x, dy: y - down.y, ms: held ? 1000 : Date.now() - down.t, x: x - r.left, width: r.width });
        down = null;
        G.removeClass(overlay, 'is-holding');
        setPaused('hold', false);
        if (act === 'next') next();
        else if (act === 'prev') prev();
        else if (act === 'close') close();
      }
      function cancel() {
        w.clearTimeout(holdT);
        down = null;
        if (overlay) { G.removeClass(overlay, 'is-holding'); setPaused('hold', false); }
      }
      if (w.PointerEvent) {
        stage.addEventListener('pointerdown', function (e) {
          if (e.pointerType === 'mouse' && e.button !== 0) return;
          start(e.clientX, e.clientY, e.target);
        }, false);
        stage.addEventListener('pointerup', function (e) { end(e.clientX, e.clientY); }, false);
        stage.addEventListener('pointercancel', cancel, false);
      } else {
        stage.addEventListener('touchstart', function (e) { var t = e.changedTouches[0]; start(t.clientX, t.clientY, e.target); }, { passive: true });
        stage.addEventListener('touchend', function (e) { var t = e.changedTouches[0]; end(t.clientX, t.clientY); }, false);
        stage.addEventListener('touchcancel', cancel, false);
      }
      // Seite dahinter darf nicht scrollen/zoomen (iOS ignoriert overflow: hidden teilweise)
      overlay.addEventListener('touchmove', function (e) { if (e.cancelable) e.preventDefault(); }, { passive: false });
      stage.addEventListener('contextmenu', function (e) { e.preventDefault(); }, false);
    }

    buildStart();
  });
})();
