import path from 'node:path';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()] as any,
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
    hmr: {
      port: 3002,
      host: 'localhost', // Use localhost instead of 0.0.0.0
      clientPort: 443, // Tell client to connect to HTTPS port
      path: '/vite-hmr', // Use a specific path
    },
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
