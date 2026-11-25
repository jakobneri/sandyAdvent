/* Wordle mit eigenen Wörtern: mehrere Runden, variable Länge (3–12),
   Umlaute und ß, Bildschirm-Tastatur (QWERTZ + ÄÖÜß) und echte Tastatur. */
(function () {
  'use strict';
  var G = window.SAGame, L = window.WordleLogic;
  if (!G || !L) return;
  var h = G.h;
  var w = window, doc = document;
  var ROWS = ['QWERTZUIOP', 'ASDFGHJKL', 'ÄÖÜß', 'YXCVBNM'];
  var BACK = '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M9 5h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H9l-6-7z"/><path d="M12.5 9.5l5 5m0-5l-5 5"/></svg>';

  G.start('wordle', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var WORDS = data.words || [];
    var MAX = data.max_guesses || 6;
    var state, buffer = [], busy = false;
    var boardEl, rowsEls = [], keyEls = {}, msgEl, headEl, hintEl, endBox, kbd;

    function fresh() {
      var rounds = [];
      for (var i = 0; i < WORDS.length; i++) rounds.push({ guesses: [], hint: false, status: 'playing' });
      return { v: 1, round: 0, rounds: rounds, done: false };
    }
    function valid(s) {
      return s && s.v === 1 && s.rounds instanceof Array && s.rounds.length === WORDS.length &&
        typeof s.round === 'number' && s.round >= 0 && s.round < WORDS.length;
    }
    function save() { store.save(state); }
    function cur() { return state.rounds[state.round]; }
    function answer() { return L.normalize(WORDS[state.round].word); }
    function len() { return L.letters(answer()).length; }

    /* ---------- Meldungen ---------- */
    var msgT = 0;
    function say(text, keep) {
      w.clearTimeout(msgT);
      msgEl.textContent = text;
      G.replayClass(msgEl, 'is-on');
      if (!keep) msgT = w.setTimeout(function () { G.removeClass(msgEl, 'is-on'); }, 2200);
    }

    /* ---------- Aufbau ---------- */
    // Kachelgröße nach Breite und – damit alle Versuche über die Bildschirm-Tastatur
    // passen – nach Höhe. Die Höhe wird nur beim Aufbau/bei Breitenänderung gemessen
    // (ein-/ausfahrende Adressleiste soll nichts verschieben).
    var baseH = 0;
    function sizeBoard() {
      var n = len();
      var W = root.clientWidth || 320;
      var gap = n > 8 ? 4 : 6;
      var t = Math.floor((W - 8 - (n - 1) * gap) / n);
      if (!baseH) baseH = w.innerHeight || 640;
      var kh = kbd && !kbd.hidden ? kbd.offsetHeight : 0;
      var byH = Math.floor((baseH - kh - 150 - (MAX - 1) * gap) / MAX);
      t = Math.min(t, Math.max(byH, 34));
      t = Math.max(20, Math.min(t, 58));
      boardEl.style.setProperty('--tile', t + 'px');
      boardEl.style.setProperty('--gap', gap + 'px');
    }

    function tile(i) {
      return h('span', { class: 'wtile', 'data-i': String(i) }, h('span', { class: 'wtile__in' }));
    }

    function buildRound() {
      G.clear(root);
      buffer = [];
      busy = false;
      var r = cur();
      var n = len();
      headEl = h('div', { class: 'gbar' }, [
        h('span', { class: 'gchip', 'data-round': '', 'aria-label': 'Runde ' + (state.round + 1) + ' von ' + WORDS.length, text: 'Runde ' + (state.round + 1) + '/' + WORDS.length }),
        h('span', { class: 'gchip gchip--gold', text: G.count(n, 'Buchstabe', 'Buchstaben') }),
        h('span', { class: 'gchip', 'data-tries': '', text: tryText() })
      ]);
      root.appendChild(headEl);
      var hint = WORDS[state.round].hint;
      hintEl = null;
      if (hint) {
        hintEl = h('div', { class: 'whint' });
        renderHint();
        root.appendChild(hintEl);
      }
      boardEl = h('div', { class: 'wboard', role: 'grid', 'aria-label': 'Rateversuche', 'data-length': String(n) });
      rowsEls = [];
      for (var g = 0; g < MAX; g++) {
        var row = h('div', { class: 'wrow', role: 'row' });
        for (var i = 0; i < n; i++) row.appendChild(tile(i));
        rowsEls.push(row);
        boardEl.appendChild(row);
      }
      root.appendChild(boardEl);
      msgEl = h('p', { class: 'wmsg', role: 'status', 'aria-live': 'polite' });
      root.appendChild(msgEl);
      endBox = h('div', { class: 'game__win' });
      root.appendChild(endBox);
      buildKeyboard();
      sizeBoard();
      // gespeicherte Versuche ohne Animation wiederherstellen
      var keys = {};
      for (var k = 0; k < r.guesses.length; k++) {
        var res = L.score(r.guesses[k], answer());
        paintRow(k, r.guesses[k], res, false);
        keys = L.mergeKeys(keys, r.guesses[k], res);
      }
      paintKeys(keys);
      renderTyping();
      if (r.status !== 'playing') roundOver(false);
    }

    function tryText() {
      var left = MAX - cur().guesses.length;
      return G.count(left, 'Versuch', 'Versuche');
    }

    function renderHint() {
      G.clear(hintEl);
      var hint = WORDS[state.round].hint;
      if (cur().hint) {
        hintEl.appendChild(h('p', { class: 'whint__text' }, [h('strong', { text: 'Tipp: ' }), hint]));
      } else {
        hintEl.appendChild(G.button('Tipp anzeigen', {
          variant: 'small',
          onclick: function () { cur().hint = true; save(); renderHint(); }
        }));
      }
    }

    function buildKeyboard() {
      keyEls = {};
      kbd = h('div', { class: 'wkbd', role: 'group', 'aria-label': 'Tastatur' });
      for (var r = 0; r < ROWS.length; r++) {
        var row = h('div', { class: 'wkbd__row' + (r === 2 ? ' wkbd__row--umlaut' : '') });
        if (r === 3) row.appendChild(h('button', { type: 'button', class: 'wkey wkey--wide wkey--enter', tabindex: '-1', 'data-key': 'Enter', text: 'Prüfen' }));
        var chars = ROWS[r].split('');
        for (var i = 0; i < chars.length; i++) {
          var b = h('button', { type: 'button', class: 'wkey', tabindex: '-1', 'data-key': chars[i], text: chars[i] });
          keyEls[chars[i]] = b;
          row.appendChild(b);
        }
        if (r === 3) {
          var bs = h('button', { type: 'button', class: 'wkey wkey--wide wkey--back', tabindex: '-1', 'data-key': 'Backspace', 'aria-label': 'Löschen' });
          bs.innerHTML = BACK; // statisches SVG
          row.appendChild(bs);
        }
        kbd.appendChild(row);
      }
      // Tasten nie fokussieren: sonst löst Leertaste/Enter auf einer echten Tastatur die zuletzt
      // getippte Bildschirm-Taste erneut aus.
      kbd.addEventListener('mousedown', function (e) { e.preventDefault(); }, false);
      kbd.addEventListener('click', function (e) {
        var t = e.target;
        while (t && t !== kbd && !t.getAttribute('data-key')) t = t.parentNode;
        if (!t || t === kbd) return;
        press(t.getAttribute('data-key'));
      }, false);
      root.appendChild(kbd);
    }

    /* ---------- Eingabe ---------- */
    function press(key) {
      if (busy || state.done || cur().status !== 'playing') return;
      if (key === 'Enter') return submit();
      if (key === 'Backspace') { buffer.pop(); renderTyping(); return; }
      var ch = L.normalize(key);
      if (!L.isLetter(ch)) return;
      if (buffer.length >= len()) { G.replayClass(rowsEls[cur().guesses.length], 'is-shake'); return; }
      buffer.push(ch);
      renderTyping(true);
      keepRowVisible();
    }

    // Die Zeile, in die gerade getippt wird, über der (klebenden) Bildschirm-Tastatur halten.
    function keepRowVisible() {
      var row = rowsEls[cur().guesses.length];
      if (!row || !row.getBoundingClientRect) return;
      var r = row.getBoundingClientRect();
      var limit = w.innerHeight || 640;
      if (kbd && !kbd.hidden) limit = Math.min(limit, kbd.getBoundingClientRect().top);
      limit -= 10;
      var dy = 0;
      if (r.bottom > limit) dy = r.bottom - limit;
      else if (r.top < 8) dy = r.top - 8;
      if (dy) {
        try { w.scrollBy({ top: dy, left: 0, behavior: G.reduceMotion ? 'auto' : 'smooth' }); } catch (e) { w.scrollBy(0, dy); }
      }
    }

    function renderTyping(pop) {
      var row = rowsEls[cur().guesses.length];
      if (!row) return;
      var tiles = row.children;
      for (var i = 0; i < tiles.length; i++) {
        var ch = buffer[i] || '';
        tiles[i].firstChild.textContent = ch;
        G.toggleClass(tiles[i], 'is-filled', !!ch);
        if (pop && i === buffer.length - 1) G.replayClass(tiles[i], 'is-pop');
      }
      G.addClass(row, 'is-active');
    }

    function paintRow(idx, guess, res, animate, done) {
      var row = rowsEls[idx];
      var g = L.letters(guess);
      var tiles = row.children;
      var step = G.reduceMotion || !animate ? 0 : Math.min(260, Math.floor(1500 / g.length));
      G.removeClass(row, 'is-active');
      for (var i = 0; i < tiles.length; i++) {
        (function (i) {
          tiles[i].firstChild.textContent = g[i];
          tiles[i].setAttribute('aria-label', g[i] + ': ' + (res[i] === 'correct' ? 'richtig' : res[i] === 'present' ? 'woanders' : 'nicht enthalten'));
          if (!step) {
            tiles[i].className = 'wtile is-filled is-' + res[i];
            return;
          }
          w.setTimeout(function () { G.addClass(tiles[i], 'is-flip'); }, i * step);
          w.setTimeout(function () { tiles[i].className = 'wtile is-filled is-flip is-' + res[i]; }, i * step + 180);
        })(i);
      }
      if (done) w.setTimeout(done, step ? (tiles.length - 1) * step + 420 : 0);
    }

    function paintKeys(keys) {
      for (var k in keyEls) {
        if (!keyEls.hasOwnProperty(k)) continue;
        var st = keys[k];
        keyEls[k].className = 'wkey' + (st ? ' is-' + st : '');
      }
    }

    function currentKeys() {
      var keys = {}, r = cur();
      for (var k = 0; k < r.guesses.length; k++) keys = L.mergeKeys(keys, r.guesses[k], L.score(r.guesses[k], answer()));
      return keys;
    }

    function submit() {
      var guess = buffer.join('');
      var chk = L.check(guess, len());
      if (!chk.ok) {
        G.replayClass(rowsEls[cur().guesses.length], 'is-shake');
        G.haptic([20, 30, 20]);
        say(chk.reason === 'short' ? 'Noch ' + G.count(chk.missing, 'Buchstabe', 'Buchstaben') + ' …' : 'Das passt nicht.');
        return;
      }
      var r = cur();
      var res = L.score(guess, answer());
      var idx = r.guesses.length;
      r.guesses.push(guess);
      var won = L.isWin(res);
      if (won) r.status = 'won';
      else if (r.guesses.length >= MAX) r.status = 'lost';
      if (r.status !== 'playing') {
        var allOver = true;
        for (var i = 0; i < state.rounds.length; i++) if (state.rounds[i].status === 'playing') allOver = false;
        if (allOver) state.done = true;
      }
      save();
      buffer = [];
      busy = true;
      paintRow(idx, guess, res, true, function () {
        busy = false;
        paintKeys(currentKeys());
        var tries = headEl.querySelector('[data-tries]');
        if (tries) tries.textContent = tryText();
        if (r.status !== 'playing') roundOver(true);
        else { renderTyping(); keepRowVisible(); }
      });
    }

    /* ---------- Rundenende ---------- */
    function roundOver(animate) {
      var r = cur();
      kbd.hidden = true; // Tastatur weg, Platz für Rundenende/Gewinn
      var last = state.round === WORDS.length - 1;
      if (r.status === 'won') {
        G.replayClass(rowsEls[r.guesses.length - 1], 'is-win');
        if (animate) G.haptic(25);
      }
      if (last || state.done) { finish(animate); return; }
      var word = answer();
      var box = h('div', { class: 'wround ' + (r.status === 'won' ? 'is-won' : 'is-lost') }, [
        h('p', { class: 'wround__title', text: r.status === 'won' ? roundPraise(r.guesses.length) : 'Knapp daneben!' }),
        r.status === 'won' ? h('p', { class: 'wround__sub', text: 'Gelöst ' + triesLabel(r.guesses.length) + '.' })
          : h('p', { class: 'wround__sub' }, ['Das Wort war ', h('strong', { text: word }), '.']),
        G.button('Nächstes Wort', {
          onclick: function () {
            state.round++;
            save();
            buildRound();
            G.scrollIntoView(root);
          }
        })
      ]);
      G.clear(endBox);
      endBox.appendChild(box);
      if (animate) {
        if (r.status === 'won') G.celebrate(boardEl, false);
        G.scrollIntoView(box);
      }
    }

    function roundPraise(n) {
      if (n === 1) return 'Wie bitte?! Beim ersten Mal!';
      if (n <= 3) return 'Stark!';
      return 'Geknackt!';
    }
    function triesLabel(n) { return n === 1 ? 'im ersten Versuch' : 'in ' + n + ' Versuchen'; }

    function finish(animate) {
      var solved = 0;
      var list = h('ul', { class: 'wsum' });
      for (var i = 0; i < WORDS.length; i++) {
        var r = state.rounds[i];
        if (r.status === 'won') solved++;
        list.appendChild(h('li', { class: 'wsum__item ' + (r.status === 'won' ? 'is-won' : 'is-lost') }, [
          h('span', { class: 'wsum__word', text: L.normalize(WORDS[i].word) }),
          h('span', { class: 'wsum__info', text: r.status === 'won' ? triesLabel(r.guesses.length) : 'nicht erraten' })
        ]));
      }
      var all = solved === WORDS.length;
      var panel = G.winPanel({
        title: all ? (WORDS.length > 1 ? 'Alle Wörter geknackt!' : 'Geknackt!') : 'Geschafft!',
        stats: [solved + ' von ' + G.count(WORDS.length, 'Wort', 'Wörtern') + ' erraten'],
        extra: list,
        text: data.win_text,
        onReplay: function () {
          store.clear();
          state = fresh();
          buildRound();
          G.scrollIntoView(root);
        }
      });
      G.clear(endBox);
      endBox.appendChild(panel);
      if (animate) {
        G.celebrate(boardEl, all);
        w.setTimeout(function () { G.scrollIntoView(panel); }, 400);
      }
    }

    /* ---------- echte Tastatur ---------- */
    doc.addEventListener('keydown', function (e) {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      var t = e.target;
      if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return;
      var k = e.key;
      if ((k === ' ' || k === 'Spacebar') && t && t.getAttribute && t.getAttribute('data-key')) {
        e.preventDefault(); // Leertaste soll keine Bildschirm-Taste auslösen
        return;
      }
      if (k === 'Enter') {
        if (t && t.tagName === 'BUTTON' && !t.getAttribute('data-key')) return; // Buttons selbst auslösen lassen
        e.preventDefault();
        press('Enter');
      } else if (k === 'Backspace') {
        e.preventDefault();
        press('Backspace');
      } else if (k && (k.length === 1 || (k.length === 2 && k.charAt(1) === '̈'))) {
        var ch = L.normalize(k);
        if (L.isLetter(ch)) { e.preventDefault(); press(ch); }
      }
    }, false);
    doc.addEventListener('keyup', function (e) {
      var t = e.target;
      if ((e.key === ' ' || e.key === 'Spacebar') && t && t.getAttribute && t.getAttribute('data-key')) e.preventDefault();
    }, false);

    G.onWidthChange(root, function () { baseH = 0; sizeBoard(); }, 120);

    state = store.load();
    if (!valid(state)) state = fresh();
    if (state.done) {
      state.round = WORDS.length - 1;
    }
    buildRound();
  });
})();
