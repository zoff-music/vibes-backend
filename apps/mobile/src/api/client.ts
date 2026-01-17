import { RequestClient, RequestDefinitions } from 'wiretyped';
import { API_BASE_PATH } from '@vibez/shared';
import {
  roomSchema,
  createRoomRequestSchema,
  sessionResponseSchema,
  createSessionRequestSchema,
  songsResponseSchema,
  songSchema,
  addSongRequestSchema,
  reorderSongsRequestSchema,
  roomActionSchema,
  playActionResponseSchema,
  pauseActionResponseSchema,
  seekActionResponseSchema,
  skipActionResponseSchema,
  voteActionResponseSchema,
} from './schemas';

const API_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';

// Endpoint definitions with yup schemas
const endpoints = {
  // Rooms
  '/rooms': {
    post: {
      body: createRoomRequestSchema,
      response: roomSchema,
    },
  },
  '/rooms/{id}': {
    get: {
      response: roomSchema,
    },
    patch: {
      response: roomSchema,
    },
    post: {
      // Room actions (play, pause, seek, skip, vote)
      body: roomActionSchema,
      // Response varies by action - handle at call site
    },
  },
  '/rooms/{id}/sessions': {
    post: {
      body: createSessionRequestSchema,
      response: sessionResponseSchema,
    },
  },

  // Songs
  '/rooms/{id}/songs': {
    get: {
      response: songsResponseSchema,
    },
    post: {
      body: addSongRequestSchema,
      response: songSchema,
    },
  },
  '/rooms/{id}/songs/{songId}': {
    delete: {},
  },
  '/rooms/{id}/songs/reorder': {
    patch: {
      body: reorderSongsRequestSchema,
      response: songsResponseSchema,
    },
  },
} satisfies RequestDefinitions;

export const api = new RequestClient({
  hostname: API_URL,
  baseUrl: API_BASE_PATH,
  endpoints,
  validation: true,
});

// Re-export for convenience
export { API_URL };
export type { RequestDefinitions };
