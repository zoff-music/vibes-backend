import type { ReactNode } from 'react';
import type { LinksFunction, LoaderFunctionArgs } from 'react-router';
import {
  Links,
  Meta,
  Scripts,
  ScrollRestoration,
  useLoaderData,
} from 'react-router';
import App from './App';
import stylesUrl from './index.css?url';
import { getThemeFromCookies } from './ssr/theme.server';

export const links: LinksFunction = () => [
  { rel: 'stylesheet', href: stylesUrl },
];

export interface RootLoaderData {
  theme: 'light' | 'dark' | 'auto';
}

export async function loader({ request }: LoaderFunctionArgs) {
  const cookieHeader = request.headers.get('cookie') ?? null;
  const theme = getThemeFromCookies(cookieHeader);
  return { theme } satisfies RootLoaderData;
}

export function Layout({ children }: { children: ReactNode }) {
  const loaderData = useLoaderData<typeof loader>() as
    | RootLoaderData
    | undefined;
  const themeId = loaderData?.theme ?? 'auto';
  const themeClass =
    themeId === 'dark' ? 'dark' : themeId === 'light' ? 'theme-light' : '';
  const initialDataJson = JSON.stringify(loaderData ?? {});

  return (
    <html lang="en" className={themeClass}>
      <head>
        <meta charSet="UTF-8" />
        <link
          rel="icon"
          type="image/png"
          href="/favicon-96x96.png"
          sizes="96x96"
        />
        <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
        <link rel="shortcut icon" href="/favicon.ico" />
        <link
          rel="apple-touch-icon"
          sizes="180x180"
          href="/apple-touch-icon.png"
        />
        <link rel="manifest" href="/site.webmanifest" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>ゾフ - Shared Music Queue</title>
        <script
          defer
          data-domain="zoff.me"
          src="https://analytics.zoff.me/js/script.outbound-links.tagged-events.js"
        />
        <script
          // biome-ignore lint/security/noDangerouslySetInnerHtml: inline plausible bootstrap (first-party analytics domain)
          dangerouslySetInnerHTML={{
            __html:
              'window.plausible = window.plausible || ((...args) => { window.plausible.q = window.plausible.q || []; window.plausible.q.push(args); });',
          }}
        />
        <Meta />
        <Links />
      </head>
      <body>
        <div id="root">{children}</div>
        <script
          id="ssr-data"
          type="application/json"
          // biome-ignore lint/security/noDangerouslySetInnerHtml: SSR hydration data
          dangerouslySetInnerHTML={{ __html: initialDataJson }}
        />
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function Root() {
  return <App />;
}
