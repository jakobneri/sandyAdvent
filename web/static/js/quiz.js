/* Quiz: eine Frage nach der anderen, sofortiges Feedback, Punktestand,
   Auswertung mit Abschlusstext. Optional Zitat-Karte („Wer hat's gesagt?“). */
(function () {
  'use strict';
  var G = window.SAGame;
  if (!G) return;
  var h = G.h;
  var PRAISE = ['Richtig!', 'Genau!', 'Volltreffer!', 'Stimmt!', 'Na klar!'];
  var LETTERS = 'ABCDEF';

  G.start('quiz', function (ctx) {
    var data = ctx.data, root = ctx.root, store = ctx.store;
    var Q = data.questions || [];
    var state;

    function fresh() {
      var order = G.range(Q.length);
      if (data.shuffle_questions) G.shuffle(order);
      var opt = [];
      for (var i = 0; i < Q.length; i++) {
        var o = G.range(Q[i].options.length);
        if (data.shuffle_options) G.shuffle(o);
        opt.push(o);
      }
      return { v: 1, order: order, opt: opt, i: 0, picks: [], started: false, done: false };
    }
    function valid(s) {
      if (!s || s.v !== 1 || !s.order || s.order.length !== Q.length || !s.opt || s.opt.length !== Q.length) return false;
      for (var i = 0; i < Q.length; i++) {
        if (!s.opt[i] || s.opt[i].length !== Q[i].options.length) return false;
      }
      return typeof s.i === 'number' && s.i >= 0 && s.i < Q.length && s.picks instanceof Array;
    }
    function save() { store.save(state); }
    function score() {
      var n = 0;
      for (var i = 0; i < Q.length; i++) {
        var q = Q[state.order[i]];
        if (state.picks[i] === q.answer) n++;
      }
      return n;
    }
    function answered() {
      var n = 0;
      for (var i = 0; i < Q.length; i++) if (typeof state.picks[i] === 'number') n++;
      return n;
    }

    state = store.load();
    if (!valid(state)) state = fresh();

    function header() {
      var bar = h('ol', { class: 'quiz__steps', 'aria-hidden': 'true' });
      for (var i = 0; i < Q.length; i++) {
        var q = Q[state.order[i]];
        var cls = 'quiz__step';
        if (typeof state.picks[i] === 'number') cls += state.picks[i] === q.answer ? ' is-right' : ' is-wrong';
        else if (i === state.i) cls += ' is-current';
        bar.appendChild(h('li', { class: cls }));
      }
      var s = score();
      return h('div', { class: 'quiz__head' }, [
        h('div', { class: 'quiz__meta' }, [
          h('p', { class: 'quiz__count', text: 'Frage ' + (state.i + 1) + ' von ' + Q.length }),
          h('p', { class: 'gchip gchip--gold', 'data-score': String(s), text: G.count(s, 'Punkt', 'Punkte') })
        ]),
        bar
      ]);
    }

    function renderStart() {
      G.clear(root);
      var card = h('div', { class: 'quiz-card quiz-card--start' }, [
        h('p', { class: 'quiz__big', text: String(Q.length) }),
        h('p', { class: 'quiz__lead', text: G.plural(Q.length, 'Frage', 'Fragen') + ' warten auf dich.' }),
        h('p', { class: 'quiz__hint', text: 'Tippe auf die Antwort, die du für richtig hältst. Du siehst sofort, ob es stimmt.' }),
        G.button('Los geht’s', {
          onclick: function () {
            state.started = true;
            save();
            renderQuestion(true);
          }
        })
      ]);
      root.appendChild(card);
    }

    function renderQuestion(focus) {
      G.clear(root);
      var qi = state.order[state.i];
      var q = Q[qi];
      var pick = state.picks[state.i];
      var isAnswered = typeof pick === 'number';
      var card = h('div', { class: 'quiz-card quiz-card--q', 'data-question': String(state.i) });
      card.appendChild(header());
      if (q.quote) {
        card.appendChild(h('figure', { class: 'quote' }, [
          h('span', { class: 'quote__mark', 'aria-hidden': 'true', text: '„' }),
          h('blockquote', { class: 'quote__text', text: q.quote })
        ]));
      }
      if (q.image) {
        card.appendChild(h('div', { class: 'polaroid quiz__img' }, h('img', { src: q.image, alt: '', decoding: 'async' })));
      }
      var title = h('h2', { class: 'quiz__q', id: 'quiz-q', tabindex: '-1', text: q.question });
      card.appendChild(title);
      var list = h('div', { class: 'quiz__opts', role: 'group', 'aria-labelledby': 'quiz-q' });
      var order = state.opt[qi];
      var btns = [];
      for (var k = 0; k < order.length; k++) {
        (function (k) {
          var oi = order[k];
          var b = h('button', {
            type: 'button', class: 'qopt', 'data-option': String(oi)
          }, [h('span', { class: 'qopt__key', 'aria-hidden': 'true', text: LETTERS.charAt(k) }), h('span', { class: 'qopt__text', text: q.options[oi] })]);
          b.addEventListener('click', function () { answer(oi); }, false);
          btns.push(b);
          list.appendChild(b);
        })(k);
      }
      card.appendChild(list);
      var fb = h('div', { class: 'quiz__feedback', 'aria-live': 'polite' });
      card.appendChild(fb);
      root.appendChild(card);

      function showResult(animate) {
        var right = pick === q.answer;
        for (var i = 0; i < btns.length; i++) {
          var oi = order[i];
          btns[i].disabled = true;
          if (oi === q.answer) G.addClass(btns[i], 'is-right');
          if (oi === pick && !right) G.addClass(btns[i], 'is-wrong');
          if (oi === pick) btns[i].setAttribute('aria-pressed', 'true');
          if (oi !== q.answer && oi !== pick) G.addClass(btns[i], 'is-dim');
        }
        G.clear(fb);
        G.addClass(fb, right ? 'is-right' : 'is-wrong');
        fb.appendChild(h('p', { class: 'quiz__verdict', text: right ? PRAISE[(state.i + Q.length) % PRAISE.length] : 'Leider daneben.' }));
        if (!right) fb.appendChild(h('p', { class: 'quiz__solution' }, ['Richtig ist: ', h('strong', { text: q.options[q.answer] })]));
        if (q.explain) fb.appendChild(h('p', { class: 'quiz__explain', text: q.explain }));
        var last = state.i === Q.length - 1;
        var next = G.button(last ? 'Zur Auswertung' : 'Weiter', {
          onclick: function () {
            if (last) {
              state.done = true;
              save();
              renderResult(true);
            } else {
              state.i++;
              save();
              renderQuestion(true);
            }
          }
        });
        next.setAttribute('data-next', '');
        fb.appendChild(h('div', { class: 'quiz__next' }, next));
        if (animate) {
          G.replayClass(card, right ? 'is-pop' : 'is-shake');
          G.haptic(right ? 12 : [20, 40, 20]);
          G.scrollIntoView(fb);
        }
      }

      function answer(oi) {
        if (typeof state.picks[state.i] === 'number') return;
        state.picks[state.i] = oi;
        pick = oi;
        save();
        var s = score();
        var chip = card.querySelector('[data-score]');
        if (chip) { chip.textContent = G.count(s, 'Punkt', 'Punkte'); chip.setAttribute('data-score', String(s)); }
        var steps = card.querySelectorAll('.quiz__step');
        if (steps[state.i]) steps[state.i].className = 'quiz__step ' + (oi === q.answer ? 'is-right' : 'is-wrong');
        showResult(true);
      }

      if (isAnswered) showResult(false);
      if (focus) { try { title.focus({ preventScroll: true }); } catch (e) {} G.scrollIntoView(card); }
    }

    function resultText(s) {
      var best = -1, text = '';
      var R = data.results || [];
      for (var i = 0; i < R.length; i++) {
        if (R[i].min_score <= s && R[i].min_score > best) { best = R[i].min_score; text = R[i].text; }
      }
      return text;
    }

    function renderResult(animate) {
      G.clear(root);
      var s = score();
      var perfect = s === Q.length;
      var recap = h('ol', { class: 'recap' });
      for (var i = 0; i < Q.length; i++) {
        var q = Q[state.order[i]];
        var ok = state.picks[i] === q.answer;
        recap.appendChild(h('li', { class: 'recap__item ' + (ok ? 'is-right' : 'is-wrong') }, [
          h('span', { class: 'recap__icon', 'aria-hidden': 'true', text: ok ? '✓' : '✗' }),
          h('span', { class: 'recap__body' }, [
            h('span', { class: 'recap__q', text: q.quote ? '„' + q.quote + '“' : q.question }),
            h('span', { class: 'recap__a', text: q.options[q.answer] })
          ]),
          h('span', { class: 'sr-only', text: ok ? ' – richtig' : ' – falsch' })
        ]));
      }
      var details = h('details', { class: 'recap__box' }, [h('summary', { text: 'Alle Antworten ansehen' }), recap]);
      var text = resultText(s);
      if (data.closing_text) text = text ? text + '\n\n' + data.closing_text : data.closing_text;
      var scoreEl = h('p', { class: 'win__score', 'data-final-score': String(s) }, [
        h('span', { class: 'win__score-n', text: String(s) }),
        h('span', { class: 'win__score-of', text: ' / ' + Q.length })
      ]);
      var panel = G.winPanel({
        kicker: 'Auswertung',
        title: perfect ? 'Volle Punktzahl!' : (s === 0 ? 'Huch!' : 'Geschafft!'),
        stats: [s + ' von ' + Q.length + ' richtig'],
        extra: scoreEl,
        text: text,
        onReplay: function () {
          store.clear();
          state = fresh();
          state.started = true;
          save();
          renderQuestion(true);
        }
      });
      panel.appendChild(details);
      root.appendChild(panel);
      if (animate) {
        G.scrollIntoView(panel);
        try { panel.focus({ preventScroll: true }); } catch (e) {}
        G.celebrate(panel, perfect);
      }
    }

    if (state.done && answered() === Q.length) renderResult(false);
    else if (state.started || answered() > 0) renderQuestion(false);
    else renderStart();
  });
})();
