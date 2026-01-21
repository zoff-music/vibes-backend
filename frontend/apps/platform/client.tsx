import { StrictMode } from 'react';
import { hydrateRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import { safeWrap } from '@vibez/shared';
import App from './src/App';

// Read initial data from script tag if present
const dataElement = document.getElementById('ssr-data');
const [parseErr, initialData] = safeWrap(() => JSON.parse(dataElement?.textContent || '{}'));

if (parseErr) {
    console.error('Failed to parse initial data:', parseErr);
}

hydrateRoot(
    document,
    <StrictMode>
        <BrowserRouter>
            <App initialData={initialData} />
        </BrowserRouter>
    </StrictMode>
);
