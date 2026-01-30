import path from 'node:path';
import legacy from '@vitejs/plugin-legacy';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    react(),
    legacy({
      targets: ['chrome >= 80'],
      additionalLegacyPolyfills: ['regenerator-runtime/runtime'],
    }),
  ] as any,
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
  build: {
    cssTarget: 'chrome80',
    outDir: 'dist',
    emptyOutDir: true,
  },
  define: {
    'import.meta.env.VITE_CAST_APP_ID': JSON.stringify(
      process.env.CAST_APP_ID || '1FAF5D9F',
    ),
    'import.meta.env.VITE_CAST_RECEIVER_URL': JSON.stringify(
      process.env.CAST_RECEIVER_URL || '/casting/receiver/',
    ),
    'import.meta.env.VITE_FRONTEND_URL': JSON.stringify(
      process.env.FRONTEND_URL || '',
    ),
    'import.meta.env.VITE_DEBUG': JSON.stringify(
      process.env.VITE_DEBUG || process.env.DEBUG || 'false',
    ),
    'import.meta.env.VITE_CAST_DEBUG_MODE': JSON.stringify(
      process.env.CAST_DEBUG_MODE || 'false',
    ),
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@vibez/ui': path.resolve(__dirname, '../../packages/ui/src'),
      '@vibez/shared': path.resolve(__dirname, '../../packages/shared/src'),
      '@vibez/models': path.resolve(__dirname, '../../packages/models/src'),
      '@vibez/api': path.resolve(__dirname, '../../packages/api/src'),
    },
  },
});
