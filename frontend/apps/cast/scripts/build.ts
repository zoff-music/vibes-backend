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
  `[Build] ${isWatch ? 'Watching' : 'Building'} static cast app with defines:`,
  Object.keys(defines),
);

async function generateStaticHTML(jsFilename: string, cssFilename: string) {
  const html = `<!DOCTYPE html>
<html lang="en" class="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vibez Cast Receiver</title>
    <link rel="stylesheet" href="./${cssFilename}" />
    <!-- Google Cast Receiver SDK -->
    <script src="//www.gstatic.com/cast/sdk/libs/caf_receiver/v3/cast_receiver_framework.js"></script>
    <!-- YouTube IFrame API -->
    <script src="https://www.youtube.com/iframe_api"></script>
  </head>
  <body>
    <div id="static-loading" style="position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; z-index: 0; background-color: #0d0d0f; color: white;">
      <div style="animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite; font-weight: bold; font-size: 2rem; background: linear-gradient(to right, #ec4899, #06b6d4); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">Vibez Cast</div>
      <p style="margin-top: 1rem; opacity: 0.7;">Initializing...</p>
    </div>
    <div id="root"></div>
    <script type="module" src="./${jsFilename}"></script>
  </body>
</html>`;

  await Bun.write('./dist/index.html', html);
  console.log('[Build] Generated static HTML file');
}

async function runBuild() {
  const result = await Bun.build({
    entrypoints: ['./client.tsx'],
    outdir: './dist',
    minify: isProd,
    define: defines,
    naming: isProd ? '[name]-[hash].[ext]' : '[name].[ext]', // Use hashes in production
    splitting: false, // Keep it simple for static build
  });

  if (!result.success) {
    console.error('Build failed');
    for (const message of result.logs) {
      console.error(message);
    }
    if (!isWatch) process.exit(1);
  } else {
    console.log(`[Build] Success! ${new Date().toLocaleTimeString()}`);

    // Copy public files to dist directory
    const publicDir = join(import.meta.dir, '../public');
    if (existsSync(publicDir)) {
      const [copyErr, _] = await safeWrapAsync(
        Bun.spawn(['cp', '-r', 'public/.', 'dist/'], {
          cwd: join(import.meta.dir, '..'),
        }).exited,
      );
      if (copyErr) {
        console.error('Failed to copy public files', copyErr);
      }
    }

    // Find the built files
    let jsFilename = 'client.js';
    let cssFilename = 'index.css';

    for (const output of result.outputs) {
      const filename = output.path.split('/').pop() || '';
      console.log(`[Build] Processing output: ${filename}`);

      if (filename.includes('client') && filename.endsWith('.js')) {
        jsFilename = filename;
      }
    }

    // Check for CSS files - prioritize tailwind.css produced by our build:css script
    console.log(`[Build] Scanning for CSS files in ./dist...`);
    const cssFiles = await Array.fromAsync(
      new Bun.Glob('*.css').scan({ cwd: './dist' }),
    );
    console.log(`[Build] Found CSS files:`, cssFiles);

    cssFilename =
      cssFiles.find((f) => f === 'tailwind.css') ||
      cssFiles.find((f) => f.startsWith('client')) ||
      cssFiles[0] ||
      'index.css';

    if (cssFilename) {
      console.log(`[Build] Selected CSS file for HTML: ${cssFilename}`);
    }

    // Generate static HTML file
    await generateStaticHTML(jsFilename, cssFilename);

    console.log(`[Build] Static build complete:`);
    console.log(`  - HTML: dist/index.html`);
    console.log(`  - JS: dist/${jsFilename}`);
    console.log(`  - CSS: dist/${cssFilename}`);

    // Trigger HMR reload in development
    if (isWatch && (globalThis as any).triggerHMR) {
      console.log(`[Build] Triggering HMR reload...`);
      (globalThis as any).triggerHMR();
    }
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
