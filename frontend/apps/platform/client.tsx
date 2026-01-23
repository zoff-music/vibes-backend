import { safeWrap } from '@vibez/shared';
import { StrictMode } from 'react';
import { createRoot, hydrateRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import App from './src/App';
import './src/index.css';

// Check if we're in development mode with Vite HMR
const isDev = import.meta.env.DEV;
const isSSR =
  typeof window !== 'undefined' && document.getElementById('ssr-data');

// Read initial data from script tag if present (SSR mode)
let initialData = {};
if (isSSR) {
  const dataElement = document.getElementById('ssr-data');
  const [parseErr, data] = safeWrap(() =>
    JSON.parse(dataElement?.textContent || '{}'),
  );

  if (parseErr) {
    console.error('Failed to parse initial data:', parseErr);
  } else {
    initialData = data;
  }
}

const rootElement = document.getElementById('root');
if (rootElement) {
  const AppComponent = (
    <StrictMode>
      <BrowserRouter>
        <App initialData={initialData} />
      </BrowserRouter>
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
  } else {
    // Development or client-only: use createRoot for better HMR
    const root = createRoot(rootElement);
    root.render(AppComponent);
  }
}

// Enable HMR in development
if (isDev && import.meta.hot) {
  import.meta.hot.accept();
}
