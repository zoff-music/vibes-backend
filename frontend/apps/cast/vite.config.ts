import path from 'node:path';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react(), tailwindcss()] as any,
  root: '.',
  publicDir: 'public',
  base: '/casting/receiver/',
  server: {
    host: true,
    port: 3003,
    strictPort: true,
    hmr: {
      host: 'localhost',
      protocol: 'wss',
      clientPort: 443,
      path: '/casting/receiver/__hmr',
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
