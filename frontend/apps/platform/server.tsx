import { existsSync, readFileSync, statSync } from 'node:fs';
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
  const [criticalErr] = safeWrap(() => {
    const pathsToTry = [
      '/app/apps/platform/dist/manifest.json', // Exact confirmed path in container
      join(import.meta.dir, '../manifest.json'), // Relative to server.js in dist/server/
      join(process.cwd(), 'dist/manifest.json'), // CWD based
    ];

    for (const p of pathsToTry) {
      const [err] = safeWrap(() => {
        console.log(`[SSR] Checking manifest: ${p}`);
        if (existsSync(p)) {
          const content = readFileSync(p, 'utf8');
          manifest = JSON.parse(content);
          console.log(`[SSR] Manifest LOADED from ${p}:`, manifest);
          return true;
        }
        return false;
      });

      if (err) {
        console.error(`[SSR] Error reading ${p}:`, err);
      } else if (manifest && Object.keys(manifest).length > 0) {
        // If we successfully loaded a manifest, we can stop
        break;
      }
    }
  });

  if (criticalErr) {
    console.error('[SSR] CRITICAL Error during manifest loading:', criticalErr);
  }
}

function createHTMLShell(
  appHTML: string,
  initialData: any,
  mainJS: string,
  mainCSS: string,
  themeClass: string = '',
) {
  const [err, dataScript] = safeWrap(() => JSON.stringify(initialData));
  const dataScriptContent = err ? '{}' : dataScript || '{}';

  return `<!DOCTYPE html>
<html lang="en" class="${themeClass}">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/favicon.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Nori - Shared Music Queue</title>
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
  // Never handle root or empty paths as static files
  if (path === '/' || path === '') return null;

  // Handle platform assets from /assets/platform/
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
        const [err, stats] = safeWrap(() => statSync(p));
        if (!err && stats?.isFile()) {
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
    }

    // Don't log 404 for obvious non-assets or favicon
    if (path.endsWith('.js') || path.endsWith('.css')) {
      console.warn(`[SSR] Asset NOT FOUND: ${path}`);
    }
    return null;
  }

  // Handle public files if they exist and are actual files
  const publicPath = join(process.cwd(), 'public', path);
  const [err, stats] = safeWrap(() => statSync(publicPath));
  if (!err && stats?.isFile()) {
    return new Response(Bun.file(publicPath));
  }

  return null;
}

async function getInitialData(path: string, req: Request) {
  const roomMatch = path.match(/^\/rooms\/([^/]+)$/);
  if (!roomMatch || roomMatch[1] === 'create') {
    if (roomMatch && roomMatch[1] === 'create') {
      const url = new URL(req.url);
      const name = url.searchParams.get('name');
      if (name) return { data: { createRoomName: name }, redirect: null };
    }
    return { data: {}, redirect: null };
  }

  const roomId = roomMatch[1];
  const cookieHeader = req.headers.get('Cookie');
  console.log(`[SSR] Room: ${roomId}, Cookie present: ${!!cookieHeader}`);
  if (cookieHeader) {
    console.log(`[SSR] Forwarding Cookie: ${cookieHeader}`);
  }

  const authenticatedApi = cookieHeader
    ? (api as any).withHeaders({ Cookie: cookieHeader })
    : api;

  // Fetch all necessary data for room view in parallel
  const [roomRes, songsRes, playbackRes] = await Promise.all([
    authenticatedApi.get('/rooms/{id}', { id: roomId }),
    authenticatedApi.get('/rooms/{id}/songs', { id: roomId }),
    authenticatedApi.get('/rooms/{id}/states', { id: roomId }),
  ]);

  const [roomErr, room] = roomRes;
  const [_songsErr, songs] = songsRes;
  const [_playbackErr, playback] = playbackRes;

  if (roomErr || !room) {
    const createUrl = new URL('/rooms/create', req.url);
    createUrl.searchParams.set('name', roomId);
    return { data: {}, redirect: Response.redirect(createUrl.toString(), 302) };
  }

  return {
    data: {
      room,
      songs: songs || [],
      playback: playback || null,
      theme: getThemeFromCookies(cookieHeader), // Pass theme to client
    },
    redirect: null,
  };
}

function getThemeFromCookies(cookieHeader: string | null): string {
  if (!cookieHeader) {
    return ''; // Default to light mode (no class)
  }

  try {
    // Parse cookies
    const cookies = cookieHeader.split(';').reduce(
      (acc, cookie) => {
        const [name, value] = cookie.trim().split('=');
        if (name && value) {
          acc[name] = decodeURIComponent(value);
        }
        return acc;
      },
      {} as Record<string, string>,
    );

    const preferencesEncoded = cookies.preferences;
    if (!preferencesEncoded) {
      return ''; // Default to light mode (no class)
    }

    // Decode base64 JSON - use Buffer for Node.js compatibility
    const preferencesJson = Buffer.from(preferencesEncoded, 'base64').toString(
      'utf-8',
    );
    const preferences = JSON.parse(preferencesJson);

    // Only return 'dark' if explicitly set to dark, otherwise default to light
    if (preferences.theme === 'dark') {
      return 'dark';
    }

    return ''; // Default to light mode (no class)
  } catch (error) {
    console.log(
      '[SSR] Error parsing theme preferences, defaulting to light:',
      error,
    );
    return ''; // Default to light mode (no class)
  }
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

    // Try static files first (skipping root)
    const staticResponse = await handleStaticFiles(path);
    if (staticResponse) return staticResponse;

    // If it's a known non-route that isn't a static file, return 404 early
    if (
      path === '/favicon.ico' ||
      (path.includes('.') && !path.startsWith('/rooms/'))
    ) {
      return new Response('Not Found', { status: 404 });
    }

    const { data: initialData, redirect } = await getInitialData(path, req);
    if (redirect) return redirect;

    // Get theme from cookies for SSR
    const cookieHeader = req.headers.get('Cookie');
    const themeClass = getThemeFromCookies(cookieHeader);

    // Add theme to initialData for all routes
    initialData.theme = themeClass === 'dark' ? 'dark' : 'light';

    // Get asset filenames from manifest
    const mainJS = manifest['main.js']
      ? `/assets/platform/${manifest['main.js']}`
      : '/assets/platform/client.js';
    const mainCSS = manifest['index.css']
      ? `/assets/platform/${manifest['index.css']}`
      : '/assets/platform/index.css';

    const [error, appHTML] = safeWrap(() =>
      renderToString(
        <StaticRouter location={path}>
          <App initialData={initialData} />
        </StaticRouter>,
      ),
    );

    if (error || !appHTML) {
      console.error('[SSR Error]', error);
      return new Response('Internal Server Error', { status: 500 });
    }

    const fullHTML = createHTMLShell(
      appHTML,
      initialData,
      mainJS,
      mainCSS,
      themeClass,
    );
    return new Response(fullHTML, {
      headers: { 'Content-Type': 'text/html' },
    });
  },
  websocket: {
    message() {},
    open() {
      console.log('[HMR] Connected');
    },
  },
});

console.log(`[SSR] Running at http://localhost:${port} (Dev: ${isDev})`);
