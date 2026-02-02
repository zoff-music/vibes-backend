import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { applyConsoleLogGuard, isTruthyFlag } from '@vibez/shared';
import { createRequestHandler } from 'react-router';

const port = Number(process.env.PORT || 3000);
const nodeEnv = (process.env.NODE_ENV || '').toLowerCase();
const isDev = nodeEnv !== 'production';
const debugEnabled = isTruthyFlag(process.env.VITE_DEBUG ?? process.env.DEBUG);
applyConsoleLogGuard(debugEnabled);

const loadBuild = async () => {
  const mod = isDev
    ? await import('virtual:react-router/server-build')
    : await import(
        pathToFileURL(join(process.cwd(), 'dist/server/index.js')).href
      );
  return (mod as { default?: unknown }).default ?? mod;
};

const handler = createRequestHandler(
  loadBuild,
  isDev ? 'development' : 'production',
);

const staticMappings = [
  {
    prefix: '/assets/platform/',
    dir: join(process.cwd(), 'dist/assets/platform'),
  },
  { prefix: '/assets/', dir: join(process.cwd(), 'dist/assets') },
  { prefix: '/', dir: join(process.cwd(), 'dist/client') },
  { prefix: '/', dir: join(process.cwd(), 'public') },
];

function tryFile(pathname: string) {
  for (const mapping of staticMappings) {
    if (!pathname.startsWith(mapping.prefix)) continue;
    const relPath = pathname.slice(mapping.prefix.length);
    if (!relPath || relPath.includes('..')) continue;
    const filePath = join(mapping.dir, relPath);
    if (!existsSync(filePath)) continue;
    return new Response(Bun.file(filePath));
  }

  return null;
}

Bun.serve({
  port,
  async fetch(request) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    const staticResponse = tryFile(pathname);
    if (staticResponse) return staticResponse;

    return handler(request);
  },
});

console.log(`[SSR] Running at http://localhost:${port} (dev: ${isDev})`);
