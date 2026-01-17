import { RequestClient, RequestDefinitions } from 'wiretyped';
import { createRoomRequestSchema, roomSchema, roomUpdateSchema } from './schemas/room';
import { addSongRequestSchema, songSchema, songsListSchema, reorderSongsRequestSchema } from './schemas/songs';
import { playbackStateSchema, roomActionRequestSchema } from './schemas/playback';
import { sessionResponseSchema, createSessionRequestSchema } from './schemas/session';
import { emptyObjectSchema } from './schemas/common';

const API_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080';
const API_BASE_PATH = '/api/v1';
const URL = API_URL + API_BASE_PATH

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
    patch: {
      request: roomUpdateSchema,
      response: roomSchema,
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
  },
  '/rooms/{id}/songs/reorder/{songId}': {
    patch: {
      request: reorderSongsRequestSchema,
      response: emptyObjectSchema,
    },
  },
  '/rooms/{id}/action': {
    post: {
      request: roomActionRequestSchema,
      response: playbackStateSchema,
    },
  },
} satisfies RequestDefinitions;

const baseClient = new RequestClient({
  hostname: URL,
  baseUrl: URL,
  endpoints,
  validation: true,
});

// Helper to extract full error details from wiretyped errors
async function formatError(error: Error, context: string): Promise<Error> {
  const details: string[] = [context];
  
  // Walk the cause chain to get all error details
  let current: unknown = error;
  while (current instanceof Error) {
    details.push(`  → ${current.message}`);
    
    // Check if it's an HTTPError from wiretyped
    // We check the name property as the class might not be exported or easily accessible
    if (current.name === 'HTTPError' || (current as any).response) {
      const response = (current as any).response as Response;
      if (response) {
        details.push(`  → Status: ${response.status} ${response.statusText}`);
        try {
          // Clone response if we need to read it multiple times
          const body = await response.clone().text();
          if (body) {
            details.push(`  → Response Body: ${body}`);
          }
        } catch (e) {
          details.push(`  → (Failed to read response body)`);
        }
      }
    }
    
    current = (current as any).cause;
  }
  
  // If there's a non-Error cause, include it
  if (current !== undefined && current !== null) {
    details.push(`  → ${JSON.stringify(current)}`);
  }
  
  const fullMessage = details.join('\n');
  console.error('[API Error]', fullMessage);
  
  // Create a new error with the full details but keep original cause chain
  const formatted = new Error(fullMessage);
  (formatted as any).cause = error;
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
    console.log('[API] POST', args[0], args[1] ? JSON.stringify(args[1]) : '', args[2] ? JSON.stringify(args[2]) : '');
    const [err, data] = await baseClient.post(...args);
    if (err) {
      return [await formatError(err, `POST ${args[0]} failed`), null] as const;
    }
    console.log('[API] POST', args[0], '→ success');
    return [null, data] as const;
  },

  async patch(...args: Parameters<typeof baseClient.patch>) {
    console.log('[API] PATCH', args[0], args[1] ? JSON.stringify(args[1]) : '', args[2] ? JSON.stringify(args[2]) : '');
    const [err, data] = await baseClient.patch(...args);
    if (err) {
      return [await formatError(err, `PATCH ${args[0]} failed`), null] as const;
    }
    console.log('[API] PATCH', args[0], '→ success');
    return [null, data] as const;
  },

  async delete(...args: Parameters<typeof baseClient.delete>) {
    console.log('[API] DELETE', args[0], args[1] ? JSON.stringify(args[1]) : '');
    const [err, data] = await baseClient.delete(...args);
    if (err) {
      return [await formatError(err, `DELETE ${args[0]} failed`), null] as const;
    }
    console.log('[API] DELETE', args[0], '→ success');
    return [null, data] as const;
  },

  // Expose SSE and other methods from base client
  sse: baseClient.sse.bind(baseClient),
  url: baseClient.url.bind(baseClient),
};

export const api = wrappedClient as unknown as typeof baseClient;

