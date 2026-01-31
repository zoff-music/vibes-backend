import path from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// https://vitejs.dev/config/
export default defineConfig(({ command, ssrBuild }) => {
  const isBuild = command === 'build';
  const isServerBuild = Boolean(ssrBuild);
  const isClientBuild = isBuild && !isServerBuild;
  const nodeEnv =
    process.env.NODE_ENV || (isBuild ? 'production' : 'development');

  return {
    plugins: [react()] as any,
    root: '.',
    publicDir: 'public',
    base: isClientBuild ? '/assets/platform/' : '/',
    build: isServerBuild
      ? {
          outDir: 'dist/server',
          emptyOutDir: true,
          target: 'esnext',
          rollupOptions: {
            output: {
              format: 'esm',
              entryFileNames: 'server.js',
            },
          },
        }
      : isClientBuild
        ? {
            outDir: 'dist/assets/platform',
            emptyOutDir: true,
            manifest: true,
            rollupOptions: {
              input: 'client.tsx',
              output: {
                entryFileNames: 'client-[hash].js',
                chunkFileNames: 'chunk-[hash].js',
                assetFileNames: '[name]-[hash].[ext]',
              },
            },
          }
        : {
            outDir: 'dist/assets/platform',
            emptyOutDir: false,
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
      dedupe: [
        'react',
        'react-dom',
        'zustand',
        '@vibez/shared',
        '@vibez/api',
        '@vibez/models',
        '@vibez/ui',
      ],
      preserveSymlinks: true,
    },
    ssr: {
      noExternal: ['@vibez/api', '@vibez/models', '@vibez/shared', '@vibez/ui'],
    },
    define: {
      'process.env.NODE_ENV': JSON.stringify(nodeEnv),
      'import.meta.env.VITE_CAST_APP_ID': JSON.stringify(
        process.env.CAST_APP_ID || '1FAF5D9F',
      ),
      'import.meta.env.VITE_CAST_RECEIVER_URL': JSON.stringify(
        process.env.CAST_RECEIVER_URL || '/casting/receiver/',
      ),
      'import.meta.env.VITE_FRONTEND_URL': JSON.stringify(
        process.env.FRONTEND_URL || 'http://localhost:3001',
      ),
      'import.meta.env.VITE_DEVELOPMENT_MODE': JSON.stringify(
        process.env.DEVELOPMENT_MODE ||
          (nodeEnv !== 'production' ? 'true' : 'false'),
      ),
      'import.meta.env.VITE_DEBUG': JSON.stringify(
        process.env.VITE_DEBUG || process.env.DEBUG || 'false',
      ),
    },
  };
});
