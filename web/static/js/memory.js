/* Memory: Karten mit Fotos, 3D-Flip, Sperre während des Vergleichs,
   Zugzähler, optionaler Timer und kurzes Aufdecken am Anfang (peek_ms). */
(function () {
  'use strict';
  var G = window.SAGame;
  if (!G) return;
  var h = G.h;
  var w = window;

  G.start('memory', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var IMGS = data.images || [];
    var PAIRS = Math.min(data.pairs || 0, IMGS.length);
    var state, cards = [], open = [], locked = false, timerId = 0, runningSince = 0;
    var grid, board, pairsEl, movesEl, timeEl, statusEl, winBox, peek = null, baseH = 0;

    function fresh() {
      var pick = G.shuffle(G.range(IMGS.length)).slice(0, PAIRS);
      var deck = G.shuffle(pick.concat(pick));
      return { v: 1, deck: deck, matched: [], moves: 0, elapsed: 0, done: false };
    }
    function valid(s) {
      if (!s || s.v !== 1 || !(s.deck instanceof Array) || s.deck.length !== PAIRS * 2) return false;
      var seen = {};
      for (var i = 0; i < s.deck.length; i++) {
        var d = s.deck[i];
        if (typeof d !== 'number' || d < 0 || d >= IMGS.length) return false;
        seen[d] = (seen[d] || 0) + 1;
      }
      for (var k in seen) if (seen.hasOwnProperty(k) && seen[k] !== 2) return false;
      return s.matched instanceof Array;
    }
    function isMatched(i) { return state.matched.indexOf(i) >= 0; }
    function matchedPairs() { return state.matched.length / 2; }
    function elapsed() { return state.elapsed + (runningSince ? Date.now() - runningSince : 0); }
    function save() {
      var s = { v: 1, deck: state.deck, matched: state.matched, moves: state.moves, elapsed: elapsed(), done: state.done };
      store.save(s);
    }

    /* ---------- Timer (pausiert, wenn die Seite im Hintergrund ist) ---------- */
    function startTimer() {
      if (runningSince || state.done) return;
      runningSince = Date.now();
      if (data.show_timer && !timerId) timerId = w.setInterval(renderStats, 500);
    }
    function stopTimer() {
      if (runningSince) { state.elapsed += Date.now() - runningSince; runningSince = 0; }
      if (timerId) { w.clearInterval(timerId); timerId = 0; }
    }
    document.addEventListener('visibilitychange', function () {
      if (document.hidden) { if (runningSince) { stopTimer(); save(); state.paused = true; } }
      else if (state.paused) { state.paused = false; startTimer(); }
    }, false);

    function renderStats() {
      pairsEl.textContent = 'Paare ' + matchedPairs() + '/' + PAIRS;
      if (movesEl) movesEl.textContent = G.count(state.moves, 'Zug', 'Züge');
      if (timeEl) timeEl.textContent = G.fmtTime(elapsed());
    }

    /* ---------- Raster: Spalten so wählen, dass bis 12 Paare ohne Scrollen passen ---------- */
    // Höhe nur beim ersten Aufbau bzw. bei Breitenänderung messen, damit die
    // ein-/ausfahrende Adressleiste die Karten nicht springen lässt.
    function layout() {
      var n = state.deck.length;
      var W = innerWidth(board) || 320;
      var gap = n > 16 ? 6 : 8;
      if (!baseH) baseH = w.innerHeight || 640;
      var H = Math.max(240, baseH * 0.62);
      var best = null;
      for (var c = 2; c <= 6; c++) {
        var r = Math.ceil(n / c);
        var sw = (W - (c - 1) * gap) / c;
        var sh = (H - (r - 1) * gap) / r;
        var size = n > 24 ? sw : Math.min(sw, sh);
        if (n > 24 && c < 6) continue; // >12 Paare: so viele Spalten wie möglich, Scrollen erlaubt
        // große Karten, aber volle Reihen bevorzugen (4×4 statt 3×6 mit Lücken)
        var rank = size * n / (r * c) - (c > r + 1 ? 6 : 0);
        if (!best || rank > best.rank) best = { c: c, size: size, rank: rank };
      }
      var s = Math.floor(Math.min(best.size, 132));
      grid.style.gridTemplateColumns = 'repeat(' + best.c + ', ' + s + 'px)';
      grid.style.gap = gap + 'px';
      grid.style.setProperty('--card', s + 'px');
    }

    // Breite ohne Innenabstand (clientWidth enthält das Padding von .mboard).
    function innerWidth(el) {
      var W = el.clientWidth;
      try {
        var cs = w.getComputedStyle(el);
        W -= (parseFloat(cs.paddingLeft) || 0) + (parseFloat(cs.paddingRight) || 0);
      } catch (e) { W -= 20; }
      return W;
    }

    /* ---------- Karten ---------- */
    function setFace(i, up) {
      var c = cards[i];
      G.toggleClass(c, 'is-up', up);
      c.setAttribute('aria-label', 'Karte ' + (i + 1) + (isMatched(i) ? ', gefunden' : up ? ', aufgedeckt' : ', verdeckt'));
    }

    function makeCard(i) {
      var back = h('span', { class: 'mcard__face mcard__back' + (data.card_back ? ' has-img' : '') });
      if (data.card_back) back.appendChild(h('img', { src: data.card_back, alt: '', draggable: 'false' }));
      else back.appendChild(h('span', { class: 'mcard__star', 'aria-hidden': 'true', text: '✦' }));
      var front = h('span', { class: 'mcard__face mcard__front' }, h('img', { src: IMGS[state.deck[i]], alt: '', draggable: 'false', decoding: 'async' }));
      var b = h('button', { type: 'button', class: 'mcard', 'data-card': String(i), 'data-pair': String(state.deck[i]) },
        h('span', { class: 'mcard__inner' }, [back, front]));
      b.addEventListener('click', function () { flip(i); }, false);
      return b;
    }

    function flip(i) {
      if (locked || state.done || isMatched(i) || open.indexOf(i) >= 0) return;
      if (peek) { peek.cancel(); peek = null; G.addClass(statusEl, 'is-faded'); } // schon losgelegt: kein Aufdecken mehr
      startTimer();
      open.push(i);
      setFace(i, true);
      G.haptic(8);
      if (open.length < 2) return;
      state.moves++;
      locked = true;
      var a = open[0], b = open[1];
      if (state.deck[a] === state.deck[b]) {
        state.matched.push(a, b);
        open = [];
        w.setTimeout(function () {
          G.addClass(cards[a], 'is-matched');
          G.addClass(cards[b], 'is-matched');
          setFace(a, true); setFace(b, true);
          locked = false;
          renderStats();
          if (matchedPairs() === PAIRS) win(true);
          else save();
        }, 320);
      } else {
        renderStats();
        save();
        w.setTimeout(function () {
          G.replayClass(cards[a], 'is-miss');
          G.replayClass(cards[b], 'is-miss');
        }, 450);
        w.setTimeout(function () {
          setFace(a, false); setFace(b, false);
          open = [];
          locked = false;
        }, 1000);
      }
      renderStats();
    }

    function win(animate) {
      state.done = true;
      stopTimer();
      save();
      renderStats();
      G.addClass(grid, 'is-done');
      var stats = [];
      stats.push(G.count(state.moves, 'Zug', 'Züge'));
      if (data.show_timer) stats.push(G.fmtTime(state.elapsed) + ' min');
      var panel = G.winPanel({
        title: state.moves <= PAIRS + 2 ? 'Wahnsinn!' : 'Alle Paare gefunden!',
        stats: stats,
        text: data.win_text,
        onReplay: function () { store.clear(); state = fresh(); build(true); }
      });
      G.clear(winBox);
      winBox.appendChild(panel);
      if (animate) {
        G.celebrate(grid, true);
        w.setTimeout(function () { G.scrollIntoView(panel); }, 600);
      }
    }

    function build(isReplay) {
      stopTimer();
      if (peek) { peek.cancel(); peek = null; }
      open = []; locked = false; cards = [];
      G.clear(root);
      pairsEl = h('span', { class: 'gchip', 'data-pairs': '' });
      movesEl = data.show_moves ? h('span', { class: 'gchip gchip--gold', 'data-moves': '' }) : null;
      timeEl = data.show_timer ? h('span', { class: 'gchip gchip--time', 'data-timer': '', role: 'timer' }) : null;
      statusEl = h('p', { class: 'game__status', 'aria-live': 'polite' });
      var bar = h('div', { class: 'gbar' }, [pairsEl, movesEl, timeEl]);
      grid = h('div', { class: 'mgrid', role: 'group', 'aria-label': 'Memory-Karten' });
      for (var i = 0; i < state.deck.length; i++) {
        var c = makeCard(i);
        cards.push(c);
        grid.appendChild(c);
      }
      winBox = h('div', { class: 'game__win' });
      root.appendChild(bar);
      root.appendChild(statusEl);
      board = h('div', { class: 'mboard' }, grid);
      root.appendChild(board);
      root.appendChild(winBox);
      for (var j = 0; j < state.matched.length; j++) G.addClass(cards[state.matched[j]], 'is-matched');
      for (var k = 0; k < cards.length; k++) setFace(k, isMatched(k));
      layout();
      renderStats();
      if (state.done) { win(false); return; }
      if (isReplay) G.scrollIntoView(root);
      // Kurz alle Karten zeigen (nur bei einem frischen Spiel) – erst nach der
      // Öffnungsanimation und wenn das Spielfeld zu sehen ist, sonst verpufft es.
      if (data.peek_ms > 0 && state.moves === 0 && state.matched.length === 0) {
        statusEl.textContent = 'Gleich siehst du alle Karten kurz.';
        peek = G.whenSeen(board, function () {
          peek = null;
          locked = true;
          statusEl.textContent = 'Gut merken …';
          for (var i = 0; i < cards.length; i++) setFace(i, true);
          w.setTimeout(function () {
            for (var i = 0; i < cards.length; i++) setFace(i, false);
            locked = false;
            statusEl.textContent = 'Los geht’s!';
            // Text nur ausblenden, nicht entfernen: das Spielfeld soll beim Tippen nicht springen.
            w.setTimeout(function () { G.addClass(statusEl, 'is-faded'); }, 1600);
          }, 450 + data.peek_ms);
        }, 0.7);
      }
    }

    // Bilder vorladen, damit beim Umdrehen nichts leer bleibt.
    for (var p = 0; p < IMGS.length; p++) { var im = new Image(); im.src = IMGS[p]; }
    if (data.card_back) { var cb = new Image(); cb.src = data.card_back; }

    state = store.load();
    if (!valid(state)) state = fresh();
    state.matched = state.matched.filter(function (x) { return typeof x === 'number' && x >= 0 && x < state.deck.length; });
    build(false);
    G.onWidthChange(root, function () { baseH = 0; layout(); }, 120);
  });
})();
