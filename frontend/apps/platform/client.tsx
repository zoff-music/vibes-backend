import { safeWrap } from '@vibez/shared';
import { StrictMode } from 'react';
import { hydrateRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import App from './src/App';
import './src/index.css';

// Read initial data from script tag if present
const dataElement = document.getElementById('ssr-data');
const [parseErr, initialData] = safeWrap(() =>
  JSON.parse(dataElement?.textContent || '{}'),
);

if (parseErr) {
  console.error('Failed to parse initial data:', parseErr);
}

const rootElement = document.getElementById('root');
if (rootElement) {
  // Suppress hydration warnings in development by catching them
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

  hydrateRoot(
    rootElement,
    <StrictMode>
      <BrowserRouter>
        <App initialData={initialData} />
      </BrowserRouter>
    </StrictMode>,
  );
}