import { renderToReadableStream } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import App from './src/App';
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

const port = process.env.PORT || 3000;
const isDev = process.env.NODE_ENV !== 'production';

async function handleStaticFiles(path: string) {
    const publicFile = Bun.file(`./public${path}`);
    if (await publicFile.exists()) return new Response(publicFile);

    const distFile = Bun.file(`./dist/client${path}`);
    if (await distFile.exists()) return new Response(distFile);

    return null;
}

async function getInitialData(path: string, req: Request) {
    const roomMatch = path.match(/^\/room\/([^\/]+)$/);
    if (!roomMatch || roomMatch[1] === 'create') return { data: {}, redirect: null };

    const roomId = roomMatch[1];
    console.log(`[SSR] Fetching room data for ${roomId}`);
    const [err, room] = await api.get('/rooms/{id}', { id: roomId });

    if (err || !room) {
        console.log(`[SSR] Room ${roomId} not found, redirecting to create...`);
        const createUrl = new URL('/room/create', req.url);
        createUrl.searchParams.set('name', roomId);
        return { data: {}, redirect: Response.redirect(createUrl.toString(), 302) };
    }

    return { data: { room }, redirect: null };
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

        const { data: initialData, redirect } = await getInitialData(path, req);
        if (redirect) return redirect;

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
        open() { console.log('[HMR] Client connected'); },
    },
});

console.log(`[SSR] Server running at http://localhost:${port} (Dev: ${isDev})`);
