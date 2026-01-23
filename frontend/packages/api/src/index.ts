// Global type declaration for Node.js process in Bun environment
declare const process:
  | {
      env: Record<string, string | undefined>;
    }
  | undefined;

import {
  addSongRequestSchema,
  connectedSchema,
  createRoomRequestSchema,
  createSessionRequestSchema,
  emptyObjectSchema,
  messageResponseSchema,
  playbackStateSchema,
  providersSchema,
  providerTokenSchema,
  roomActionRequestSchema,
  roomSchema,
  roomUpdateSchema,
  searchQuerySchema,
  searchResponseSchema,
  sessionResponseSchema,
  songSchema,
  songsListSchema,
  spotifyTokenSchema,
  usersUpdateSchema,
  youTubeSearchQuerySchema,
  youTubeSearchResponseSchema,
  youTubeVideoSchema,
} from '@vibez/models';
import { safeWrapAsync } from '@vibez/shared';
import { RequestClient, type RequestDefinitions } from 'wiretyped';

const API_BASE_PATH = '/api/v1';

const getApiUrl = () => {
  // If explicitly set via runtime env var (e.g. in SSR), use it first
  const runtimeApiUrl =
    typeof process !== 'undefined' ? process.env?.VITE_API_URL : undefined;
  if (runtimeApiUrl) {
    return runtimeApiUrl;
  }

  // If set via build-time env var, use it
  if (import.meta.env?.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }

  // If in a browser environment
  if (typeof window !== 'undefined') {
    const { protocol, hostname, origin } = window.location;

    // Local development
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      // If using HTTPS locally (likely via Caddy/reverse proxy)
      // We assume the proxy handles the /api/* routing to the backend
      if (protocol === 'https:') {
        return origin;
      }
      // If using HTTP locally (likely direct dev server)
      // We assume backend is on standard 8080
      return 'http://localhost:8080';
    }

    // Production/Deployed: use the same origin
    return origin;
  }

  // Fallback for non-browser environments
  return 'http://localhost:8080';
};

const API_URL = getApiUrl();
export const API_BASE_URL = `${API_URL}${API_BASE_PATH}`.replace(
  /([^:]\/)\/+/g,
  '$1',
); // Remove double slashes except after protocol

console.log('[API] Initialized with base URL:', API_BASE_URL);

const endpoints = {
  '/rooms': {
    post: {
      request: createRoomRequestSchema,
      response: roomSchema,
    },
  },
  '/rooms/{id}': {
    get: {
      response: roomSchema,
    },
    post: {
      request: roomActionRequestSchema,
      response: playbackStateSchema,
    },
  },
  '/rooms/{id}/settings': {
    patch: {
      request: roomUpdateSchema,
      response: roomSchema,
    },
  },
  '/rooms/{id}/skips': {
    post: {
      response: playbackStateSchema,
    },
  },

  '/rooms/{id}/states': {
    get: {
      response: playbackStateSchema,
    },
    put: {
      request: roomActionRequestSchema,
      response: playbackStateSchema,
    },
  },
  '/rooms/{id}/sessions': {
    post: {
      request: createSessionRequestSchema,
      response: sessionResponseSchema,
    },
  },
  '/rooms/{id}/songs': {
    get: {
      response: songsListSchema,
    },
    post: {
      request: addSongRequestSchema,
      response: songSchema,
    },
  },
  '/rooms/{id}/songs/{songId}': {
    delete: {
      response: emptyObjectSchema,
    },
    post: {
      response: emptyObjectSchema,
    },
  },

  '/youtube/search': {
    get: {
      $search: youTubeSearchQuerySchema,
      response: youTubeSearchResponseSchema,
    },
  },
  '/youtube/videos/{id}': {
    get: {
      response: youTubeVideoSchema,
    },
  },
  '/rooms/{id}/events': {
    sse: {
      events: {
        connected: connectedSchema,
        playback_update: playbackStateSchema,
        songs_update: songsListSchema,
        song_added: songSchema,
        settings_update: roomSchema,
        users_update: usersUpdateSchema,
      },
    },
  },

  '/tokens/{provider}': {
    get: {
      response: providerTokenSchema,
    },
  },
  '/authorizations/spotify/token': {
    get: {
      response: spotifyTokenSchema,
    },
  },
  '/authorizations/spotify': {
    get: {
      response: messageResponseSchema,
    },
  },
  '/authorizations/youtube': {
    get: {
      response: messageResponseSchema,
    },
  },
  '/authorizations/soundcloud': {
    get: {
      response: messageResponseSchema,
    },
  },
  '/providers': {
    get: {
      response: providersSchema,
    },
  },
  '/spotify/search': {
    get: {
      $search: searchQuerySchema,
      response: searchResponseSchema,
    },
  },
  '/soundcloud/search': {
    get: {
      $search: searchQuerySchema,
      response: searchResponseSchema,
    },
  },
} satisfies RequestDefinitions;

// Helper interface for wiretyped errors
interface HTTPError extends Error {
  response?: Response;
  cause?: unknown;
}

// Type guard for HTTPError
function isHTTPError(error: unknown): error is HTTPError {
  return (
    error instanceof Error &&
    ('response' in error || error.name === 'HTTPError')
  );
}

// Helper to extract full error details from wiretyped errors
async function formatError(error: Error, context: string): Promise<Error> {
  const details: string[] = [context];

  // Walk the cause chain to get all error details
  let current: unknown = error;
  while (current instanceof Error) {
    details.push(`  → ${current.message}`);

    // Check if it's an HTTPError from wiretyped
    if (isHTTPError(current)) {
      const response = current.response;
      if (response) {
        details.push(`  → Status: ${response.status} ${response.statusText}`);

        // Use safeWrapAsync for the body reading
        const [body, _] = await safeWrapAsync(response.clone().text());
        if (body) {
          details.push(`  → Response Body: ${body}`);
        } else {
          details.push(`  → (Failed to read response body)`);
        }
      }
    }

    current = current.cause;
  }

  // If there's a non-Error cause, include it
  if (current !== undefined && current !== null) {
    details.push(`  → ${JSON.stringify(current)}`);
  }

  const fullMessage = details.join('\n');
  console.error('[API Error]', fullMessage);

  // Create a new error with the full details but keep original cause chain
  const formatted = new Error(fullMessage);
  (formatted as HTTPError).cause = error; // Type assertion here is acceptable as we are extending Error
  return formatted;
}

const createWrappedClient = (customHeaders: Record<string, string> = {}) => {
  const localBaseClient = new RequestClient({
    hostname: API_BASE_URL,
    baseUrl: API_BASE_URL,
    endpoints,
    validation: true,
    fetchOpts: {
      credentials: 'include',
      headers: customHeaders,
    },
  });

  const wrapped = {
    async get(...args: Parameters<typeof localBaseClient.get>) {
      console.log('[API] GET', args[0], args[1] ? JSON.stringify(args[1]) : '');
      const [err, data] = await localBaseClient.get(...args);
      if (err) {
        return [await formatError(err, `GET ${args[0]} failed`), null] as const;
      }
      console.log('[API] GET', args[0], '→ success');
      return [null, data] as const;
    },

    async post(...args: Parameters<typeof localBaseClient.post>) {
      console.log(
        '[API] POST',
        args[0],
        args[1] ? JSON.stringify(args[1]) : '',
        args[2] ? JSON.stringify(args[2]) : '',
      );
      const [err, data] = await localBaseClient.post(...args);
      if (err) {
        return [
          await formatError(err, `POST ${args[0]} failed`),
          null,
        ] as const;
      }
      console.log('[API] POST', args[0], '→ success');
      return [null, data] as const;
    },

    async patch(...args: Parameters<typeof localBaseClient.patch>) {
      console.log(
        '[API] PATCH',
        args[0],
        args[1] ? JSON.stringify(args[1]) : '',
        args[2] ? JSON.stringify(args[2]) : '',
      );
      const [err, data] = await localBaseClient.patch(...args);
      if (err) {
        return [
          await formatError(err, `PATCH ${args[0]} failed`),
          null,
        ] as const;
      }
      console.log('[API] PATCH', args[0], '→ success');
      return [null, data] as const;
    },

    async put(...args: Parameters<typeof localBaseClient.put>) {
      console.log(
        '[API] PUT',
        args[0],
        args[1] ? JSON.stringify(args[1]) : '',
        args[2] ? JSON.stringify(args[2]) : '',
      );
      const [err, data] = await localBaseClient.put(...args);
      if (err) {
        return [await formatError(err, `PUT ${args[0]} failed`), null] as const;
      }
      console.log('[API] PUT', args[0], '→ success');
      return [null, data] as const;
    },

    async delete(...args: Parameters<typeof localBaseClient.delete>) {
      console.log(
        '[API] DELETE',
        args[0],
        args[1] ? JSON.stringify(args[1]) : '',
      );
      const [err, data] = await localBaseClient.delete(...args);
      if (err) {
        return [
          await formatError(err, `DELETE ${args[0]} failed`),
          null,
        ] as const;
      }
      console.log('[API] DELETE', args[0], '→ success');
      return [null, data] as const;
    },

    // Expose SSE and other methods from local base client
    sse: localBaseClient.sse.bind(localBaseClient),
    url: localBaseClient.url.bind(localBaseClient),

    // Method to create a new client with additional headers
    withHeaders(headers: Record<string, string>) {
      return createWrappedClient({ ...customHeaders, ...headers });
    },
  };

  return wrapped as any;
};

export const api = createWrappedClient();
