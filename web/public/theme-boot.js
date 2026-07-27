// Applies the stored theme before first paint, so a light preference never
// flashes dark. It has to run without the bundle, so the key and the default are
// spelled out here as well as in src/lib/theme.ts; theme.test.ts asserts the two
// still agree. Loaded synchronously from <head>, and same-origin because the
// server's CSP blocks inline script.
(function () {
  var pref = 'dark';
  try {
    var raw = window.localStorage.getItem('reeve-theme');
    if (raw === 'light' || raw === 'dark') {
      pref = raw;
    }
  } catch {
    // Private mode or a blocked store: the default stands.
  }
  document.documentElement.setAttribute('data-theme', pref);
  var meta = document.querySelector('meta[name="theme-color"]');
  if (meta) {
    if (pref === 'light') {
      meta.setAttribute('content', '#ffffff');
    } else {
      meta.setAttribute('content', '#1f1e24');
    }
  }
})();
