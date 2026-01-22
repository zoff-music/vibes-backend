import { api } from '@vibez/api';
import { safeWrap } from '@vibez/shared';
import { renderToString } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';

const port = process.env.PORT || 3000;
const isDev = process.env.NODE_ENV !== 'production';

// Load manifest for hashed filenames (only in production)
let manifest: Record<string, string> = {};
if (!isDev) {
  try {
    const manifestFile = Bun.file('./dist/manifest.json');
    if (await manifestFile.exists()) {
      manifest = await manifestFile.json();
      console.log('[SSR] Loaded manifest:', manifest);
    } else {
      console.warn('[SSR] Manifest file not found at ./dist/manifest.json');
    }
  } catch (err) {
    console.warn('[SSR] Could not load manifest.json:', err);
  }
} else {
  console.log('[SSR] Development mode: skipping manifest loading');
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
<html lang="en">
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
  // Handle platform assets from /assets/platform/
  if (path.startsWith('/assets/platform/')) {
    const assetPath = path.replace('/assets/platform/', '');
    const distFile = Bun.file(`./dist/assets/platform/${assetPath}`);
    if (await distFile.exists()) {
      const headers: Record<string, string> = {};
      if (assetPath.endsWith('.css')) {
        headers['Content-Type'] = 'text/css';
      } else if (assetPath.endsWith('.js')) {
        headers['Content-Type'] = 'application/javascript';
      } else if (assetPath.endsWith('.ico')) {
        headers['Content-Type'] = 'image/x-icon';
      } else if (assetPath.endsWith('.png')) {
        headers['Content-Type'] = 'image/png';
      }
      return new Response(distFile, { headers });
    }
  }

  // Handle public files
  const publicFile = Bun.file(`./public${path}`);
  if (await publicFile.exists()) return new Response(publicFile);

  return null;
}

async function getInitialData(path: string, req: Request) {
  console.log(`[SSR] Processing path: ${path}`);
  console.log(`[SSR] Request URL: ${req.url}`);

  const roomMatch = path.match(/^\/room\/([^/]+)$/);
  console.log(`[SSR] Room match:`, roomMatch);

  if (!roomMatch || roomMatch[1] === 'create') {
    console.log(`[SSR] No room match or create page, returning empty data`);

    // If it's the create page, check for query parameters
    if (roomMatch && roomMatch[1] === 'create') {
      const url = new URL(req.url);
      const name = url.searchParams.get('name');
      console.log(`[SSR] Create page - URL: ${url.toString()}`);
      console.log(
        `[SSR] Create page - Query params:`,
        Object.fromEntries(url.searchParams.entries()),
      );
      console.log(`[SSR] Create page - Name parameter: ${name}`);
      if (name) {
        console.log(`[SSR] Create page with name parameter: ${name}`);
        const data = { createRoomName: name };
        console.log(`[SSR] Returning data:`, data);
        return { data, redirect: null };
      }
    }

    return { data: {}, redirect: null };
  }

  const roomId = roomMatch[1];
  console.log(`[SSR] Fetching room data for ${roomId}`);
  const [err, room] = await api.get('/rooms/{id}', { id: roomId });

  if (err || !room) {
    console.log(`[SSR] Room ${roomId} not found, redirecting to create...`);
    const createUrl = new URL('/room/create', req.url);
    createUrl.searchParams.set('name', roomId);
    console.log(`[SSR] Redirect URL: ${createUrl.toString()}`);
    console.log(`[SSR] Original request URL: ${req.url}`);
    return { data: {}, redirect: Response.redirect(createUrl.toString(), 302) };
  }

  console.log(`[SSR] Room ${roomId} found, returning room data`);
  return { data: { room }, redirect: null };
}

Bun.serve({
  port,
  async fetch(req, server) {
    const url = new URL(req.url);
    const path = url.pathname;

    console.log(`[Platform Server] Request: ${req.method} ${path}`);

    if (isDev && path === '/__hmr') {
      return server.upgrade(req)
        ? undefined
        : new Response('Upgrade failed', { status: 400 });
    }

    const staticResponse = await handleStaticFiles(path);
    if (staticResponse) {
      console.log(`[Platform Server] Serving static file: ${path}`);
      return staticResponse;
    }

    console.log(`[Platform Server] Processing route: ${path}`);
    const { data: initialData, redirect } = await getInitialData(path, req);
    if (redirect) return redirect;

    // Get the correct asset filenames from manifest (same format as cast app)
    const mainJS = manifest['main.js']
      ? `/assets/platform/${manifest['main.js']}`
      : '/assets/platform/client.js';
    const mainCSS = manifest['index.css']
      ? `/assets/platform/${manifest['index.css']}`
      : '/assets/platform/index.css';

    try {
      // Render the App component to string
      const appHTML = renderToString(
        <StaticRouter location={path}>
          <App initialData={initialData} />
        </StaticRouter>,
      );

      // Create the full HTML with the rendered app
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
      console.log('[HMR] Client connected');
    },
  },
});

console.log(`[SSR] Server running at http://localhost:${port} (Dev: ${isDev})`);
