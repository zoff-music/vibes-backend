import * as yup from 'yup';
import { songSchema } from './songs';

export const playbackStateSchema = yup.object({
  currentSongId: yup.string().nullable().defined(),
  currentSong: songSchema.nullable().optional(),
  isPlaying: yup.boolean().required(),
  positionMs: yup.number().required(),
  updatedAt: yup.string().required(),
  serverTimeMs: yup.number().required(),
});

export const roomActionRequestSchema = yup.object({
  action: yup.string().oneOf(['play', 'pause', 'seek', 'skip', 'vote']).required(),
  positionMs: yup.number().optional(),
});
