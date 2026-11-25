/* Läuft synchron im <head>: setzt .js und entscheidet vor dem ersten Paint,
   ob die Öffnungsanimation gespielt wird (nur beim ersten Besuch eines Türchens). */
(function () {
  var d = document.documentElement;
  d.className = d.className.replace(/\bno-js\b/, 'js');
  if (d.getAttribute('data-intro') !== '1') return;
  var day = d.getAttribute('data-day');
  var force = /[?&]intro=1\b/.test(window.location.search);
  var reduce = false;
  try { reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches; } catch (e) {}
  var seen = false;
  try { seen = window.localStorage.getItem('sa-door-' + day) === '1'; } catch (e) {}
  if (force || (!seen && !reduce)) d.className += ' intro';
})();
