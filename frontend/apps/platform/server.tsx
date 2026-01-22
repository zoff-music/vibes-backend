import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { api } from '@vibez/api';
import { safeWrap } from '@vibez/shared';
import { renderToString } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';

const port = process.env.PORT || 3000;
const isDev = process.env.NODE_ENV !== 'production';

// Load manifest for hashed filenames (Synchronously at startup for absolute reliability)
let manifest: Record<string, string> = {};

console.log('[SSR] --- Production Startup (Final Refinement) ---');
console.log('[SSR] NODE_ENV:', process.env.NODE_ENV);
console.log('[SSR] isDev resolving to:', isDev);
console.log('[SSR] CWD:', process.cwd());
console.log('[SSR] import.meta.dir:', import.meta.dir);

if (!isDev || process.env.FORCE_MANIFEST === 'true') {
  try {
    const pathsToTry = [
      '/app/apps/platform/dist/manifest.json', // Exact confirmed path in container
      join(import.meta.dir, '../manifest.json'), // Relative to server.js in dist/server/
      join(process.cwd(), 'dist/manifest.json'), // CWD based
    ];

    for (const p of pathsToTry) {
      try {
        console.log(`[SSR] Checking manifest: ${p}`);
        if (existsSync(p)) {
          const content = readFileSync(p, 'utf8');
          manifest = JSON.parse(content);
          console.log(`[SSR] Manifest LOADED from ${p}:`, manifest);
          break;
        }
      } catch (e) {
        console.error(`[SSR] Error reading ${p}:`, e);
      }
    }
  } catch (err) {
    console.error('[SSR] CRITICAL Error during manifest loading:', err);
  }
}

function createHTMLShell(
  appHTML: string,
  initialData: any,
  mainJS: string,
  mainCSS: string,
) {
  const [err, dataScript] = safeWrap(() => JSON.stringify(initialData));
  const dataScriptContent = err ? '{}' : dataScript || '{}';

  return `<!DOCTYPE html>
<html lang="en" class="dark">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/favicon.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vibez - Shared Music Queue</title>
    <link rel="stylesheet" href="${mainCSS}" />
    <script id="ssr-data" type="application/json">${dataScriptContent}</script>
  </head>
  <body>
    <div id="root">${appHTML}</div>
    <script type="module" src="${mainJS}"></script>
  </body>
</html>`;
}

async function handleStaticFiles(path: string) {
  if (path.startsWith('/assets/platform/')) {
    const assetPath = path.replace('/assets/platform/', '');

    // Confirmed paths in container
    const paths = [
      join('/app/apps/platform/dist/assets/platform', assetPath),
      join(import.meta.dir, '../assets/platform', assetPath),
      join(process.cwd(), 'dist/assets/platform', assetPath),
    ];

    for (const p of paths) {
      if (existsSync(p)) {
        const file = Bun.file(p);
        const headers: Record<string, string> = {};
        if (assetPath.endsWith('.css')) headers['Content-Type'] = 'text/css';
        else if (assetPath.endsWith('.js'))
          headers['Content-Type'] = 'application/javascript';
        else if (assetPath.endsWith('.ico'))
          headers['Content-Type'] = 'image/x-icon';
        else if (assetPath.endsWith('.png'))
          headers['Content-Type'] = 'image/png';

        console.log(`[SSR] Serving: ${path} -> ${p}`);
        return new Response(file, { headers });
      }
    }
    console.warn(`[SSR] NOT FOUND: ${path} (Checked: ${paths})`);
    return new Response('Not Found', { status: 404 });
  }

  // Handle public files
  const publicPath = join(process.cwd(), 'public', path);
  if (existsSync(publicPath)) return new Response(Bun.file(publicPath));

  return null;
}

async function getInitialData(path: string, req: Request) {
  const roomMatch = path.match(/^\/room\/([^/]+)$/);
  if (!roomMatch || roomMatch[1] === 'create') {
    if (roomMatch && roomMatch[1] === 'create') {
      const url = new URL(req.url);
      const name = url.searchParams.get('name');
      if (name) return { data: { createRoomName: name }, redirect: null };
    }
    return { data: {}, redirect: null };
  }

  const roomId = roomMatch[1];
  const [err, room] = await api.get('/rooms/{id}', { id: roomId });

  if (err || !room) {
    const createUrl = new URL('/room/create', req.url);
    createUrl.searchParams.set('name', roomId);
    return { data: {}, redirect: Response.redirect(createUrl.toString(), 302) };
  }

  return { data: { room }, redirect: null };
}

Bun.serve({
  port,
  async fetch(req, server) {
    const url = new URL(req.url);
    const path = url.pathname;

    if (isDev && path === '/__hmr') {
      return server.upgrade(req)
        ? undefined
        : new Response('Upgrade failed', { status: 400 });
    }

    const staticResponse = await handleStaticFiles(path);
    if (staticResponse) return staticResponse;

    const { data: initialData, redirect } = await getInitialData(path, req);
    if (redirect) return redirect;

    // Get asset filenames from manifest
    const mainJS = manifest['main.js']
      ? `/assets/platform/${manifest['main.js']}`
      : '/assets/platform/client.js';
    const mainCSS = manifest['index.css']
      ? `/assets/platform/${manifest['index.css']}`
      : '/assets/platform/index.css';

    try {
      const appHTML = renderToString(
        <StaticRouter location={path}>
          <App initialData={initialData} />
        </StaticRouter>,
      );

      const fullHTML = createHTMLShell(appHTML, initialData, mainJS, mainCSS);
      return new Response(fullHTML, {
        headers: { 'Content-Type': 'text/html' },
      });
    } catch (error) {
      console.error('[SSR Error]', error);
      return new Response('Internal Server Error', { status: 500 });
    }
  },
  websocket: {
    message() {},
    open() {
      console.log('[HMR] Connected');
    },
  },
});

console.log(`[SSR] Running at http://localhost:${port} (Dev: ${isDev})`);
