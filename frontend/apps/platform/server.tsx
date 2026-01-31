import { existsSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { api, createApiClient } from '@vibez/api';
import {
  applyConsoleLogGuard,
  isTruthyFlag,
  type PlaybackState,
  safeWrap,
  safeWrapAsync,
} from '@vibez/shared';
import { renderToString } from 'react-dom/server';
import {
  StaticRouterProvider,
  createStaticHandler,
  createStaticRouter,
} from 'react-router';
import type { SSRInitialData } from './src/App';
import { createServerRoutes } from './src/routes.server';

const port = process.env.PORT || 3000;
const isDev = process.env.NODE_ENV !== 'production';
const debugEnabled = isTruthyFlag(process.env.VITE_DEBUG ?? process.env.DEBUG);
applyConsoleLogGuard(debugEnabled);

// Load manifest for hashed filenames (Synchronously at startup for absolute reliability)
let manifest: Record<string, unknown> = {};

console.log('[SSR] --- Production Startup (Final Refinement) ---');
console.log('[SSR] NODE_ENV:', process.env.NODE_ENV);
console.log('[SSR] isDev resolving to:', isDev);
console.log('[SSR] CWD:', process.cwd());
console.log('[SSR] import.meta.dir:', import.meta.dir);

if (!isDev || process.env.FORCE_MANIFEST === 'true') {
  const [criticalErr] = safeWrap(() => {
    const pathsToTry = [
      '/app/apps/platform/dist/assets/platform/manifest.json', // Vite manifest path in container
      '/app/apps/platform/dist/manifest.json', // Legacy Bun manifest path in container
      join(import.meta.dir, '../assets/platform/manifest.json'), // Relative to server.js in dist/server/
      join(import.meta.dir, '../manifest.json'), // Legacy Bun manifest relative to server.js
      join(process.cwd(), 'dist/assets/platform/manifest.json'), // CWD based Vite manifest
      join(process.cwd(), 'dist/manifest.json'), // Legacy Bun manifest
    ];

    for (const p of pathsToTry) {
      const [err] = safeWrap(() => {
        console.log(`[SSR] Checking manifest: ${p}`);
        if (existsSync(p)) {
          const content = readFileSync(p, 'utf8');
          manifest = JSON.parse(content) as Record<string, unknown>;
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
  initialData: SSRInitialData,
  mainJS: string,
  cssFiles: string[],
  themeClass: string = '',
) {
  const [err, dataScript] = safeWrap(() => JSON.stringify(initialData));
  const dataScriptContent = err ? '{}' : dataScript || '{}';

  return `<!DOCTYPE html>
<html lang="en" class="${themeClass}">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/favicon-96x96.png" sizes="96x96" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <link rel="shortcut icon" href="/favicon.ico" />
    <link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
    <link rel="manifest" href="/site.webmanifest" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>ゾフ - Shared Music Queue</title>
    ${cssFiles
      .map((css) => `<link rel="stylesheet" href="${css}" />`)
      .join('\n    ')}
    <script id="ssr-data" type="application/json">${dataScriptContent}</script>
  </head>
  <body>
    <div id="root">${appHTML}</div>
    <script type="module" src="${mainJS}"></script>
  </body>
</html>`;
}

function resolveAssetsFromManifest() {
  const toAssetUrl = (file: string | undefined) => {
    if (!file) return '';
    return file.startsWith('/') ? file : `/assets/platform/${file}`;
  };

  if (manifest['main.js'] && typeof manifest['main.js'] === 'string') {
    const mainJS = toAssetUrl(manifest['main.js'] as string);
    const cssEntry = manifest['index.css'];
    const cssFiles =
      typeof cssEntry === 'string' ? [toAssetUrl(cssEntry)] : [];

    return {
      mainJS,
      cssFiles,
    };
  }

  const manifestEntries = Object.values(manifest).filter(
    (entry) => entry && typeof entry === 'object',
  ) as Array<{
    file?: string;
    css?: string[];
    isEntry?: boolean;
  }>;

  const entryByKey = Object.keys(manifest).find((key) =>
    key.endsWith('client.tsx'),
  );
  const entry =
    (entryByKey ? (manifest[entryByKey] as any) : null) ||
    manifestEntries.find((item) => item.isEntry);

  const mainJS = entry?.file ? toAssetUrl(entry.file) : '';
  const cssFiles = Array.isArray(entry?.css)
    ? entry.css.map((css) => toAssetUrl(css))
    : [];

  return {
    mainJS,
    cssFiles,
  };
}

async function handleStaticFiles(path: string) {
  // Never handle root or empty paths as static files
  if (path === '/' || path === '') return null;

  // Handle platform assets from /assets/platform/
  if (path.startsWith('/assets/platform/')) {
    // Strip query parameters for file lookup
    const assetPath = path.replace('/assets/platform/', '').split('?')[0];

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

async function getInitialData(
  path: string,
  req: Request,
): Promise<{ data: SSRInitialData; redirect: Response | null }> {
  if (path === '/admin' || path === '/admin/') {
    const cookieHeader = req.headers.get('cookie') ?? req.headers.get('Cookie');
    console.log('[SSR Admin] Handling admin SSR path:', path);
    const [roomsErr, rooms] = await api.get('/admin/rooms', null, {
      headers: { Cookie: cookieHeader },
    });
    if (roomsErr) {
      console.log('[SSR Admin] /admin/rooms failed:', roomsErr.message);
    } else {
      console.log('[SSR Admin] /admin/rooms success:', rooms?.length ?? 0);
    }

    return {
      data: {
        adminRooms: roomsErr ? [] : rooms || [],
        adminAuthorized: !roomsErr,
      },
      redirect: null,
    };
  }

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
  const cookieHeader = req.headers.get('cookie') ?? req.headers.get('Cookie');
  console.log(`[SSR] Room: ${roomId}, Cookie present: ${!!cookieHeader}`);
  if (cookieHeader) {
    console.log(`[SSR] Forwarding Cookie: ${cookieHeader}`);
  }

  const authenticatedApi = cookieHeader
    ? createApiClient({ Cookie: cookieHeader })
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
      playback: (playback || undefined) as PlaybackState | undefined,
      theme: getThemeFromCookies(cookieHeader) as any, // Pass theme to client
    },
    redirect: null,
  };
}

function getThemeFromCookies(
  cookieHeader: string | null,
): 'light' | 'dark' | 'auto' {
  console.log('COOKIEHEADER', cookieHeader);
  if (!cookieHeader) {
    return 'auto';
  }

  try {
    const cookies = cookieHeader
      .split(';')
      .reduce<Record<string, string>>((acc, cookie) => {
        const trimmed = cookie.trim();
        if (!trimmed) return acc;
        const separatorIndex = trimmed.indexOf('=');
        if (separatorIndex === -1) return acc;
        const name = trimmed.slice(0, separatorIndex);
        const value = trimmed.slice(separatorIndex + 1);
        if (!name || !value) return acc;
        acc[name] = decodeURIComponent(value);
        return acc;
      }, {});

    let preferencesEncoded = cookies.preferences;
    if (!preferencesEncoded) {
      return 'auto';
    }

    // Strip optional quotes around cookie values
    if (
      preferencesEncoded.startsWith('"') &&
      preferencesEncoded.endsWith('"')
    ) {
      preferencesEncoded = preferencesEncoded.slice(1, -1);
    }

    // Decode base64 JSON - try both standard and URL-safe base64
    const normalized = preferencesEncoded.replace(/-/g, '+').replace(/_/g, '/');
    const preferencesJson = Buffer.from(normalized, 'base64').toString('utf-8');
    const preferences = JSON.parse(preferencesJson);

    // Only return 'dark' if explicitly set to dark, etc.
    if (preferences.theme === 'dark') return 'dark';
    if (preferences.theme === 'auto') return 'auto';
    if (preferences.theme === 'light') return 'light';

    return 'auto';
  } catch (error) {
    console.error('[SSR] Error parsing theme preferences:', error);
    return 'auto';
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
    const cookieHeader = req.headers.get('cookie') ?? req.headers.get('Cookie');
    const themeId = getThemeFromCookies(cookieHeader);

    // Map theme ID to HTML class
    const themeClassMap = {
      dark: 'dark',
      light: 'theme-light',
      auto: '',
    };
    const themeClass = themeClassMap[themeId] ?? '';

    // Add theme to initialData for all routes
    initialData.theme = themeId;

    // Get asset filenames from manifest
    const { mainJS, cssFiles } = resolveAssetsFromManifest();
    const resolvedMainJS = mainJS || '/assets/platform/client.js';
    const resolvedCSSFiles =
      cssFiles.length > 0 ? cssFiles : ['/assets/platform/index.css'];

    const routes = createServerRoutes(initialData);
    const { query, dataRoutes } = createStaticHandler(routes);
    const [queryErr, context] = await safeWrapAsync(query(req));
    if (queryErr || !context) {
      console.error('[SSR] Static handler error:', queryErr);
      return new Response('Internal Server Error', { status: 500 });
    }
    if (context instanceof Response) {
      return context;
    }

    const router = createStaticRouter(dataRoutes, context);
    const [error, appHTML] = safeWrap(() =>
      renderToString(
        <StaticRouterProvider router={router} context={context} />,
      ),
    );

    if (error || !appHTML) {
      console.error('[SSR Error]', error);
      return new Response('Internal Server Error', { status: 500 });
    }

    const fullHTML = createHTMLShell(
      appHTML,
      initialData,
      resolvedMainJS,
      resolvedCSSFiles,
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
