import { renderToReadableStream } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';
import { safeWrapAsync } from '@vibez/shared';

const port = process.env.PORT || 3001;
const isDev = process.env.NODE_ENV !== 'production';

async function handleStaticFiles(path: string) {
    const publicFile = Bun.file(`./public${path}`);
    if (await publicFile.exists()) return new Response(publicFile);

    const distFile = Bun.file(`./dist/client${path}`);
    if (await distFile.exists()) return new Response(distFile);

    return null;
}

Bun.serve({
    port,
    async fetch(req, server) {
        const url = new URL(req.url);
        const path = url.pathname;

        if (isDev && path === '/__hmr') {
            return server.upgrade(req) ? undefined : new Response('Upgrade failed', { status: 400 });
        }

        const staticResponse = await handleStaticFiles(path);
        if (staticResponse) return staticResponse;

        const initialData = {};

        const [ssrErr, stream] = await safeWrapAsync(
            renderToReadableStream(
                <StaticRouter location={path}>
                    <App initialData={initialData} />
                </StaticRouter>,
                {
                    bootstrapModules: ['/client.js'],
                    onError(error) {
                        console.error('[SSR Stream Error]', error);
                    },
                }
            )
        );

        if (ssrErr || !stream) {
            console.error('[SSR Error]', ssrErr);
            return new Response('Internal Server Error', { status: 500 });
        }

        return new Response(stream, { headers: { 'Content-Type': 'text/html' } });
    },
    websocket: {
        message() { },
        open() { console.log('[HMR] Cast Client connected'); },
    },
});

console.log(`[SSR] Cast Server running at http://localhost:${port} (Dev: ${isDev})`);
