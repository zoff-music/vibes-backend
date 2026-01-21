import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { safeWrapAsync } from '@vibez/shared';

const isWatch = process.argv.includes('--watch');
const isProd = process.env.NODE_ENV === 'production';

// Use Bun.env (Bun automatically loads .env from project root and parent dirs)
const envVars = Bun.env;

// Map variables to VITE_ prefix
const defines: Record<string, string> = {
  'process.env.NODE_ENV': JSON.stringify(isProd ? 'production' : 'development'),
};

// Map all VITE_ variables and common ones
for (const [key, value] of Object.entries(envVars)) {
  if (key.startsWith('VITE_')) {
    defines[`import.meta.env.${key}`] = JSON.stringify(value);
  } else if (
    ['CAST_APP_ID', 'CAST_RECEIVER_URL', 'FRONTEND_URL'].includes(key)
  ) {
    defines[`import.meta.env.VITE_${key}`] = JSON.stringify(value);
  }
}

// Default values if missing
if (!defines['import.meta.env.VITE_CAST_APP_ID']) {
  defines['import.meta.env.VITE_CAST_APP_ID'] = JSON.stringify('1FAF5D9F');
}
if (!defines['import.meta.env.VITE_CAST_RECEIVER_URL']) {
  defines['import.meta.env.VITE_CAST_RECEIVER_URL'] =
    JSON.stringify('/casting/receiver/');
}

console.log(
  `[Build] ${isWatch ? 'Watching' : 'Building'} with defines:`,
  Object.keys(defines),
);

async function runBuild() {
  const result = await Bun.build({
    entrypoints: ['./client.tsx'],
    outdir: './dist/assets/cast',
    minify: isProd,
    define: defines,
    naming: '[dir]/[name]-[hash].[ext]', // Always use hashes for cache busting
    splitting: true,
  });

  if (!result.success) {
    console.error('Build failed');
    for (const message of result.logs) {
      console.error(message);
    }
    if (!isWatch) process.exit(1);
  } else {
    console.log(`[Build] Success! ${new Date().toLocaleTimeString()}`);

    // Copy public files to cast assets directory
    const publicDir = join(import.meta.dir, '../public');
    if (existsSync(publicDir)) {
      const [copyErr, _] = await safeWrapAsync(
        Bun.spawn(['cp', '-r', 'public/.', 'dist/assets/cast/'], {
          cwd: join(import.meta.dir, '..'),
        }).exited,
      );
      if (copyErr) {
        console.error('Failed to copy public files', copyErr);
      }
    }

    // Write build manifest for SSR to know the hashed filenames
    const manifest: Record<string, string> = {};

    // Find the built files and map them
    for (const output of result.outputs) {
      const filename = output.path.split('/').pop() || '';
      const fullPath = output.path;

      console.log(`[Build] Processing output: ${fullPath} -> ${filename}`);

      if (filename.includes('client') && filename.endsWith('.js')) {
        manifest['client.js'] = filename;
      }
    }

    // Check for CSS files in the output directory
    const cssFiles = await Array.fromAsync(
      new Bun.Glob('*.css').scan({ cwd: './dist/assets/cast' }),
    );

    if (cssFiles.length > 0) {
      manifest['index.css'] = cssFiles[0]; // Take the first CSS file
      console.log(`[Build] Found CSS file: ${cssFiles[0]}`);
    }

    manifest.timestamp = Date.now().toString();

    console.log(`[Build] Writing manifest:`, manifest);
    await Bun.write('./dist/manifest.json', JSON.stringify(manifest, null, 2));
  }
}

if (isWatch) {
  // Use Node.js fs.watch since Bun.watch is not available in v1.3.6
  const fs = await import('node:fs');
  const watchDir = join(import.meta.dir, '..');

  console.log(`[Build] Using fs.watch, watching ${watchDir}`);

  const watcher = fs.watch(
    watchDir,
    { recursive: true },
    (_eventType, filename) => {
      if (
        filename &&
        (filename.endsWith('.tsx') ||
          filename.endsWith('.ts') ||
          filename.endsWith('.css') ||
          filename.includes('public/')) &&
        !filename.includes('dist/')
      ) {
        // Exclude dist directory to prevent rebuild loops
        console.log(`[Build] Change detected in ${filename}, rebuilding...`);
        runBuild();
      }
    },
  );

  // Handle cleanup
  process.on('SIGINT', () => {
    watcher.close();
    process.exit(0);
  });
}

// Always run initial build
runBuild();
