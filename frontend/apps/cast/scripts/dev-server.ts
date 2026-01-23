const port = process.env.PORT || 3000;
const _isDev = true;

console.log('[Cast Dev Server] Starting development server...');

// Simple static file server for development
Bun.serve({
  port,
  async fetch(req, server) {
    const url = new URL(req.url);
    const path = url.pathname;

    console.log(`[Cast Dev Server] Request: ${req.method} ${path}`);

    // Handle HMR WebSocket upgrade
    if (path === '/__hmr') {
      return server.upgrade(req)
        ? undefined
        : new Response('Upgrade failed', { status: 400 });
    }

    // Serve static files from dist directory
    if (path === '/' || path === '/index.html') {
      const indexFile = Bun.file('./dist/index.html');
      if (await indexFile.exists()) {
        // Inject HMR script into HTML for development
        const html = await indexFile.text();
        const hmrScript = `
<script>
  // Simple HMR client that works through proxy
  const isProxied = window.location.protocol === 'https:' && window.location.hostname === 'localhost';
  const wsUrl = isProxied 
    ? 'wss://localhost/casting/receiver/__hmr'
    : 'ws://localhost:${port}/__hmr';
  
  const ws = new WebSocket(wsUrl);
  ws.onmessage = (event) => {
    if (event.data === 'reload') {
      console.log('[Cast HMR] Reloading page...');
      window.location.reload();
    }
  };
  ws.onopen = () => console.log('[Cast HMR] Connected to', wsUrl);
  ws.onclose = () => console.log('[Cast HMR] Disconnected');
  ws.onerror = (error) => console.log('[Cast HMR] Error:', error);
</script>`;

        const modifiedHtml = html.replace('</body>', `${hmrScript}</body>`);
        return new Response(modifiedHtml, {
          headers: { 'Content-Type': 'text/html; charset=utf-8' },
        });
      }
    }

    // Serve other static files (JS, CSS, etc.)
    const filePath = path === '/' ? '/index.html' : path;
    const file = Bun.file(`./dist${filePath}`);

    if (await file.exists()) {
      const headers: Record<string, string> = {};

      if (filePath.endsWith('.css')) {
        headers['Content-Type'] = 'text/css';
      } else if (filePath.endsWith('.js')) {
        headers['Content-Type'] = 'application/javascript';
      } else if (filePath.endsWith('.html')) {
        headers['Content-Type'] = 'text/html; charset=utf-8';
      }

      return new Response(file, { headers });
    }

    // 404 for missing files
    return new Response('Not Found', { status: 404 });
  },

  websocket: {
    message() {},
    open(ws) {
      console.log('[HMR] Client connected');
      // Store the websocket for later use
      (globalThis as any).hmrClients =
        (globalThis as any).hmrClients || new Set();
      (globalThis as any).hmrClients.add(ws);
    },
    close(ws) {
      console.log('[HMR] Client disconnected');
      if ((globalThis as any).hmrClients) {
        (globalThis as any).hmrClients.delete(ws);
      }
    },
  },
});

// Function to trigger HMR reload
(globalThis as any).triggerHMR = () => {
  const clients = (globalThis as any).hmrClients;
  if (clients) {
    for (const client of clients) {
      try {
        client.send('reload');
      } catch (err) {
        console.log('[HMR] Failed to send reload signal:', err);
      }
    }
  }
};

console.log(`[Cast Dev Server] Running at http://localhost:${port}`);
console.log(`[Cast Dev Server] Serving static files from ./dist/`);
console.log(
  `[Cast Dev Server] HMR enabled - files will auto-reload on changes`,
);
