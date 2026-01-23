import * as path from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// Simple live reload plugin for proxy usage
function liveReloadPlugin() {
  return {
    name: 'live-reload',
    configureServer(server) {
      // Add live reload script to HTML
      server.middlewares.use((req, res, next) => {
        if (req.url === '/' || req.url?.endsWith('.html')) {
          const originalSend = res.send;
          res.send = function(body) {
            if (typeof body === 'string' && body.includes('<head>')) {
              const liveReloadScript = `
                <script>
                  // Simple live reload for proxy usage
                  let lastModified = Date.now();
                  setInterval(async () => {
                    try {
                      const response = await fetch('/__live_reload_check');
                      const data = await response.json();
                      if (data.modified > lastModified) {
                        console.log('[Live Reload] Reloading page...');
                        window.location.reload();
                      }
                    } catch (e) {
                      // Ignore errors
                    }
                  }, 1000);
                </script>
              `;
              body = body.replace('<head>', `<head>${liveReloadScript}`);
            }
            return originalSend.call(this, body);
          };
        }
        next();
      });

      // Add endpoint for checking modifications
      server.middlewares.use('/__live_reload_check', (req, res) => {
        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify({ modified: Date.now() }));
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), liveReloadPlugin()],
  root: '.',
  publicDir: 'public',
  build: {
    outDir: 'dist/assets/platform',
    emptyOutDir: true,
    rollupOptions: {
      input: 'client.tsx',
      output: {
        entryFileNames: '[name].js',
        chunkFileNames: '[name].js',
        assetFileNames: '[name].[ext]',
      },
    },
  },
  server: {
    port: 3001,
    host: '0.0.0.0',
    hmr: false, // Disable Vite's built-in HMR
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  css: {
    postcss: './postcss.config.js',
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify(
      process.env.NODE_ENV || 'development',
    ),
    'import.meta.env.VITE_CAST_APP_ID': JSON.stringify(
      process.env.CAST_APP_ID || '1FAF5D9F',
    ),
    'import.meta.env.VITE_CAST_RECEIVER_URL': JSON.stringify(
      process.env.CAST_RECEIVER_URL || '/casting/receiver/',
    ),
    'import.meta.env.VITE_FRONTEND_URL': JSON.stringify(
      process.env.FRONTEND_URL || 'http://localhost:3001',
    ),
  },
});