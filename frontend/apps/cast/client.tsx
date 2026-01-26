import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './src/styles/index.css';
import App from './src/App';

const rootElement = document.getElementById('root');
if (rootElement) {
  // Hide the loading screen once React mounts
  const loadingElement = document.getElementById('static-loading');
  if (loadingElement) {
    loadingElement.style.display = 'none';
  }

  // Create root and render the app
  const root = createRoot(rootElement);
  root.render(
    <StrictMode>
      <div className="hidden p-4">Debug</div>
      <App />
    </StrictMode>,
  );
}
