import path from 'node:path';
import { reactRouter } from '@react-router/dev/vite';
import { defineConfig } from 'vite';

// https://vitejs.dev/config/
export default defineConfig(({ command }) => {
  const isBuild = command === 'build';
  const nodeEnv =
    process.env.NODE_ENV || (isBuild ? 'production' : 'development');

  return {
    plugins: [reactRouter()],
    root: '.',
    publicDir: 'public',
    server: {
      port: 3001,
      host: '0.0.0.0',
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
