import {
  addSongRequestSchema,
  connectedSchema,
  createRoomRequestSchema,
  createSessionRequestSchema,
  emptyObjectSchema,
  playbackStateSchema,
  reorderSongsRequestSchema,
  roomActionRequestSchema,
  roomSchema,
  roomUpdateSchema,
  sessionResponseSchema,
  songSchema,
  songsListSchema,
  youTubeSearchQuerySchema,
  youTubeSearchResponseSchema,
  youTubeVideoSchema,
} from '@vibez/models';
import { safeWrapAsync } from '@vibez/shared';
import { RequestClient, RequestDefinitions } from 'wiretyped';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const API_BASE_PATH = '/api/v1';
const URL = API_URL + API_BASE_PATH;

console.log('[API] Initialized with base URL:', URL);

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
    patch: {
      request: reorderSongsRequestSchema,
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
      },
    },
  },
} satisfies RequestDefinitions;

const baseClient = new RequestClient({
  hostname: URL,
  baseUrl: URL,
  endpoints,
  validation: true,
  fetchOpts: {
    credentials: 'include',
  },
});

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

// Create wrapped client that adds logging
const wrappedClient = {
  async get(...args: Parameters<typeof baseClient.get>) {
    console.log('[API] GET', args[0], args[1] ? JSON.stringify(args[1]) : '');
    const [err, data] = await baseClient.get(...args);
    if (err) {
      return [await formatError(err, `GET ${args[0]} failed`), null] as const;
    }
    console.log('[API] GET', args[0], '→ success');
    return [null, data] as const;
  },

  async post(...args: Parameters<typeof baseClient.post>) {
    console.log(
      '[API] POST',
      args[0],
      args[1] ? JSON.stringify(args[1]) : '',
      args[2] ? JSON.stringify(args[2]) : '',
    );
    const [err, data] = await baseClient.post(...args);
    if (err) {
      return [await formatError(err, `POST ${args[0]} failed`), null] as const;
    }
    console.log('[API] POST', args[0], '→ success');
    return [null, data] as const;
  },

  async patch(...args: Parameters<typeof baseClient.patch>) {
    console.log(
      '[API] PATCH',
      args[0],
      args[1] ? JSON.stringify(args[1]) : '',
      args[2] ? JSON.stringify(args[2]) : '',
    );
    const [err, data] = await baseClient.patch(...args);
    if (err) {
      return [await formatError(err, `PATCH ${args[0]} failed`), null] as const;
    }
    console.log('[API] PATCH', args[0], '→ success');
    return [null, data] as const;
  },

  async put(...args: Parameters<typeof baseClient.put>) {
    console.log(
      '[API] PUT',
      args[0],
      args[1] ? JSON.stringify(args[1]) : '',
      args[2] ? JSON.stringify(args[2]) : '',
    );
    const [err, data] = await baseClient.put(...args);
    if (err) {
      return [await formatError(err, `PUT ${args[0]} failed`), null] as const;
    }
    console.log('[API] PUT', args[0], '→ success');
    return [null, data] as const;
  },

  async delete(...args: Parameters<typeof baseClient.delete>) {
    console.log(
      '[API] DELETE',
      args[0],
      args[1] ? JSON.stringify(args[1]) : '',
    );
    const [err, data] = await baseClient.delete(...args);
    if (err) {
      return [
        await formatError(err, `DELETE ${args[0]} failed`),
        null,
      ] as const;
    }
    console.log('[API] DELETE', args[0], '→ success');
    return [null, data] as const;
  },

  // Expose SSE and other methods from base client
  sse: baseClient.sse.bind(baseClient),
  url: baseClient.url.bind(baseClient),
};

export const api = wrappedClient as unknown as typeof baseClient;
