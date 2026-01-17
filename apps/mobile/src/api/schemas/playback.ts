import * as yup from 'yup';
import { songSchema } from './songs';

export const playbackStateSchema = yup.object({
  currentSongId: yup.string().nullable(),
  currentSong: songSchema.nullable(),
  isPlaying: yup.boolean().required(),
  positionMs: yup.number().required(),
  updatedAt: yup.string().required(),
  serverTimeMs: yup.number().required(),
});

export const roomActionSchema = yup.object({
  action: yup
    .string()
    .oneOf(['play', 'pause', 'seek', 'skip', 'vote'])
    .required(),
  positionMs: yup.number().when('action', {
    is: 'seek',
    then: (schema) => schema.required(),
    otherwise: (schema) => schema.optional(),
  }),
});

export const playActionResponseSchema = yup.object({
  action: yup.string().oneOf(['play']).required(),
  playback: playbackStateSchema.required(),
});

export const pauseActionResponseSchema = yup.object({
  action: yup.string().oneOf(['pause']).required(),
  playback: playbackStateSchema.required(),
});

export const seekActionResponseSchema = yup.object({
  action: yup.string().oneOf(['seek']).required(),
  playback: playbackStateSchema.required(),
});

export const skipActionResponseSchema = yup.object({
  action: yup.string().oneOf(['skip']).required(),
  skipped: yup.boolean().required(),
  nextSong: songSchema.nullable(),
  playback: playbackStateSchema.required(),
});

export const voteActionResponseSchema = yup.object({
  action: yup.string().oneOf(['vote']).required(),
  voted: yup.boolean().required(),
  currentVotes: yup.number().required(),
  requiredVotes: yup.number().required(),
  skipped: yup.boolean().required(),
  nextSong: songSchema.nullable(),
  playback: playbackStateSchema.optional(),
});
