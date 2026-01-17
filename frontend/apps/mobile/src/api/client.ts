import { RequestClient, RequestDefinitions } from 'wiretyped';
import { createRoomRequestSchema, roomSchema, roomUpdateSchema } from './schemas/room';
import { addSongRequestSchema, songSchema, songsListSchema, reorderSongsRequestSchema } from './schemas/songs';
import { playbackStateSchema, roomActionRequestSchema } from './schemas/playback';
import { sessionResponseSchema, createSessionRequestSchema } from './schemas/session';
import { emptyObjectSchema } from './schemas/common';

const API_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080';
const API_BASE_PATH = '/api/v1';

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

export const api = new RequestClient({
  hostname: API_URL,
  baseUrl: API_BASE_PATH,
  endpoints,
  validation: true,
});
