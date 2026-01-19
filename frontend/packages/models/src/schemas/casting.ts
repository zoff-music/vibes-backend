import * as yup from 'yup';

// Enums / Unions
export const castDeviceTypeSchema = yup.string().oneOf(['chromecast', 'airplay', 'dlna']).required();
export const castSessionStateSchema = yup.string().oneOf(['connecting', 'connected', 'syncing', 'error', 'disconnected']).required();

// Schemas
export const castDeviceSchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  type: castDeviceTypeSchema,
  capabilities: yup.array().of(yup.string().required()).defined().default([]),
  isAvailable: yup.boolean().required(),
  lastSeen: yup.date().required(),
});

export const castSessionSchema = yup.object({
  id: yup.string().required(),
  deviceId: yup.string().required(),
  deviceName: yup.string().required(),
  deviceType: castDeviceTypeSchema,
  state: castSessionStateSchema,
  startedAt: yup.date().required(),
  lastSyncAt: yup.date().optional(),
  mediaSessionId: yup.string().optional(),
});

export const mediaInfoSchema = yup.object({
  contentId: yup.string().required(),
  contentType: yup.string().required(),
  streamType: yup.string().oneOf(['BUFFERED', 'LIVE']).required(),
  metadata: yup.object({
    title: yup.string().required(),
    artist: yup.string().optional(),
    albumArtist: yup.string().optional(),
    albumName: yup.string().optional(),
    images: yup.array().of(yup.object({
      url: yup.string().required(),
      height: yup.number().optional(),
      width: yup.number().optional(),
    }).required()).optional(),
  }).required(),
  duration: yup.number().optional(),
});

export const castErrorSchema = yup.object({
  code: yup.string().required(),
  description: yup.string().required(),
  details: yup.mixed().optional(),
});
