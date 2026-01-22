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
    // For platform client, we don't bake in API_URL - it should use origin fallback
    if (key === 'VITE_API_URL') continue;
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
    outdir: './dist/assets/platform',
    minify: isProd,
    define: defines,
    naming: isProd ? '[dir]/[name]-[hash].[ext]' : '[dir]/[name].[ext]', // Only use hashes in production
    splitting: false, // Disable code splitting to avoid duplicate exports
  });

  if (!result.success) {
    console.error('Build failed');
    for (const message of result.logs) {
      console.error(message);
    }
    if (!isWatch) process.exit(1);
  } else {
    console.log(`[Build] Success! ${new Date().toLocaleTimeString()}`);

    // Copy public files to platform assets directory
    const publicDir = join(import.meta.dir, '../public');
    if (existsSync(publicDir)) {
      const [copyErr, _] = await safeWrapAsync(
        Bun.spawn(['cp', '-r', 'public/.', 'dist/assets/platform/'], {
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
        manifest['main.js'] = filename;
      }
    }

    // Check for CSS files in the output directory and hash them for production
    const cssFiles = await Array.fromAsync(
      new Bun.Glob('*.css').scan({ cwd: './dist/assets/platform' }),
    );

    if (cssFiles.length > 0 && cssFiles[0]) {
      const originalName = cssFiles[0];
      if (isProd && originalName === 'index.css') {
        const cssContent = await Bun.file(join('./dist/assets/platform', originalName)).arrayBuffer();
        const hash = Bun.hash(cssContent).toString(16).slice(0, 8);
        const hashedName = `index-${hash}.css`;
        
        const [renameErr] = await safeWrapAsync(
          Bun.spawn(['mv', originalName, hashedName], {
            cwd: './dist/assets/platform',
          }).exited,
        );

        if (!renameErr) {
          manifest['index.css'] = hashedName;
          console.log(`[Build] Hashed CSS: ${originalName} -> ${hashedName}`);
        } else {
          manifest['index.css'] = originalName;
          console.warn(`[Build] Failed to rename CSS, using original: ${originalName}`);
        }
      } else {
        manifest['index.css'] = originalName;
        console.log(`[Build] Found CSS file: ${originalName}`);
      }
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
await runBuild();

// Exit explicitly in non-watch mode
if (!isWatch) {
  process.exit(0);
}
