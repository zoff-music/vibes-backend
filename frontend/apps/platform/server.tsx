import { join } from 'node:path';
import { api } from '@vibez/api';
import { safeWrap } from '@vibez/shared';
import { renderToString } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';
import { readdirSync, existsSync } from 'node:fs';

const port = process.env.PORT || 3000;
const isDev = process.env.NODE_ENV !== 'production';

// Load manifest for hashed filenames (only in production)
let manifest: Record<string, string> = {};
if (!isDev) {
  console.log('[SSR] --- Production Startup Diagnostics ---');
  console.log('[SSR] CWD:', process.cwd());
  console.log('[SSR] import.meta.dir:', import.meta.dir);

  try {
    // List files recursively for debugging (non-blocking enough for startup)
    const listFiles = (dir: string, depth = 0) => {
      if (depth > 3 || !existsSync(dir)) return;
      try {
        const files = readdirSync(dir, { withFileTypes: true });
        for (const file of files) {
          const fullPath = join(dir, file.name);
          console.log(`[SSR] [DIR-SCN] ${'  '.repeat(depth)}${file.isDirectory() ? '[D]' : '[F]'} ${file.name} (${fullPath})`);
          if (file.isDirectory() && !file.name.includes('node_modules')) {
            listFiles(fullPath, depth + 1);
          }
        }
      } catch (e) { }
    };

    console.log('[SSR] Scanning current directory structure...');
    listFiles(process.cwd());

    const findManifest = async () => {
      const pathsToTry = [
        join(import.meta.dir, '../manifest.json'), // Platform production: dist/manifest.json
        join(process.cwd(), 'dist/manifest.json'),
        join(import.meta.dir, 'manifest.json'),
        '/app/apps/platform/dist/manifest.json',
      ];

      for (const p of pathsToTry) {
        console.log('[SSR] Checking manifest at:', p);
        const file = Bun.file(p);
        if (await file.exists()) {
          const content = await file.json();
          console.log('[SSR] Manifest SUCCESS at:', p);
          console.log('[SSR] Manifest content:', content);
          return content;
        }
      }
      return null;
    };

    const loadedManifest = await findManifest();
    if (loadedManifest) {
      manifest = loadedManifest;
    } else {
      console.error('[SSR] CRITICAL: Manifest not found anywhere!');
    }
  } catch (err) {
    console.error('[SSR] Error during startup diagnostics/manifest loading:', err);
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

    // Check locations relative to the server entry point (dist/server/server.js)
    // Assets are in dist/assets/platform/
    const p1 = join(import.meta.dir, '../assets/platform', assetPath);
    const p2 = join(process.cwd(), 'dist/assets/platform', assetPath);

    for (const p of [p1, p2]) {
      const file = Bun.file(p);
      if (await file.exists()) {
        const headers: Record<string, string> = {};
        if (assetPath.endsWith('.css')) headers['Content-Type'] = 'text/css';
        else if (assetPath.endsWith('.js')) headers['Content-Type'] = 'application/javascript';
        else if (assetPath.endsWith('.ico')) headers['Content-Type'] = 'image/x-icon';
        else if (assetPath.endsWith('.png')) headers['Content-Type'] = 'image/png';
        return new Response(file, { headers });
      }
    }
    console.warn(`[SSR] Asset 404: ${path} (Checked ${p1}, ${p2})`);
    return new Response('Not Found', { status: 404 });
  }

  const publicFile = Bun.file(join(process.cwd(), 'public', path));
  if (await publicFile.exists()) return new Response(publicFile);

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
      return server.upgrade(req) ? undefined : new Response('Upgrade failed', { status: 400 });
    }

    const staticResponse = await handleStaticFiles(path);
    if (staticResponse) return staticResponse;

    const { data: initialData, redirect } = await getInitialData(path, req);
    if (redirect) return redirect;

    // Get asset filenames from manifest
    const mainJS = manifest['main.js'] ? `/assets/platform/${manifest['main.js']}` : '/assets/platform/client.js';
    const mainCSS = manifest['index.css'] ? `/assets/platform/${manifest['index.css']}` : '/assets/platform/index.css';

    try {
      const appHTML = renderToString(
        <StaticRouter location={path}>
          <App initialData={initialData} />
        </StaticRouter>,
      );

      const fullHTML = createHTMLShell(appHTML, initialData, mainJS, mainCSS);
      return new Response(fullHTML, { headers: { 'Content-Type': 'text/html' } });
    } catch (error) {
      console.error('[SSR Error]', error);
      return new Response('Internal Server Error', { status: 500 });
    }
  },
  websocket: {
    message() { },
    open() { console.log('[HMR] Client connected'); },
  },
});

console.log(`[SSR] Running at http://localhost:${port} (Dev: ${isDev})`);
