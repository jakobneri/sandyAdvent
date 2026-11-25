/* Foto-Puzzle: Jigsaw (Teile mit Nasen per Canvas, ziehen + einrasten) oder
   Schiebepuzzle (N×N mit Lücke, tippen oder wischen). Pointer Events mit
   Touch-/Maus-Fallback für ältere iOS-Safaris. */
(function () {
  'use strict';
  var G = window.SAGame, P = window.PuzzleLogic;
  if (!G || !P) return;
  var h = G.h;
  var w = window, doc = document;
  var HAS_POINTER = !!w.PointerEvent;

  G.start('puzzle', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var N = data.grid || 4;
    var img = new Image();
    var state, aspect = 1;
    var bar, boardWrap, winBox, previewEl, previewBtn;

    root.appendChild(h('p', { class: 'game__loading', text: 'Das Bild wird geladen …' }));
    img.onload = function () {
      aspect = (img.naturalHeight || 3) / (img.naturalWidth || 4);
      state = store.load();
      if (data.mode === 'sliding') initSliding(); else initJigsaw();
    };
    img.onerror = function () {
      G.clear(root);
      root.appendChild(h('p', { class: 'game__error', text: 'Das Puzzle-Bild konnte nicht geladen werden.' }));
    };
    img.src = data.image;

    /* ---------- gemeinsame Teile ---------- */
    function boardSize(maxFrac) {
      var W = root.clientWidth || 320;
      var bw = W, bh = W * aspect;
      var maxH = Math.max(220, (w.innerHeight || 640) * maxFrac);
      if (bh > maxH) { bh = maxH; bw = bh / aspect; }
      return { W: W, w: Math.floor(bw), h: Math.floor(bh) };
    }

    function frame(extraBtns) {
      G.clear(root);
      bar = h('div', { class: 'gbar' });
      root.appendChild(bar);
      var tools = h('div', { class: 'ptools' });
      if (data.show_preview) {
        previewBtn = G.button('Vorschau', { variant: 'small' });
        previewBtn.setAttribute('aria-pressed', 'false');
        previewBtn.addEventListener('click', function () { setPreview(!G.hasClass(previewEl, 'is-on')); }, false);
        tools.appendChild(previewBtn);
      }
      for (var i = 0; i < (extraBtns || []).length; i++) tools.appendChild(extraBtns[i]);
      if (tools.firstChild) root.appendChild(tools);
      boardWrap = h('div', { class: 'pwrap' });
      root.appendChild(boardWrap);
      winBox = h('div', { class: 'game__win' });
      root.appendChild(winBox);
    }

    function makePreview(bw, bh) {
      previewEl = h('div', { class: 'ppreview', 'aria-hidden': 'true' }, h('img', { src: data.image, alt: '', draggable: 'false' }));
      previewEl.style.width = bw + 'px';
      previewEl.style.height = bh + 'px';
      previewEl.addEventListener('click', function () { setPreview(false); }, false);
      return previewEl;
    }
    function setPreview(on) {
      if (!previewEl) return;
      G.toggleClass(previewEl, 'is-on', on);
      if (previewBtn) {
        previewBtn.setAttribute('aria-pressed', on ? 'true' : 'false');
        previewBtn.textContent = on ? 'Vorschau aus' : 'Vorschau';
      }
    }

    function showWin(stats, animate, anchor) {
      setPreview(false);
      var tools = root.querySelector('.ptools');
      if (tools) tools.hidden = true;
      var panel = G.winPanel({
        title: 'Puzzle gelöst!',
        stats: stats,
        text: data.win_text,
        onReplay: function () {
          store.clear();
          state = null;
          if (data.mode === 'sliding') initSliding(); else initJigsaw();
          G.scrollIntoView(root);
        }
      });
      G.clear(winBox);
      winBox.appendChild(panel);
      if (animate) {
        G.celebrate(anchor, true);
        G.haptic([30, 50, 30]);
        w.setTimeout(function () { G.scrollIntoView(panel); }, 900);
      }
    }

    // Nur auf Breitenänderungen reagieren (G.onWidthChange): die ein-/ausfahrende
    // Adressleiste auf dem Handy ändert nur die Höhe und darf das Puzzle nicht neu bauen.
    var resizeHandler = null;
    function setResize(fn) {
      if (!resizeHandler) G.onWidthChange(root, function () { if (resizeHandler) resizeHandler(); });
      resizeHandler = fn;
    }

    // Einheitliche Zeiger-Ereignisse: Pointer Events, sonst Touch + Maus.
    function bindPointer(el, down, move, up) {
      if (HAS_POINTER) {
        el.addEventListener('pointerdown', function (e) {
          if (e.pointerType === 'mouse' && e.button !== 0) return;
          if (down(e.clientX, e.clientY, e) && el.setPointerCapture) {
            try { el.setPointerCapture(e.pointerId); } catch (x) {}
          }
        }, false);
        el.addEventListener('pointermove', function (e) { move(e.clientX, e.clientY, e); }, false);
        el.addEventListener('pointerup', function (e) { up(e.clientX, e.clientY, e); }, false);
        el.addEventListener('pointercancel', function (e) { up(e.clientX, e.clientY, e, true); }, false);
        return;
      }
      el.addEventListener('touchstart', function (e) {
        var t = e.changedTouches[0];
        if (t && down(t.clientX, t.clientY, e)) e.preventDefault();
      }, { passive: false });
      el.addEventListener('touchmove', function (e) {
        var t = e.changedTouches[0];
        if (t && move(t.clientX, t.clientY, e)) e.preventDefault();
      }, { passive: false });
      el.addEventListener('touchend', function (e) {
        var t = e.changedTouches[0];
        if (t) up(t.clientX, t.clientY, e);
      }, false);
      el.addEventListener('touchcancel', function (e) {
        var t = e.changedTouches[0];
        up(t ? t.clientX : 0, t ? t.clientY : 0, e, true);
      }, false);
      var mouseDown = false;
      el.addEventListener('mousedown', function (e) { if (e.button === 0 && down(e.clientX, e.clientY, e)) { mouseDown = true; e.preventDefault(); } }, false);
      doc.addEventListener('mousemove', function (e) { if (mouseDown) move(e.clientX, e.clientY, e); }, false);
      doc.addEventListener('mouseup', function (e) { if (mouseDown) { mouseDown = false; up(e.clientX, e.clientY, e); } }, false);
    }

    /* =====================================================================
       Schiebepuzzle
       ===================================================================== */
    function initSliding() {
      if (!state || state.v !== 1 || state.mode !== 'sliding' || state.n !== N || !(state.tiles instanceof Array) ||
          state.tiles.length !== N * N || !P.isSolvable(state.tiles, N)) {
        state = { v: 1, mode: 'sliding', n: N, tiles: P.shuffleSolvable(N), moves: 0, done: false, nums: false };
      }
      var numsBtn = G.button('Zahlen', { variant: 'small' });
      numsBtn.setAttribute('aria-pressed', state.nums ? 'true' : 'false');
      frame([numsBtn]);
      var movesEl = h('span', { class: 'gchip gchip--gold', 'data-moves': '' });
      var sizeEl = h('span', { class: 'gchip', text: N + ' × ' + N });
      bar.appendChild(sizeEl);
      bar.appendChild(movesEl);
      var board, tiles = [], sz, swiped = false;

      function build() {
        sz = boardSize(0.62);
        G.clear(boardWrap);
        board = h('div', { class: 'sboard', role: 'group', 'aria-label': 'Schiebepuzzle, ' + N + ' mal ' + N });
        board.style.width = sz.w + 'px';
        board.style.height = sz.h + 'px';
        G.toggleClass(board, 'show-nums', !!state.nums);
        tiles = [];
        var tw = sz.w / N, th = sz.h / N;
        for (var id = 0; id < N * N; id++) {
          var r0 = Math.floor(id / N), c0 = id % N;
          // Bild auf einem inneren Element: der Knopf trägt nur die Position (transform),
          // Animationen wie das „geht nicht“-Wackeln laufen auf dem Bild.
          var face = h('span', { class: 'stile__face', 'aria-hidden': 'true' });
          face.style.backgroundImage = 'url("' + data.image + '")';
          face.style.backgroundSize = sz.w + 'px ' + sz.h + 'px';
          face.style.backgroundPosition = (-c0 * tw) + 'px ' + (-r0 * th) + 'px';
          var t = h('button', { type: 'button', class: 'stile' + (id === N * N - 1 ? ' stile--last' : ''), 'data-tile': String(id) },
            [face, h('span', { class: 'stile__num', 'aria-hidden': 'true', text: String(id + 1) })]);
          t.style.width = Math.ceil(tw) + 'px';
          t.style.height = Math.ceil(th) + 'px';
          (function (id) {
            t.addEventListener('click', function () {
              if (swiped) { swiped = false; return; }
              tap(state.tiles.indexOf(id));
            }, false);
          })(id);
          tiles.push(t);
          board.appendChild(t);
        }
        board.appendChild(makePreview(sz.w, sz.h));
        boardWrap.appendChild(board);
        bindSwipe();
        place(false);
        render();
        if (state.done) { G.addClass(board, 'is-solved'); }
      }

      function place(animate) {
        var tw = sz.w / N, th = sz.h / N;
        G.toggleClass(board, 'is-anim', !!animate && !G.reduceMotion);
        for (var pos = 0; pos < state.tiles.length; pos++) {
          var id = state.tiles[pos];
          var t = tiles[id];
          var r = Math.floor(pos / N), c = pos % N;
          t.style.transform = 'translate3d(' + (c * tw) + 'px,' + (r * th) + 'px,0)';
          t.style.webkitTransform = t.style.transform;
          t.setAttribute('data-pos', String(pos));
          var isBlank = id === N * N - 1;
          t.setAttribute('aria-label', isBlank ? 'Lücke' : 'Teil ' + (id + 1) + (P.slide(state.tiles, N, pos) ? ', verschiebbar' : ''));
          if (isBlank) t.tabIndex = -1;
        }
      }

      function render() {
        movesEl.textContent = G.count(state.moves, 'Zug', 'Züge');
      }

      function tap(pos) {
        if (state.done) return;
        var next = P.slide(state.tiles, N, pos);
        if (!next) {
          if (pos >= 0 && state.tiles[pos] !== N * N - 1) G.replayClass(tiles[state.tiles[pos]], 'is-nope');
          return;
        }
        state.tiles = next;
        state.moves++;
        G.haptic(6);
        place(true);
        render();
        if (P.isSolved(state.tiles)) {
          state.done = true;
          store.save(state);
          w.setTimeout(function () {
            G.addClass(board, 'is-solved');
            showWin([G.count(state.moves, 'Zug', 'Züge')], true, board);
          }, 200);
        } else {
          store.save(state);
        }
      }

      function bindSwipe() {
        var sx = 0, sy = 0, active = false;
        bindPointer(board, function (x, y) {
          if (state.done) return false;
          sx = x; sy = y; active = true;
          return false; // kein Capture nötig, Klicks sollen normal durchgehen
        }, function () { return active; }, function (x, y, e, cancel) {
          if (!active) return;
          active = false;
          if (cancel) return;
          var dx = x - sx, dy = y - sy;
          if (Math.abs(dx) < 24 && Math.abs(dy) < 24) return;
          var dir = Math.abs(dx) > Math.abs(dy) ? (dx < 0 ? 'left' : 'right') : (dy < 0 ? 'up' : 'down');
          var target = P.swipeTarget(state.tiles, N, dir);
          swiped = true;
          w.setTimeout(function () { swiped = false; }, 350);
          if (target >= 0) tap(target);
        });
      }

      numsBtn.addEventListener('click', function () {
        state.nums = !state.nums;
        numsBtn.setAttribute('aria-pressed', state.nums ? 'true' : 'false');
        G.toggleClass(board, 'show-nums', state.nums);
        store.save(state);
      }, false);

      build();
      if (state.done) showWin([G.count(state.moves, 'Zug', 'Züge')], false, board);
      setResize(build);
    }

    /* =====================================================================
       Jigsaw
       ===================================================================== */
    function initJigsaw() {
      var total = N * N;
      if (!state || state.v !== 1 || state.mode !== 'jigsaw' || state.n !== N || !P.validEdges(state.edges, N) ||
          !(state.pieces instanceof Array) || state.pieces.length !== total) {
        var slots = G.shuffle(G.range(total));
        var pieces = [];
        for (var i = 0; i < total; i++) pieces.push({ id: i, slot: slots[i], locked: false, fx: null, fy: null });
        state = { v: 1, mode: 'jigsaw', n: N, edges: P.makeEdges(N), pieces: pieces, done: false };
      }
      frame();
      var countEl = h('span', { class: 'gchip gchip--gold', 'data-placed': '' });
      bar.appendChild(h('span', { class: 'gchip', text: total + ' Teile' }));
      bar.appendChild(countEl);
      var stage, boardEl, trayEl, full, L, els = [], drag = null;
      var dpr = Math.min(w.devicePixelRatio || 1, 2);

      function placedCount() {
        var n = 0;
        for (var i = 0; i < state.pieces.length; i++) if (state.pieces[i].locked) n++;
        return n;
      }
      function render() { countEl.textContent = placedCount() + ' / ' + total + ' gelegt'; }

      function computeLayout() {
        var sz = boardSize(0.5);
        var cw = sz.w / N, ch = sz.h / N;
        var t = Math.round(Math.min(cw, ch) * 0.22);
        var bx = Math.floor((sz.W - sz.w) / 2);
        var trayTop = sz.h + 14;
        var maxTray = Math.max(150, (w.innerHeight || 640) * 0.36);
        var ts = 0.78, cols, rows, px, py;
        for (; ts >= 0.4; ts -= 0.02) {
          px = (cw + t * 1.2) * ts + 4;
          py = (ch + t * 1.2) * ts + 4;
          cols = Math.max(1, Math.floor((sz.W - 8) / px));
          rows = Math.ceil(total / cols);
          if (rows * py + 12 <= maxTray) break;
        }
        if (ts < 0.4) ts = 0.4;
        var trayH = Math.ceil(rows * py + 12);
        var used = cols * px;
        return { W: sz.W, bw: sz.w, bh: sz.h, cw: cw, ch: ch, t: t, bx: bx, by: 0, trayTop: trayTop, trayH: trayH,
          ts: ts, cols: cols, px: px, py: py, ox: Math.max(4, (sz.W - used) / 2), stageH: trayTop + trayH };
      }

      function slotPos(slot) {
        var c = slot % L.cols, r = Math.floor(slot / L.cols);
        var s = L.ts;
        // Zellmitte in die Slotmitte legen
        var cx = L.ox + c * L.px + L.px / 2, cy = L.trayTop + 6 + r * L.py + L.py / 2;
        return { x: cx - L.cw * s / 2, y: cy - L.ch * s / 2, s: s };
      }
      function targetPos(id) {
        var r = Math.floor(id / N), c = id % N;
        return { x: L.bx + c * L.cw, y: L.by + r * L.ch };
      }

      // Umriss eines Teils mit Nasen (Ursprung = linke obere Ecke der Zelle).
      function tracePiece(c2, e) {
        var cw = L.cw, ch = L.ch, t = L.t;
        c2.beginPath();
        c2.moveTo(0, 0);
        edge(c2, 0, 0, 1, 0, 0, -1, cw, e.top, t);
        edge(c2, cw, 0, 0, 1, 1, 0, ch, e.right, t);
        edge(c2, cw, ch, -1, 0, 0, 1, cw, e.bottom, t);
        edge(c2, 0, ch, 0, -1, -1, 0, ch, e.left, t);
        c2.closePath();
      }
      function edge(c2, x0, y0, dx, dy, nx, ny, len, s, t) {
        if (!s) { c2.lineTo(x0 + dx * len, y0 + dy * len); return; }
        function X(a, b) { return x0 + dx * len * a + nx * t * b * s; }
        function Y(a, b) { return y0 + dy * len * a + ny * t * b * s; }
        c2.lineTo(X(0.36, 0), Y(0.36, 0));
        c2.bezierCurveTo(X(0.42, 0.05), Y(0.42, 0.05), X(0.44, 0.35), Y(0.44, 0.35), X(0.40, 0.55), Y(0.40, 0.55));
        c2.bezierCurveTo(X(0.34, 0.85), Y(0.34, 0.85), X(0.42, 1.05), Y(0.42, 1.05), X(0.50, 1.05), Y(0.50, 1.05));
        c2.bezierCurveTo(X(0.58, 1.05), Y(0.58, 1.05), X(0.66, 0.85), Y(0.66, 0.85), X(0.60, 0.55), Y(0.60, 0.55));
        c2.bezierCurveTo(X(0.56, 0.35), Y(0.56, 0.35), X(0.58, 0.05), Y(0.58, 0.05), X(0.64, 0), Y(0.64, 0));
        c2.lineTo(x0 + dx * len, y0 + dy * len);
      }

      function drawPiece(cv, id) {
        var r = Math.floor(id / N), c = id % N;
        var cwT = L.cw + 2 * L.t, chT = L.ch + 2 * L.t;
        cv.width = Math.ceil(cwT * dpr);
        cv.height = Math.ceil(chT * dpr);
        cv.style.width = cwT + 'px';
        cv.style.height = chT + 'px';
        var c2 = cv.getContext('2d');
        c2.setTransform(dpr, 0, 0, dpr, 0, 0);
        c2.translate(L.t, L.t);
        var e = P.pieceEdges(state.edges, N, r, c);
        c2.save();
        tracePiece(c2, e);
        c2.clip();
        c2.drawImage(img, -c * L.cw, -r * L.ch, L.bw, L.bh);
        // leichte Kante oben/links (Licht) und unten/rechts (Schatten)
        c2.lineWidth = 2.5;
        c2.strokeStyle = 'rgba(255, 248, 230, .35)';
        c2.translate(1, 1);
        tracePiece(c2, e);
        c2.stroke();
        c2.restore();
        c2.lineWidth = 1.2;
        c2.strokeStyle = 'rgba(40, 20, 10, .55)';
        tracePiece(c2, e);
        c2.stroke();
      }

      function setPos(i, x, y, s) {
        var p = els[i];
        p.x = x; p.y = y; p.s = s;
        var tf = 'translate3d(' + (x - L.t) + 'px,' + (y - L.t) + 'px,0) scale(' + s + ')';
        p.el.style.transform = tf;
        p.el.style.webkitTransform = tf;
      }

      // Lose Teile ganz im Bild halten (inkl. Nasen), sonst ragen sie über den Bildschirmrand.
      function keepOnScreen(x, y) {
        return { x: Math.max(L.t, Math.min(L.W - L.cw - L.t, x)), y: Math.max(L.t, y) };
      }

      function homePos(i) {
        var st = state.pieces[i];
        if (st.locked) { var tp = targetPos(st.id); return { x: tp.x, y: tp.y, s: 1 }; }
        if (st.fx !== null && st.fx !== undefined) {
          var k = keepOnScreen(L.bx + st.fx * L.bw, L.by + st.fy * L.bh);
          return { x: k.x, y: k.y, s: 1 };
        }
        return slotPos(st.slot);
      }

      function build() {
        L = computeLayout();
        G.clear(boardWrap);
        stage = h('div', { class: 'jstage' });
        stage.style.height = L.stageH + 'px';
        boardEl = h('div', { class: 'jboard', 'data-board': '', 'data-grid': String(N) });
        boardEl.style.left = L.bx + 'px';
        boardEl.style.top = L.by + 'px';
        boardEl.style.width = L.bw + 'px';
        boardEl.style.height = L.bh + 'px';
        boardEl.style.backgroundSize = L.cw + 'px ' + L.ch + 'px';
        full = h('img', { class: 'jboard__full', src: data.image, alt: 'Das fertige Puzzle', draggable: 'false' });
        boardEl.appendChild(full);
        boardEl.appendChild(makePreview(L.bw, L.bh));
        trayEl = h('div', { class: 'jtray', 'aria-hidden': 'true' });
        trayEl.style.top = L.trayTop + 'px';
        trayEl.style.height = L.trayH + 'px';
        stage.appendChild(boardEl);
        stage.appendChild(trayEl);
        els = [];
        var order = [];
        for (var i = 0; i < state.pieces.length; i++) order.push(i);
        // gelegte Teile zuerst (unten), lose oben drauf
        order.sort(function (a, b) { return (state.pieces[b].locked ? 1 : 0) - (state.pieces[a].locked ? 1 : 0); });
        for (var k = 0; k < order.length; k++) {
          var idx = order[k];
          var st = state.pieces[idx];
          var r = Math.floor(st.id / N), c = st.id % N;
          var cv = h('canvas', { class: 'jpiece' + (st.locked ? ' is-locked' : ''), 'data-piece': r + ',' + c, 'aria-hidden': 'true' });
          cv.style.transformOrigin = L.t + 'px ' + L.t + 'px';
          els[idx] = { el: cv, x: 0, y: 0, s: 1 };
          drawPiece(cv, st.id);
          stage.appendChild(cv);
        }
        for (var j = 0; j < els.length; j++) { var hp = homePos(j); setPos(j, hp.x, hp.y, hp.s); }
        boardWrap.appendChild(stage);
        bindPointer(stage, down, move, up);
        render();
        if (state.done) showDone();
      }

      // fertiges Bild zeigen und die leere Ablage einklappen (auch nach jedem Neuaufbau)
      function showDone() {
        G.addClass(boardEl, 'is-done');
        trayEl.hidden = true;
        stage.style.height = Math.ceil(L.bh + 10) + 'px';
      }

      function local(x, y) {
        var r = stage.getBoundingClientRect();
        return { x: x - r.left, y: y - r.top };
      }

      function hit(sx, sy) {
        var kids = stage.querySelectorAll('.jpiece');
        for (var k = kids.length - 1; k >= 0; k--) {
          var i = indexOfEl(kids[k]);
          if (i < 0 || state.pieces[i].locked) continue;
          var p = els[i];
          var u = (sx - p.x) / p.s + L.t, v = (sy - p.y) / p.s + L.t;
          if (u < 0 || v < 0 || u > L.cw + 2 * L.t || v > L.ch + 2 * L.t) continue;
          // im Kern der Zelle immer, auf Nasen nur bei sichtbaren Pixeln
          if (u >= L.t && v >= L.t && u <= L.t + L.cw && v <= L.t + L.ch) return i;
          try {
            var a = p.el.getContext('2d').getImageData(Math.floor(u * dpr), Math.floor(v * dpr), 1, 1).data[3];
            if (a > 24) return i;
          } catch (e) { return i; }
        }
        return -1;
      }
      function indexOfEl(el) {
        for (var i = 0; i < els.length; i++) if (els[i] && els[i].el === el) return i;
        return -1;
      }

      var raf = 0;
      function down(x, y, e) {
        if (state.done || drag) return false;
        var pt = local(x, y);
        var i = hit(pt.x, pt.y);
        if (i < 0) return false;
        if (e && e.preventDefault && e.type !== 'touchstart') e.preventDefault();
        var p = els[i];
        var gu = (pt.x - p.x) / p.s, gv = (pt.y - p.y) / p.s;
        drag = { i: i, gu: gu, gv: gv };
        stage.appendChild(p.el); // nach vorn
        G.addClass(p.el, 'is-dragging');
        setPos(i, pt.x - gu, pt.y - gv, 1);
        G.haptic(5);
        return true;
      }
      function move(x, y, e) {
        if (!drag) return false;
        if (e && e.preventDefault && e.cancelable) e.preventDefault();
        var pt = local(x, y);
        // rechts nie über den Bildschirmrand (sonst wird die Seite breiter), links darf es etwas hinaus
        var nx = Math.max(-L.cw * 0.5, Math.min(L.W - L.cw - L.t, pt.x - drag.gu));
        var ny = Math.max(-L.ch * 0.5, Math.min(L.stageH - L.ch * 0.5, pt.y - drag.gv));
        drag.nx = nx; drag.ny = ny;
        if (!raf) {
          raf = w.requestAnimationFrame(function () {
            raf = 0;
            if (drag && drag.nx !== undefined) setPos(drag.i, drag.nx, drag.ny, 1);
          });
        }
        return true;
      }
      function up(x, y, e, cancel) {
        if (!drag) return;
        var i = drag.i, p = els[i], st = state.pieces[i];
        if (drag.nx !== undefined) setPos(i, drag.nx, drag.ny, 1);
        drag = null;
        G.removeClass(p.el, 'is-dragging');
        var tp = targetPos(st.id);
        var tol = Math.min(L.cw, L.ch) * 0.32;
        if (!cancel && P.snaps(p.x, p.y, tp.x, tp.y, tol)) {
          st.locked = true; st.fx = null; st.fy = null;
          G.addClass(p.el, 'is-locked');
          G.addClass(p.el, 'is-snap');
          setPos(i, tp.x, tp.y, 1);
          // gelegte Teile nach hinten, damit lose Teile darüber gezogen werden können
          var first = stage.querySelector('.jpiece');
          if (first && first !== p.el) stage.insertBefore(p.el, first);
          G.haptic(18);
          render();
          if (placedCount() === total) { finish(true); return; }
        } else {
          var cx = p.x + L.cw / 2, cy = p.y + L.ch / 2;
          var onBoard = cx > L.bx && cx < L.bx + L.bw && cy > L.by && cy < L.by + L.bh + L.ch * 0.25;
          if (onBoard && !cancel) {
            var k = keepOnScreen(p.x, p.y);
            if (k.x !== p.x || k.y !== p.y) setPos(i, k.x, k.y, 1);
            st.fx = (p.x - L.bx) / L.bw; st.fy = (p.y - L.by) / L.bh;
          } else {
            st.fx = null; st.fy = null;
            var sp = slotPos(st.slot);
            G.addClass(p.el, 'is-return');
            setPos(i, sp.x, sp.y, sp.s);
            w.setTimeout(function () { G.removeClass(p.el, 'is-return'); }, 320);
          }
        }
        store.save(state);
      }

      function finish(animate) {
        state.done = true;
        store.save(state);
        render();
        showDone();
        showWin([total + ' Teile', 'alle gelegt'], animate, boardEl);
      }

      build();
      if (state.done) finish(false);
      setResize(function () { if (!drag) build(); });
    }
  });
})();
