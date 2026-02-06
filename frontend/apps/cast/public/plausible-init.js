// Bootstrap `window.plausible` without inline scripts (CSP-friendly).
// This lets app code call plausible() before the analytics script has loaded.
window.plausible =
  window.plausible ||
  ((...args) => {
    window.plausible.q = window.plausible.q || [];
    window.plausible.q.push(args);
  });

