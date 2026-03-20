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

function generateCspNonce() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Buffer.from(bytes).toString('base64');
}

function buildCspHeader(nonce: string) {
  // Keep this aligned with .github/deploy/Caddyfile.tmpl (cast receiver CSP).
  // Platform needs a nonce because react-router emits inline scripts for SSR.
  return [
    "default-src 'self'",
    "base-uri 'self'",
    "object-src 'none'",
    "frame-ancestors 'none'",
    "img-src 'self' data: https:",
    "media-src 'self' https: blob:",
    "connect-src 'self' https: wss: https://analytics.zoff.me",
    `script-src 'self' 'nonce-${nonce}' https://analytics.zoff.me https://www.gstatic.com https://www.youtube.com https://www.youtube-nocookie.com https://sdk.scdn.co https://connect.soundcloud.com`,
    "style-src 'self' 'unsafe-inline' https: https://fonts.googleapis.com",
    "font-src 'self' data: https://fonts.gstatic.com",
    "frame-src https://www.youtube.com https://www.youtube-nocookie.com https://open.spotify.com https://w.soundcloud.com",
  ].join('; ');
}

function createNonceInjectTransform(nonce: string) {
  // Streaming-safe nonce injection: keep a small carry buffer so we can detect
  // `<script ...>` tags that might be split across chunks.
  let carry = '';
  const maxCarry = 8 * 1024;

  return new TransformStream<string, string>({
    transform(chunk, controller) {
      carry += chunk;

      while (true) {
        const idx = carry.search(/<script\b/i);
        if (idx === -1) {
          // Flush most of the buffer, keep a tail in case we split `<script`.
          if (carry.length > maxCarry) {
            controller.enqueue(carry.slice(0, carry.length - maxCarry));
            carry = carry.slice(carry.length - maxCarry);
          }
          return;
        }

        // Need the full open tag to decide whether to inject.
        const gt = carry.indexOf('>', idx);
        if (gt === -1) {
          // Emit everything before the start of the open tag, keep the rest.
          if (idx > 0) {
            controller.enqueue(carry.slice(0, idx));
            carry = carry.slice(idx);
          }
          if (carry.length > maxCarry) {
            // Defensive: avoid unbounded growth if markup is malformed.
            controller.enqueue(carry.slice(0, carry.length - maxCarry));
            carry = carry.slice(carry.length - maxCarry);
          }
          return;
        }

        const before = carry.slice(0, idx);
        if (before) controller.enqueue(before);

        const openTag = carry.slice(idx, gt + 1);
        const alreadyNonced = /\bnonce\s*=/.test(openTag);
        const isJsonScript = /\btype\s*=\s*["']application\/json["']/i.test(openTag);

        if (!alreadyNonced && !isJsonScript) {
          controller.enqueue(openTag.replace(/<script\b/i, `<script nonce="${nonce}"`));
        } else {
          controller.enqueue(openTag);
        }

        carry = carry.slice(gt + 1);
      }
    },
    flush(controller) {
      if (carry) controller.enqueue(carry);
      carry = '';
    },
  });
}

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

    const cspNonce = generateCspNonce();
    const response = await handler(request, { cspNonce });

    // Add CSP only to document responses (avoid polluting assets/JSON).
    const contentType = response.headers.get('content-type') ?? '';
    if (!contentType.includes('text/html')) return response;
    if (response.headers.has('content-security-policy')) return response;

    const headers = new Headers(response.headers);
    headers.set('Content-Security-Policy', buildCspHeader(cspNonce));
    // Body length changes when we inject nonces; don't lie.
    headers.delete('content-length');

    // React Router streaming SSR can emit inline <script> tags outside of the <Scripts />
    // component (e.g. stream chunk enqueues). Inject the nonce while streaming.
    const patchedBody = response.body
      ?.pipeThrough(new TextDecoderStream())
      .pipeThrough(createNonceInjectTransform(cspNonce))
      .pipeThrough(new TextEncoderStream());

    return new Response(patchedBody, {
      status: response.status,
      statusText: response.statusText,
      headers,
    });
  },
});

console.log(`[SSR] Running at http://localhost:${port} (dev: ${isDev})`);
