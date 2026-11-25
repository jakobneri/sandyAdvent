/* Kreuzworträtsel (Türchen 16): Raster aus dem Raster-JSON, Hinweislisten
   „Waagerecht“/„Senkrecht“, klebende Hinweisleiste über der Bildschirm-Tastatur.
   Eingabe über EIN echtes, unsichtbares <input> über dem aktiven Feld – so öffnen
   iOS und Android die Tastatur direkt im Tipp-Ereignis. Buchstaben kommen über
   input-/composition-Ereignisse und einen Wertevergleich (Android schickt bei
   Kompositionen keyCode 229), nicht über keydown. Pfeile/Tab/Enter für echte Tastaturen. */
(function () {
  'use strict';
  var G = window.SAGame, X = window.CrosswordLogic;
  if (!G || !X) return;
  var h = G.h;
  var w = window, doc = document;
  var SENT = ' '; // Platzhalter im Eingabefeld: Löschen auf "leerem" Feld bleibt erkennbar
  var DIR = { across: 'waagerecht', down: 'senkrecht' };

  G.start('crossword', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var m = X.model(data);
    var saved = store.load();
    var st = X.state(m, saved && saved.v === 1 && saved.w === m.W && saved.h === m.H ? saved : null);
    var done = false, engaged = false;
    var boardEl, scrollEl, cellEls = [], input, barEl, barText, spacer, statusEl, chipEl, cluesEl, clueEls = [], winBox, actions;
    var composing = false, last = SENT;

    function save() {
      var s = X.serialize(st);
      s.v = 1; s.w = m.W; s.h = m.H; s.done = done;
      store.save(s);
    }

    /* ---------- Aufbau ---------- */
    function build() {
      G.clear(root);
      cellEls = [];
      clueEls = [];
      chipEl = h('span', { class: 'gchip gchip--gold' });
      root.appendChild(h('div', { class: 'gbar' }, [
        chipEl,
        h('span', { class: 'gchip', text: G.count(m.entries.length, 'Wort', 'Wörter') })
      ]));

      boardEl = h('div', { class: 'cw-board is-pre', role: 'grid', 'aria-label': 'Kreuzworträtsel, ' + m.W + ' mal ' + m.H + ' Felder' });
      for (var r = 0; r < m.H; r++) {
        for (var c = 0; c < m.W; c++) {
          var i = r * m.W + c;
          if (!X.isOpen(m, i)) {
            cellEls.push(null);
            boardEl.appendChild(h('div', { class: 'cw-block', 'aria-hidden': 'true' }));
            continue;
          }
          var cell = h('div', { class: 'cw-cell', 'data-i': String(i), role: 'gridcell' }, [
            m.num[i] ? h('span', { class: 'cw-num', 'aria-hidden': 'true', text: String(m.num[i]) }) : null,
            h('span', { class: 'cw-letter' })
          ]);
          cell.style.setProperty('--d', ((r + c) * 22) + 'ms');
          cellEls.push(cell);
          boardEl.appendChild(cell);
        }
      }
      input = h('input', {
        class: 'cw-input', type: 'text', value: SENT, autocapitalize: 'characters', autocomplete: 'off',
        autocorrect: 'off', spellcheck: 'false', inputmode: 'text', enterkeyhint: 'next',
        'aria-label': 'Buchstabe eingeben', tabindex: '-1'
      });
      boardEl.appendChild(input);
      scrollEl = h('div', { class: 'cw-scroll' }, boardEl);
      root.appendChild(scrollEl);

      statusEl = h('p', { class: 'game__status', role: 'status', 'aria-live': 'polite' });
      root.appendChild(statusEl);

      actions = h('div', { class: 'cw-actions' });
      if (data.allow_check) actions.appendChild(G.button('Prüfen', { variant: 'small', onclick: onCheck }));
      if (data.allow_reveal_letter) actions.appendChild(G.button('Buchstabe aufdecken', { variant: 'small', onclick: onReveal }));
      if (actions.firstChild) root.appendChild(actions);

      cluesEl = h('div', { class: 'cw-clues' });
      var lists = { across: h('ol', { class: 'cw-list' }), down: h('ol', { class: 'cw-list' }) };
      for (var k = 0; k < m.entries.length; k++) {
        var e = m.entries[k];
        var b = h('button', { type: 'button', class: 'cw-clue', 'data-e': String(k) }, [
          h('span', { class: 'cw-clue__n', text: String(e.number) }),
          h('span', { class: 'cw-clue__t', text: e.clue }),
          h('span', { class: 'cw-clue__len', text: '(' + e.cells.length + ')' })
        ]);
        clueEls.push(b);
        lists[e.dir].appendChild(h('li', null, b));
      }
      if (lists.across.firstChild) cluesEl.appendChild(h('section', { class: 'cw-sec' }, [h('h2', { class: 'cw-sec__t', text: 'Waagerecht' }), lists.across]));
      if (lists.down.firstChild) cluesEl.appendChild(h('section', { class: 'cw-sec' }, [h('h2', { class: 'cw-sec__t', text: 'Senkrecht' }), lists.down]));
      root.appendChild(cluesEl);

      winBox = h('div', { class: 'game__win' });
      root.appendChild(winBox);
      spacer = h('div', { class: 'cw-spacer', 'aria-hidden': 'true' });
      root.appendChild(spacer);

      if (!barEl) {
        barText = h('p', { class: 'cw-bar__text', 'aria-live': 'polite' });
        barEl = h('div', { class: 'cw-bar', hidden: true }, [
          h('button', { type: 'button', class: 'cw-bar__nav', 'data-nav': '-1', 'aria-label': 'Vorheriger Hinweis', text: '‹' }),
          barText,
          h('button', { type: 'button', class: 'cw-bar__nav', 'data-nav': '1', 'aria-label': 'Nächster Hinweis', text: '›' })
        ]);
        doc.body.appendChild(barEl);
        // Antippen der Leiste darf die Tastatur nicht schließen
        barEl.addEventListener('mousedown', function (e) { e.preventDefault(); }, false);
        barEl.addEventListener('click', function (e) {
          var t = e.target;
          while (t && t !== barEl && !t.getAttribute('data-nav')) t = t.parentNode;
          if (t && t !== barEl) X.nextEntry(m, st, parseInt(t.getAttribute('data-nav'), 10));
          else X.select(m, st, st.cell); // Leiste antippen = Richtung wechseln
          engage();
        }, false);
      }

      // Tippen aufs Brett/die Hinweise soll das Eingabefeld nicht erst entfokussieren
      // (sonst klappt die Android-Tastatur kurz zu und wieder auf).
      var keepFocus = function (e) { if (doc.activeElement === input) e.preventDefault(); };
      boardEl.addEventListener('mousedown', keepFocus, false);
      cluesEl.addEventListener('mousedown', keepFocus, false);
      boardEl.addEventListener('click', onBoardClick, false);
      cluesEl.addEventListener('click', onClueClick, false);
      input.addEventListener('compositionstart', function () { composing = true; }, false);
      input.addEventListener('compositionend', function () { composing = false; takeValue(); resetInput(); }, false);
      input.addEventListener('input', function (e) {
        takeValue();
        if (!composing && !e.isComposing) resetInput();
      }, false);
      input.addEventListener('keydown', onKey, false);
      input.addEventListener('focus', function () { placeInput(); G.addClass(root, 'is-typing'); setCaret(); }, false);
      input.addEventListener('blur', function () { G.removeClass(root, 'is-typing'); }, false);

      sizeBoard();
      G.afterIntro(function () {
        G.removeClass(boardEl, 'is-pre');
        G.addClass(boardEl, 'is-live');
      });
      render();
      if (done) finish(false);
    }

    /* ---------- Größe ---------- */
    function sizeBoard() {
      var avail = scrollEl.clientWidth || 320;
      var gap = m.W > 11 ? 2 : 3, pad = 6;
      var cell = Math.floor((avail - (m.W - 1) * gap - 2 * pad) / m.W);
      cell = Math.max(24, Math.min(cell, 46));
      boardEl.style.setProperty('--cell', cell + 'px');
      boardEl.style.setProperty('--gap', gap + 'px');
      boardEl.style.setProperty('--cols', String(m.W));
      G.toggleClass(scrollEl, 'is-scrollable', m.W * cell + (m.W - 1) * gap + 2 * pad > avail + 1);
      placeInput();
    }

    /* ---------- Darstellung ---------- */
    function render() {
      var cur = X.current(m, st);
      var inEntry = {};
      if (cur && engaged && !done) for (var k = 0; k < cur.cells.length; k++) inEntry[cur.cells[k]] = true;
      for (var i = 0; i < cellEls.length; i++) {
        var el = cellEls[i];
        if (!el) continue;
        var letter = st.letters[i];
        var span = el.lastChild;
        if (span.textContent !== letter) span.textContent = letter;
        var cls = 'cw-cell' + (letter ? ' is-filled' : '') + (inEntry[i] ? ' is-entry' : '') +
          (engaged && !done && i === st.cell ? ' is-cell' : '') + (st.wrong[i] ? ' is-wrong' : '') +
          (st.revealed[i] ? ' is-revealed' : '') + (done ? ' is-done' : '');
        if (el.className.replace(/ is-win/, '') !== cls) el.className = cls + (/ is-win/.test(el.className) ? ' is-win' : '');
        el.setAttribute('aria-label', (m.num[i] ? m.num[i] + ', ' : '') + (letter || 'leer'));
      }
      for (var e = 0; e < clueEls.length; e++) {
        G.toggleClass(clueEls[e], 'is-active', !!(cur && engaged && !done && cur.i === e));
        G.toggleClass(clueEls[e], 'is-solved', X.entrySolved(m, st, e));
      }
      var solved = X.solvedEntries(m, st);
      chipEl.textContent = solved + ' von ' + G.count(m.entries.length, 'Wort', 'Wörtern') + ' gelöst';
      if (cur) {
        G.clear(barText);
        barText.appendChild(h('strong', { class: 'cw-bar__n', text: cur.number + ' ' + DIR[cur.dir] }));
        barText.appendChild(doc.createTextNode(' ' + cur.clue + ' '));
        barText.appendChild(h('span', { class: 'cw-bar__len', text: '(' + cur.cells.length + ')' }));
      }
      barEl.hidden = !engaged || done;
      spacer.style.height = barEl.hidden ? '0px' : (barEl.offsetHeight + 12) + 'px';
      placeBar();
      placeInput();
    }

    /* ---------- Auswahl + Fokus ---------- */
    function engage() {
      engaged = true;
      render();
      focusInput();
      ensureVisible();
    }

    function onBoardClick(e) {
      if (done) return;
      var t = e.target;
      while (t && t !== boardEl && !t.getAttribute('data-i')) t = t.parentNode;
      if (!t || t === boardEl) return;
      var i = parseInt(t.getAttribute('data-i'), 10);
      // erster Tipp aufs vorausgewählte Feld behält die Richtung, danach wechselt sie
      if (engaged || i !== st.cell) X.select(m, st, i);
      engage();
    }

    function onClueClick(e) {
      if (done) return;
      var t = e.target;
      while (t && t !== cluesEl && !t.getAttribute('data-e')) t = t.parentNode;
      if (!t || t === cluesEl) return;
      X.selectEntry(m, st, parseInt(t.getAttribute('data-e'), 10));
      engage();
    }

    // Muss synchron im Tipp-Ereignis passieren, sonst öffnet iOS keine Tastatur.
    function focusInput() {
      if (done) return;
      placeInput();
      if (doc.activeElement === input) return;
      try { input.focus({ preventScroll: true }); } catch (err) { input.focus(); }
    }

    function placeInput() {
      var el = cellEls[st.cell];
      if (!el || !input) return;
      input.style.left = el.offsetLeft + 'px';
      input.style.top = el.offsetTop + 'px';
      input.style.width = el.offsetWidth + 'px';
      input.style.height = el.offsetHeight + 'px';
    }

    function setCaret() {
      try { input.setSelectionRange(input.value.length, input.value.length); } catch (err) {}
    }

    function resetInput() {
      if (input.value !== SENT) input.value = SENT;
      last = SENT;
      setCaret();
    }

    // Neuen Wert des Eingabefelds mit dem letzten vergleichen und anwenden.
    function takeValue() {
      var d = X.diffValue(last, input.value);
      last = input.value;
      var changed = false;
      for (var k = 0; k < d.del; k++) changed = X.backspace(m, st) || changed;
      var letters = X.lettersOf(d.ins);
      for (k = 0; k < letters.length; k++) changed = X.input(m, st, letters[k]) || changed;
      if (changed) afterChange(letters.length > 0);
    }

    function onKey(e) {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      var k = e.key;
      var moves = { ArrowLeft: [0, -1], ArrowRight: [0, 1], ArrowUp: [-1, 0], ArrowDown: [1, 0] };
      if (moves[k]) {
        e.preventDefault();
        X.move(m, st, moves[k][0], moves[k][1]);
      } else if (k === 'Tab' || k === 'Enter') {
        e.preventDefault();
        X.nextEntry(m, st, e.shiftKey ? -1 : 1);
      } else if (k === ' ' || k === 'Spacebar') {
        e.preventDefault();
        X.select(m, st, st.cell); // Leertaste wechselt die Richtung
      } else {
        return; // Buchstaben + Löschen kommen über das input-Ereignis
      }
      render();
      ensureVisible();
    }

    function afterChange(typed) {
      save();
      render();
      if (typed) {
        var el = cellEls[st.cell];
        if (el) G.replayClass(el, 'is-pop');
      }
      ensureVisible();
      if (X.isSolved(m, st)) { finish(true); return; }
      if (X.isFilled(m, st)) {
        say(data.allow_check ? 'Alles ausgefüllt – aber irgendwo stimmt noch was nicht. „Prüfen“ zeigt dir, wo.'
          : 'Alles ausgefüllt – aber irgendwo hakt es noch.', true);
      }
    }

    /* ---------- Hinweisleiste über der Tastatur ---------- */
    // position: fixed klebt am Layout-Viewport; die Bildschirm-Tastatur verkleinert nur
    // den Visual Viewport (iOS, Android Chrome) – deshalb um die Differenz hochschieben.
    function keyboardOffset() {
      var vv = w.visualViewport;
      if (!vv) return 0;
      return Math.max(0, (w.innerHeight || 0) - (vv.height + vv.offsetTop));
    }
    function placeBar() {
      if (!barEl || barEl.hidden) return;
      var off = keyboardOffset();
      barEl.style.transform = off > 1 ? 'translateY(' + (-off) + 'px)' : '';
      G.toggleClass(barEl, 'is-raised', off > 1);
    }

    // Aktives Feld zwischen oberem Rand und Hinweisleiste halten.
    function ensureVisible() {
      var el = cellEls[st.cell];
      if (!el || !el.getBoundingClientRect) return;
      // waagerecht im scrollbaren Brett
      if (scrollEl.scrollWidth > scrollEl.clientWidth + 1) {
        var l = el.offsetLeft + boardEl.offsetLeft, r = l + el.offsetWidth;
        if (l < scrollEl.scrollLeft + 8) scrollEl.scrollLeft = Math.max(0, l - 24);
        else if (r > scrollEl.scrollLeft + scrollEl.clientWidth - 8) scrollEl.scrollLeft = r - scrollEl.clientWidth + 24;
      }
      var rect = el.getBoundingClientRect();
      var vv = w.visualViewport;
      var top = vv ? vv.offsetTop : 0;
      var bottom = vv ? vv.offsetTop + vv.height : (w.innerHeight || 640);
      if (barEl && !barEl.hidden) bottom = Math.min(bottom, barEl.getBoundingClientRect().top);
      var dy = 0;
      if (rect.bottom > bottom - 10) dy = rect.bottom - bottom + 10 + rect.height;
      else if (rect.top < top + 10) dy = rect.top - top - 10 - rect.height;
      if (dy) {
        try { w.scrollBy({ top: dy, left: 0, behavior: 'auto' }); } catch (err) { w.scrollBy(0, dy); }
      }
    }

    var vvT = 0;
    function onViewport() {
      placeBar();
      w.clearTimeout(vvT);
      // nach dem Ein-/Ausfahren der Tastatur das aktive Feld sichtbar halten
      vvT = w.setTimeout(function () { placeBar(); if (doc.activeElement === input) ensureVisible(); }, 120);
    }
    if (w.visualViewport) {
      w.visualViewport.addEventListener('resize', onViewport, false);
      w.visualViewport.addEventListener('scroll', placeBar, false);
    }
    w.addEventListener('scroll', placeBar, { passive: true });

    /* ---------- Prüfen / Aufdecken ---------- */
    var msgT = 0;
    function say(text, keep) {
      w.clearTimeout(msgT);
      statusEl.textContent = text;
      G.removeClass(statusEl, 'is-faded');
      if (!keep) msgT = w.setTimeout(function () { G.addClass(statusEl, 'is-faded'); }, 3200);
    }

    function onCheck() {
      if (done) return;
      var wrong = X.check(m, st);
      var c = X.counts(m, st);
      render();
      if (wrong.length) {
        say(G.count(wrong.length, 'Buchstabe ist', 'Buchstaben sind') + ' noch falsch – rot markiert.');
        G.haptic([20, 30, 20]);
        for (var i = 0; i < wrong.length; i++) G.replayClass(cellEls[wrong[i]], 'is-shake');
      } else if (c.filled === 0) {
        say('Noch ist alles leer – tipp auf ein Kästchen und leg los.');
      } else {
        say('Alles richtig bisher! Noch ' + G.count(c.open - c.filled, 'Kästchen', 'Kästchen') + ' frei.');
      }
      if (engaged) focusInput();
    }

    function onReveal() {
      if (done) return;
      engaged = true;
      X.reveal(m, st);
      afterChange(true);
      focusInput();
    }

    /* ---------- Geschafft ---------- */
    function finish(animate) {
      done = true;
      save();
      if (doc.activeElement === input) input.blur();
      render();
      if (actions) actions.hidden = true;
      G.clear(statusEl);
      var revealed = 0;
      for (var i = 0; i < st.revealed.length; i++) if (st.revealed[i]) revealed++;
      for (i = 0; i < cellEls.length; i++) if (cellEls[i] && animate) G.addClass(cellEls[i], 'is-win');
      var panel = G.winPanel({
        kicker: 'Rätsel gelöst',
        title: 'Alles richtig!',
        stats: [G.count(m.entries.length, 'Wort', 'Wörter')].concat(revealed ? [G.count(revealed, 'Buchstabe', 'Buchstaben') + ' aufgedeckt'] : []),
        text: data.closing_text,
        replayLabel: 'Nochmal lösen',
        onReplay: function () {
          store.clear();
          st = X.state(m, null);
          done = false;
          engaged = false;
          build();
          G.scrollIntoView(root);
        }
      });
      G.clear(winBox);
      winBox.appendChild(panel);
      if (animate) {
        G.haptic(30);
        G.celebrate(boardEl, true);
        w.setTimeout(function () { G.scrollIntoView(panel); }, 700);
      }
    }

    /* ---------- echte Tastatur ohne Fokus im Feld ---------- */
    doc.addEventListener('keydown', function (e) {
      if (done || e.ctrlKey || e.metaKey || e.altKey || !engaged) return;
      var t = e.target;
      if (t === input || (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'BUTTON' || t.isContentEditable))) return;
      var letters = e.key && e.key.length <= 2 ? X.lettersOf(e.key) : [];
      if (letters.length === 1 || e.key === 'Backspace') {
        e.preventDefault();
        if (letters.length) X.input(m, st, letters[0]);
        else X.backspace(m, st);
        afterChange(letters.length > 0);
        return;
      }
      onKey(e);
    }, false);

    G.onWidthChange(root, sizeBoard, 120);

    done = !!(saved && saved.done) && X.isSolved(m, st);
    build();
  });
})();
