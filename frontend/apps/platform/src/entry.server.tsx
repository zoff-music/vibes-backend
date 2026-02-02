import type { EntryContext } from 'react-router';
import { ServerRouter } from 'react-router';
import { renderToReadableStream } from 'react-dom/server';

export default async function handleRequest(
  request: Request,
  statusCode: number,
  headers: Headers,
  context: EntryContext,
) {
  const stream = await renderToReadableStream(
    <ServerRouter context={context} url={request.url} />,
    {
      signal: request.signal,
    },
  );

  headers.set('Content-Type', 'text/html');
  return new Response(stream, { status: statusCode, headers });
}
