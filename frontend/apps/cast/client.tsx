import { applyConsoleLogGuard, isTruthyFlag, safeWrap } from '@vibez/shared';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './src/styles/index.css';
import App from './src/App';

const debugEnabled = isTruthyFlag(import.meta.env.VITE_DEBUG);
applyConsoleLogGuard(debugEnabled);

// Wrap initialization in safeWrap to report errors
const [err] = safeWrap(() => {
  const rootElement = document.getElementById('root');
  if (rootElement) {
    // Hide the loading screen once React mounts
    const loadingElement = document.getElementById('static-loading');
    if (loadingElement) {
      loadingElement.style.display = 'none';
    }

    // Create root and render the app
    const root = createRoot(rootElement);

    const [renderErr] = safeWrap(() => {
      root.render(
        <StrictMode>
          <div className="hidden p-4">Debug</div>
          <App />
        </StrictMode>,
      );
    });

    if (renderErr) {
      console.error('Failed to render app', renderErr);
      document.body.innerHTML = `<div style="color:red; font-size: 24px; padding: 20px;">Failed to render: ${renderErr}</div>`;
    }
  }
});

if (err) {
  // Catch top-level errors (e.g. imports)
  console.error('Fatal startup error', err);
  const errDiv = document.createElement('div');
  errDiv.style.color = 'red';
  errDiv.style.fontSize = '24px';
  errDiv.style.padding = '20px';
  errDiv.style.backgroundColor = 'white';
  errDiv.style.zIndex = '9999';
  errDiv.style.position = 'absolute';
  errDiv.style.top = '0';
  errDiv.style.margin = '20px';
  errDiv.innerText = `Startup Error: ${err instanceof Error ? err.message : String(err)}`;
  document.body.appendChild(errDiv);
}
