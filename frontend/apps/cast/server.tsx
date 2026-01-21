import { safeWrapAsync } from '@vibez/shared';
import { renderToReadableStream } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';

const port = process.env.PORT || 3001;
const isDev = process.env.NODE_ENV !== 'production';

// Load build manifest for hashed filenames
async function loadManifest() {
  try {
    const manifestFile = Bun.file('./dist/manifest.json');
    if (await manifestFile.exists()) {
      const content = await manifestFile.text();
      return JSON.parse(content);
    }
  } catch (err) {
    console.warn('[SSR] Could not load manifest.json:', err);
  }
  return {};
}

let manifest = await loadManifest();

async function handleStaticFiles(path: string) {
  console.log(`[Static] Handling request for: ${path}`);
  console.log(`[Static] Current working directory: ${process.cwd()}`);
  
  // Handle cast assets from /assets/cast/
  if (path.startsWith('/assets/cast/')) {
    const assetPath = path.replace('/assets/cast/', '');
    const distFile = Bun.file(`./dist/assets/cast/${assetPath}`);
    const fullPath = `./dist/assets/cast/${assetPath}`;
    const absolutePath = `${process.cwd()}/dist/assets/cast/${assetPath}`;
    console.log(`[Static] Looking for cast asset: ${fullPath}`);
    console.log(`[Static] Absolute path: ${absolutePath}`);
    
    const exists = await distFile.exists();
    console.log(`[Static] File exists: ${exists}`);
    
    if (exists) {
      console.log(`[Static] Found cast asset: ${assetPath}`);
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
      console.log(`[Static] Serving with headers:`, headers);
      return new Response(distFile, { headers });
    } else {
      console.log(`[Static] Cast asset not found: ${assetPath}`);
      // List what files are actually available
      try {
        const files = await Array.fromAsync(
          new Bun.Glob('*').scan({ cwd: './dist/assets/cast' })
        );
        console.log(`[Static] Available files:`, files);
      } catch (err) {
        console.log(`[Static] Error listing files:`, err);
      }
      return null; // Let it fall through instead of 404
    }
  }

  console.log(`[Static] No file found for: ${path}`);
  return null;
}

async function renderHTML(path: string, initialData: any) {
  // Reload manifest in development mode for hot updates
  if (isDev) {
    manifest = await loadManifest();
  }

  const [ssrErr, stream] = await safeWrapAsync(
    renderToReadableStream(
      <StaticRouter location={path}>
        <App initialData={initialData} />
      </StaticRouter>,
      {
        bootstrapModules: [`/assets/cast/${manifest['client.js'] || 'client.js'}`],
        onError(error) {
          console.error('[SSR Stream Error]', error);
        },
      },
    ),
  );

  if (ssrErr || !stream) {
    console.error('[SSR Error]', ssrErr);
    throw new Error('SSR failed');
  }

  // Convert React stream to string to avoid ReadableStream lock issues
  const chunks: Uint8Array[] = [];
  const reader = stream.getReader();
  
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }

  // Combine all chunks into a single string
  const decoder = new TextDecoder();
  const reactHTML = chunks.map(chunk => decoder.decode(chunk)).join('');

  // Create complete HTML document with SSR content
  const cssPath = `/assets/cast/${manifest['index.css'] || 'index.css'}`;
  
  const fullHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vibez Cast Receiver</title>
    <link rel="stylesheet" href="${cssPath}" />
    <!-- Google Cast Receiver SDK -->
    <script src="//www.gstatic.com/cast/sdk/libs/caf_receiver/v3/cast_receiver_framework.js"></script>
    <!-- YouTube IFrame API -->
    <script src="https://www.youtube.com/iframe_api"></script>
    <script id="ssr-data" type="application/json">${JSON.stringify(initialData)}</script>
  </head>
  <body>
    <div id="static-loading" style="position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; z-index: 0; background-color: #0d0d0f; color: white;">
      <div style="animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite; font-weight: bold; font-size: 2rem; background: linear-gradient(to right, #ec4899, #06b6d4); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">Vibez Cast</div>
      <p style="margin-top: 1rem; opacity: 0.7;">Initializing...</p>
    </div>
    <div id="root" style="position: relative; z-index: 1;">${reactHTML}</div>
  </body>
</html>`;

  return new Response(fullHTML, {
    headers: { 'Content-Type': 'text/html; charset=utf-8' }
  });
}

Bun.serve({
  port,
  async fetch(req, server) {
    const url = new URL(req.url);
    const path = url.pathname;

    console.log(`[Cast Server] Request: ${req.method} ${path}`);

    if (isDev && path === '/__hmr') {
      return server.upgrade(req)
        ? undefined
        : new Response('Upgrade failed', { status: 400 });
    }

    const staticResponse = await handleStaticFiles(path);
    if (staticResponse) {
      console.log(`[Cast Server] Serving static file: ${path}`);
      return staticResponse;
    }

    // If it's an asset request that we couldn't serve, return proper 404
    if (path.startsWith('/assets/cast/')) {
      console.log(`[Cast Server] Asset not found, returning 404: ${path}`);
      return new Response('Asset Not Found', { 
        status: 404,
        headers: { 'Content-Type': 'text/plain' }
      });
    }

    console.log(`[Cast Server] Rendering HTML for: ${path}`);
    const initialData = {};

    try {
      return await renderHTML(path, initialData);
    } catch (error) {
      console.error('[SSR Error]', error);
      return new Response('Internal Server Error', { status: 500 });
    }
  },
  websocket: {
    message() {},
    open() {
      console.log('[HMR] Cast Client connected');
    },
  },
});

console.log(
  `[SSR] Cast Server running at http://localhost:${port} (Dev: ${isDev})`,
);
console.log(`[SSR] Cast Server should handle /assets/cast/* requests`);
console.log(`[SSR] Current working directory: ${process.cwd()}`);
