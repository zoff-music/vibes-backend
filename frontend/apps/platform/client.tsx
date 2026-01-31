import {
  applyConsoleLogGuard,
  isTruthyFlag,
  safeWrap,
  safeWrapAsync,
} from '@vibez/shared';
import { type ComponentType, StrictMode } from 'react';
import { createRoot, hydrateRoot } from 'react-dom/client';
import {
  createBrowserRouter,
  matchRoutes,
  type RouteObject,
  RouterProvider,
} from 'react-router';
import type { SSRInitialData } from './src/App';
import { createClientRoutes } from './src/routes.client';
import './src/index.css';

const debugEnabled = isTruthyFlag(import.meta.env.VITE_DEBUG);
applyConsoleLogGuard(debugEnabled);

// Check if we're in development mode with Vite HMR
const isDev = import.meta.env.DEV;
const isSSR =
  typeof window !== 'undefined' && document.getElementById('ssr-data');

// Read initial data from script tag if present (SSR mode)
let initialData: SSRInitialData | undefined;
if (isSSR) {
  const dataElement = document.getElementById('ssr-data');
  const [parseErr, data] = safeWrap(() =>
    JSON.parse(dataElement?.textContent || '{}'),
  );

  if (parseErr) {
    console.error('Failed to parse initial data:', parseErr);
  } else {
    initialData = data as SSRInitialData;
  }
}

const rootElement = document.getElementById('root');
if (rootElement) {
  const routes = createClientRoutes(initialData);

  const preloadLazyRoutes = async (routeList: RouteObject[]) => {
    const matches = matchRoutes(routeList, window.location.pathname);
    if (!matches || matches.length === 0) return;

    for (const match of matches) {
      if (typeof match.route.lazy !== 'function') {
        continue;
      }

      const lazyRoute = match.route.lazy;
      const [loadErr, module] = await safeWrapAsync(lazyRoute());

      if (loadErr || !module) {
        if (loadErr) {
          console.error('[Router] Failed to preload route chunk', loadErr);
        }
        continue;
      }

      if (typeof module !== 'object') {
        continue;
      }

      const moduleRecord = module as {
        Component?: ComponentType;
        default?: ComponentType;
      };
      const resolvedComponent =
        moduleRecord.Component ?? moduleRecord.default ?? null;

      if (resolvedComponent) {
        match.route.Component = resolvedComponent;
      }

      match.route.lazy = undefined;
    }
  };

  const renderApp = (
    routerInstance: ReturnType<typeof createBrowserRouter>,
  ) => {
    const AppComponent = (
      <StrictMode>
        <RouterProvider router={routerInstance} />
      </StrictMode>
    );

    if (isSSR && !isDev) {
      // Production SSR: hydrate the server-rendered content
      const originalConsoleError = console.error;
      console.error = (...args) => {
        if (
          typeof args[0] === 'string' &&
          (args[0].includes('Hydration failed') ||
            args[0].includes('hydration') ||
            args[0].includes('did not match'))
        ) {
          // Suppress hydration warnings in development
          return;
        }
        originalConsoleError.apply(console, args);
      };

      hydrateRoot(rootElement, AppComponent);
      return;
    }

    // Development or client-only: use createRoot for better HMR
    const root = createRoot(rootElement);
    root.render(AppComponent);
  };

  if (isSSR && !isDev) {
    const start = async () => {
      await preloadLazyRoutes(routes);
      const router = createBrowserRouter(routes);
      renderApp(router);
    };

    start();
  } else {
    const router = createBrowserRouter(routes);
    renderApp(router);
  }
}

// Enable HMR in development
if (isDev && import.meta.hot) {
  import.meta.hot.accept();
}
