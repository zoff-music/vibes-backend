package handler

// Branding follows zoff-music/vibes-frontend; the UI and assets remain backend-owned.
const documentationHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="dark light"><title>Zoff · API reference</title>
<link rel="icon" href="https://zoff.me/logo.png">
<link rel="stylesheet" href="/api/swagger/swagger-ui.css">
<link rel="stylesheet" href="/api/swagger/theme.css">
</head>
<body>
<a class="skip-link" href="#api-reference">Skip to API reference</a>
<header class="site-header shell">
<a href="/" class="brand" aria-label="Zoff home"><img src="https://zoff.me/logo.png" alt="" width="64" height="64"><span class="brand-wordmark"><strong lang="ja" aria-hidden="true">ゾフ</strong><small>Shared rooms</small></span></a>
<nav aria-label="Product navigation"><a href="/explore/rooms">Rooms</a><a href="/#explore-zoff">Explore</a><a href="/discovery/apps">Apps</a><button id="theme" type="button" aria-label="Switch color theme" title="Switch color theme"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8 2h8v2h3v3h3v10h-3v3h-3v2H8v-2H5v-3H2V7h3V4h3zm4 4H8v2H6v8h2v2h4z"/></svg></button></nav>
</header>
<main class="shell" id="api-reference">
<section class="intro" aria-labelledby="page-title">
<div><p class="eyebrow"><span></span> BUILT FOR SHARED LISTENING</p><h1 id="page-title">Good music.<br><span>Great integrations.</span></h1><p class="lede">Build your own way to listen together.<br>Explore rooms, queues, playback, and everything in between.</p></div>
<aside class="reference-card"><span class="card-label">DEVELOPER REFERENCE</span><div class="card-title">Vibes API <span>v1 + v2</span></div><p>REST endpoints &amp; live events</p><div class="card-footer"><span id="access" role="status">Loading reference…</span><a id="spec-link" href="/api/swagger/doc.json">JSON ↗</a></div></aside>
</section>
<div class="reference-heading"><div><span class="section-index">01 /</span><h2>API reference</h2></div><button id="refresh" type="button">Refresh access ↻</button></div>
<p id="docs-error" role="alert" hidden>Could not load the API reference. Please refresh to try again.</p>
<div id="swagger-ui"></div>
<noscript>This interactive reference requires JavaScript. <a href="/api/swagger/doc.json">Read the public JSON specification.</a></noscript>
<footer><a href="/">ゾフ / Zoff</a><span>Made for listening together.</span><a href="https://github.com/zoff-music/vibes-backend">View on GitHub ↗</a></footer>
</main>
<script src="/api/swagger/swagger-ui-bundle.js"></script>
<script src="/api/swagger/theme.js"></script>
</body></html>`

const documentationCSS = `
/* These content-hashed fonts are served by zoff-music/vibes-frontend at the same origin. */
@font-face {
  font-family:MSW98UI;
  font-style:normal;
  font-weight:400;
  font-display:swap;
  src:url("/assets/MSW98UI-Regular-q68EpDs9.woff2") format("woff2");
}
@font-face {
  font-family:MSW98UI;
  font-style:normal;
  font-weight:700;
  font-display:swap;
  src:url("/assets/MSW98UI-Bold-BY0kSkNY.woff2") format("woff2");
}
:root {
  --font-body:"MSW98UI","MS Sans Serif",Tahoma,"Zen Maru Gothic",system-ui,sans-serif;
  color-scheme:dark;
  --bg:#120b1e;
  --paper:#1a102a;
  --surface:#1e1330;
  --text:#e8dff5;
  --muted:#bfa8d6;
  --line:#493459;
  --pink:#ff74be;
  --cyan:#00d9ff;
  --glow:rgba(178,75,243,.10)
}
:root[data-theme="light"] {
  color-scheme:light;
  --bg:#f6ebff;
  --paper:#fdf7ff;
  --surface:#f2ddff;
  --text:#2a1840;
  --muted:#5b4276;
  --line:#d6bde8;
  --pink:#ab145c;
  --cyan:#006d82;
  --glow:rgba(255,63,164,.08)
}
* {
  box-sizing:border-box
}
body {
  margin:0;
  background:radial-gradient(ellipse at 75% 0,var(--glow),transparent 55%),var(--bg);
  color:var(--text);
  font-family:var(--font-body);
  font-size:14px
}
a {
  color:var(--cyan);
  text-decoration:none
}
button,a {
  touch-action:manipulation
}
button {
  font:inherit;
  cursor:pointer
}
a:focus-visible,button:focus-visible,input:focus-visible {
  outline:2px solid var(--cyan);
  outline-offset:5px
}
.shell {
  max-width:1152px;
  margin:auto;
  padding:0 24px;
}
.site-header {
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:16px;
  padding:28px 24px;
  line-height:24px;
  font-size:16px;
}
.brand {
  display:flex;
  flex-shrink:0;
  align-items:center;
  gap:12px;
  color:var(--text);
  border-radius:12px;
  transition:opacity .15s;
}
.brand img {
  border-radius:50%;
  width:64px;
  height:64px;
  transition:transform .5s;
}
.brand strong {
  display:block;
  font-family:Syncopate,sans-serif;
  font-size:48px;
  font-weight:400;
  line-height:48px;
  letter-spacing:-1.44px;
  color:var(--text);
  text-shadow:0 0 10px #ff50c859,0 0 24px #00d9ff40;
}
.brand small {
  display:block;
  margin-top:8px;
  font-size:12px;
  line-height:16px;
  letter-spacing:.24px;
  color:var(--muted);
}
nav {
  display:flex;
  align-items:center;
  gap:8px;
}
nav a {
  display:inline-flex;
  align-items:center;
  min-height:44px;
  padding:0 16px;
  border-radius:12px;
  color:var(--muted);
  font-size:14px;
  line-height:20px;
  letter-spacing:.14px;
  transition:color .15s,background .15s;
}
nav a[aria-current] {
  color:var(--cyan)
}
#theme {
  display:inline-flex;
  align-items:center;
  justify-content:center;
  flex-shrink:0;
  color:var(--text);
  background:var(--surface);
  border:1px solid rgba(112,72,140,.25);
  border-radius:12px;
  padding:0;
  width:44px;
  height:44px;
}
.intro {
  display:grid;
  grid-template-columns:1fr 330px;
  align-items:center;
  gap:55px;
  padding:62px 0 52px
}
.eyebrow {
  color:var(--cyan);
  font-size:10px;
  letter-spacing:2px;
  display:flex;
  align-items:center;
  gap:10px
}
.eyebrow span {
  height:6px;
  width:6px;
  border-radius:50%;
  background:var(--cyan);
  box-shadow:0 0 15px var(--cyan)
}
h1 {
  font-size:clamp(32px,4vw,49px);
  font-weight:400;
  line-height:1.17;
  letter-spacing:-1.8px;
  margin:22px 0 20px
}
h1 span {
  color:var(--pink)
}
.lede {
  color:var(--muted);
  font-size:14px;
  line-height:1.85;
  margin:0
}
.reference-card {
  background:linear-gradient(130deg,var(--surface),var(--paper));
  border:1px solid var(--line);
  border-radius:18px;
  padding:25px;
  box-shadow:0 16px 60px #00000010;
  position:relative;
  overflow:hidden
}
.reference-card:before {
  content:"";
  position:absolute;
  top:0;
  left:24px;
  right:24px;
  height:2px;
  background:linear-gradient(90deg,#ff2e97,#00d9ff)
}
.card-label {
  font-size:9px;
  letter-spacing:2px;
  color:var(--muted)
}
.card-title {
  display:flex;
  align-items:center;
  justify-content:space-between;
  font-size:24px;
  margin:24px 0 10px;
  gap:12px
}
.card-title span {
  font-size:10px;
  border:1px solid var(--line);
  border-radius:6px;
  padding:5px 7px;
  color:var(--cyan)
}
.reference-card p {
  font-size:12px;
  color:var(--muted)
}
.card-footer {
  display:flex;
  justify-content:space-between;
  gap:12px;
  border-top:1px solid var(--line);
  padding-top:18px;
  margin-top:26px;
  font-size:11px
}
#access {
  color:var(--muted)
}
.reference-heading,.reference-heading>div {
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:12px
}
.reference-heading {
  padding:22px 0;
  border-top:1px solid var(--line)
}
.section-index {
  color:var(--pink);
  font:12px monospace
}
h2 {
  font-size:17px;
  font-weight:500;
  margin:0
}
.reference-heading button {
  background:none;
  border:1px solid var(--line);
  border-radius:8px;
  color:var(--muted);
  padding:9px 12px;
  font-size:11px
}
.reference-heading button:hover {
  color:var(--cyan);
  border-color:var(--cyan)
}
footer {
  display:flex;
  justify-content:space-between;
  gap:15px;
  border-top:1px solid var(--line);
  padding:30px 0 40px;
  margin-top:40px;
  color:var(--muted);
  font-size:11px
}
footer a:first-child {
  color:var(--pink)
}
.skip-link {
  position:absolute;
  top:-100px;
  left:20px;
  padding:12px;
  background:var(--paper);
  z-index:10
}
.skip-link:focus {
  top:10px
}
#docs-error {
  padding:20px;
  border:1px solid var(--pink);
  border-radius:10px
}
/* Keep Swagger's interaction and method colors, while matching the surrounding product UI. */
.swagger-ui {
  color:var(--text);
  font-family:var(--font-body)
}
.swagger-ui .wrapper {
  padding:0;
  max-width:none
}
.swagger-ui .information-container,.swagger-ui .scheme-container,.swagger-ui .topbar {
  display:none
}
.swagger-ui .opblock-tag {
  color:var(--text);
  border-bottom:1px solid var(--line);
  padding:19px 5px;
  font-size:17px;
  font-weight:500
}
.swagger-ui .opblock-tag small {
  color:var(--muted)
}
.swagger-ui .opblock-tag:hover {
  background:var(--surface)
}
.swagger-ui .opblock-tag svg,.swagger-ui .expand-operation svg,.swagger-ui .opblock-control-arrow svg,.swagger-ui .opblock-summary svg,.swagger-ui .model-toggle:after {
  fill:var(--muted)
}
.swagger-ui .opblock {
  box-shadow:none;
  border-radius:9px;
  margin:0 0 12px;
  overflow:hidden
}
.swagger-ui .opblock .opblock-summary {
  padding:7px 9px
}
.swagger-ui .opblock .opblock-summary-path {
  font-size:13px;
  font-weight:500;
  color:var(--text)
}
.swagger-ui .opblock .opblock-summary-description {
  font-size:12px;
  color:var(--muted)
}
.swagger-ui .opblock .opblock-summary-method {
  font-size:11px;
  min-width:65px;
  padding:6px 0;
  text-shadow:none
}
.swagger-ui .opblock.opblock-get .opblock-summary-method {
  background:#2875ad
}
.swagger-ui .opblock.opblock-post .opblock-summary-method {
  background:#167852
}
.swagger-ui .opblock.opblock-put .opblock-summary-method {
  background:#985900
}
.swagger-ui .opblock.opblock-delete .opblock-summary-method {
  background:#b3334a
}
.swagger-ui .opblock.opblock-patch .opblock-summary-method {
  background:#6e50b0
}
.swagger-ui .opblock.opblock-options .opblock-summary-method {
  background:#66549d
}
.swagger-ui .opblock.opblock-head .opblock-summary-method {
  background:#765098
}
.swagger-ui .opblock .opblock-section-header {
  background:var(--surface);
  box-shadow:none
}
.swagger-ui .opblock .opblock-section-header h4,.swagger-ui .opblock .opblock-section-header label,.swagger-ui .parameter__name,.swagger-ui .parameter__type,.swagger-ui .parameter__in,.swagger-ui .response-col_description,.swagger-ui .response-col_status,.swagger-ui .response-col_links,.swagger-ui .tab li,.swagger-ui table thead tr th,.swagger-ui .opblock-description-wrapper p,.swagger-ui .opblock-external-docs-wrapper p,.swagger-ui .opblock-title_normal p,.swagger-ui .renderedMarkdown p,.swagger-ui .renderedMarkdown li,.swagger-ui .model,.swagger-ui .model-title,.swagger-ui section.models h4,.swagger-ui section.models h4 span,.swagger-ui .prop-format,.swagger-ui label {
  color:var(--text)
}
.swagger-ui .btn {
  color:var(--text);
  border-color:var(--muted);
  box-shadow:none
}
.swagger-ui .btn.execute {
  background:#286ba5;
  border-color:#286ba5;
  color:white
}
.swagger-ui select,.swagger-ui input[type=text],.swagger-ui input[type=password],.swagger-ui input[type=search],.swagger-ui input[type=email],.swagger-ui input[type=file],.swagger-ui textarea {
  background:var(--paper);
  color:var(--text);
  border-color:var(--line)
}
.swagger-ui input::placeholder,.swagger-ui textarea::placeholder {
  color:var(--muted)
}
.swagger-ui section.models {
  border-color:var(--line);
  border-radius:10px
}
.swagger-ui section.models .model-container {
  background:var(--surface)
}
.swagger-ui section.models h4 {
  border-color:var(--line)
}
.swagger-ui .model-box {
  background:var(--surface)
}
.swagger-ui .model-toggle:after {
  filter:invert(.6)
}
.swagger-ui .prop-type {
  color:var(--cyan)
}
.swagger-ui .markdown code,.swagger-ui .renderedMarkdown code {
  color:var(--pink);
  background:var(--surface)
}
.swagger-ui .dialog-ux .modal-ux {
  background:var(--paper);
  border-color:var(--line)
}
.swagger-ui .dialog-ux .modal-ux-header h3,.swagger-ui .dialog-ux .modal-ux-content h4,.swagger-ui .dialog-ux .modal-ux-content p {
  color:var(--text)
}
.swagger-ui .download-contents {
  color:white
}
.swagger-ui .loading-container .loading:after {
  color:var(--muted)
}
.swagger-ui .filter-container {
  padding:0 0 15px
}
.swagger-ui .filter .operation-filter-input {
  width:100%;
  width:100%;
  border:1px solid var(--line);
  border-radius:9px;
  padding:12px 14px;
  background:var(--paper);
  color:var(--text);
  font:13px var(--font-body)
}
@media(max-width:720px) {
  .swagger-ui .opblock .opblock-section-header {
    flex-wrap:wrap;
    gap:12px;
  }
  .swagger-ui .opblock .opblock-section-header > label {
    flex-wrap:wrap;
    gap:8px;
    min-width:0;
  }
  .shell {
    padding-left:18px;
    padding-right:18px
  }
  .intro {
    grid-template-columns:1fr;
    gap:28px;
    padding:32px 0
  }
  .reference-card {
    padding:22px
  }
  .card-title {
    margin-top:18px
  }
  .card-footer {
    margin-top:20px
  }
  .swagger-ui .opblock .opblock-summary {
    flex-wrap:wrap;
    gap:4px
  }
  .swagger-ui .opblock .opblock-summary-path {
    max-width:calc(100% - 105px);
    font-size:11px;
    overflow-wrap:anywhere
  }
  .swagger-ui .opblock .opblock-summary-description {
    flex-basis:100%;
    padding:4px
  }
  .swagger-ui .opblock-tag {
    font-size:15px
  }
  .swagger-ui .opblock-tag small {
    font-size:11px
  }
  footer {
    flex-wrap:wrap
  }
  footer span {
    display:none
  }
}
.brand:hover {
  opacity:.8;
}
nav a:hover {
  background:var(--surface);
  color:var(--text);
}
#theme:hover {
  border-color:rgba(112,72,140,.4);
}
#theme svg {
  width:20px;
  height:20px;
}
#swagger-ui .swagger-ui, #swagger-ui .swagger-ui * {
  font-family:var(--font-body);
}
@media(prefers-reduced-motion:no-preference) {
  .brand:hover img {
    transform:rotate(12deg);
  }
}
@media(max-width:639px) {
  .site-header {
    padding:20px;
    gap:12px;
  }
  .brand img {
    width:48px;
    height:48px;
  }
  .brand-wordmark {
    display:none;
  }
  nav {
    gap:4px;
  }
  nav a {
    padding:0 8px;
  }
}
`

const documentationJS = `
(() => {
  const root = document.documentElement;
  root.dataset.theme = window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
  document.getElementById('theme').addEventListener('click', () => {
    root.dataset.theme = root.dataset.theme === 'dark' ? 'light' : 'dark';
  });
  let ui;
  let adminVisible = false;
  let loading = false;
  const access = document.getElementById('access');
  const specLink = document.getElementById('spec-link');
  async function refresh() {
    if (loading) return;
    loading = true;
    document.getElementById('docs-error').hidden = true;
    try {
      const adminResponse = await fetch('/api/swagger/admin.json', { credentials: 'same-origin', cache: 'no-store' });
      const isAdmin = adminResponse.ok;
      if (!isAdmin && adminVisible) {
        document.getElementById('swagger-ui').replaceChildren();
        ui = null;
        adminVisible = false;
        access.textContent = 'Public reference';
        specLink.href = '/api/swagger/doc.json';
      }
      const response = isAdmin ? adminResponse : await fetch('/api/swagger/doc.json', { credentials: 'same-origin', cache: 'no-store' });
      if (!response.ok) throw new Error('Unable to load documentation');
      const spec = await response.json();
      if (!ui || isAdmin !== adminVisible) {
        ui = SwaggerUIBundle({
          spec, dom_id: '#swagger-ui', deepLinking: true, docExpansion: 'list',
          filter: true, tagsSorter: 'alpha', operationsSorter: 'alpha', displayRequestDuration: true, defaultModelsExpandDepth: 1,
          validatorUrl: null, presets: [SwaggerUIBundle.presets.apis], layout: 'BaseLayout',
          requestInterceptor: (request) => { request.credentials = 'same-origin'; return request; },
          responseInterceptor: (response) => {
            if (/\/api\/v1\/admin\/sessions(?:\?|$)/.test(response.url)) window.setTimeout(refresh, 0);
            return response;
          }
        });
      }
      adminVisible = isAdmin;
      access.textContent = isAdmin ? 'Admin reference · authenticated' : 'Public reference';
      specLink.href = isAdmin ? '/api/swagger/admin.json' : '/api/swagger/doc.json';
    } catch (error) {
      // A failed session check must never leave an admin specification on screen.
      document.getElementById('swagger-ui').replaceChildren();
      ui = null;
      adminVisible = false;
      access.textContent = 'Reference unavailable';
      specLink.href = '/api/swagger/doc.json';
      document.getElementById('docs-error').hidden = false;
    } finally {
      loading = false;
    }
  }
  document.getElementById('refresh').addEventListener('click', refresh);
  window.addEventListener('focus', refresh);
  window.addEventListener('pageshow', (event) => { if (event.persisted) refresh(); });
  window.setInterval(() => { if (!document.hidden) refresh(); }, 60000);
  refresh();
})();
`
