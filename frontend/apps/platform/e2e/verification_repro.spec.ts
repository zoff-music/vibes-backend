import { test, expect } from '@playwright/test';

test('verify network requests on room join', async ({ page }) => {
  // Mock interactions to avoid external dependencies if possible, 
  // but for this "verification" we want to see actual network traffic behavior 
  // against a running local backend if available, or just potential duplicate triggers.
  // Assuming the app is running locally.
  
  const requestCounts = {
    events: 0,
    ws_events: 0,
    providers: 0,
    authorizations: 0,
    room: 0,
    sessions: 0,
    songs: 0,
  };

  await page.route('**/*', async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    
    if (url.includes('/events')) {
        requestCounts.events++;
    }
    if (url.includes('/providers')) {
        requestCounts.providers++;
    }
    if (url.includes('/authorizations')) {
        requestCounts.authorizations++;
    }
    if (url.includes('/rooms/') && !url.includes('/events') && !url.includes('/sessions') && !url.includes('/songs')) {
        requestCounts.room++;
    }
    if (url.includes('/sessions') && method === 'POST') {
        requestCounts.sessions++;
    }
    if (url.includes('/songs')) {
        requestCounts.songs++;
    }
    
    await route.continue();
  });

  // Navigate to a random room to ensure we hit the "join" logic
  const roomId = `test-room-${Math.floor(Math.random() * 10000)}`;
  await page.goto(`/room/${roomId}`);
  
  // Wait for the room to load (look for specific UI element)
  await page.waitForSelector('h1', { timeout: 10000 });
  
  // Wait a bit more for any delayed effects/rerenders to fire requests
  await page.waitForTimeout(3000);

  console.log('Request Counts:', requestCounts);

  // Expect reasonable limits. 
  // Note: specific counts might vary slightly due to strict mode in dev (double invoke),
  // but we want to fail if it's "excessive" like the user reported (3-4 fetch calls).
  // In a perfect world: 1 of each. In React Stict Mode Dev: maybe 2.
  // The user reported: 
  // 4 events
  // 2 providers
  // 3 authorizations
  // 1 room
  // 1 songs
  // 1 sessions
  
  // We want to reduce this to:
  // 1 events (connection should persist or reuse)
  // 1 providers (cached)
  // 1 authorizations (cached)
  // 1 room (deduped)
  // 1 sessions (deduped or managed)

  // Assertions (soft for now to see what fails in the first run)
  expect(requestCounts.events, 'Excessive SSE connections').toBeLessThanOrEqual(2); 
  expect(requestCounts.providers, 'Excessive providers fetch').toBeLessThanOrEqual(1);
  expect(requestCounts.authorizations, 'Excessive authorizations fetch').toBeLessThanOrEqual(1);
  expect(requestCounts.sessions, 'Excessive session joins').toBeLessThanOrEqual(1);
  
});
