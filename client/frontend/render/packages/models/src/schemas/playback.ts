import * as yup from 'yup';
import { songSchema } from './songs';

export const playbackStateSchema = yup.object({
  currentSong: songSchema.nullable().defined(),
  isPlaying: yup.boolean().required(),
  positionMs: yup.number().required(),
  updatedAt: yup.string().required(),
  serverTimeMs: yup.number().required(),
});
export type PlaybackState = yup.InferType<typeof playbackStateSchema>;

export const roomActionRequestSchema = yup.object({
  action: yup
    .string()
    .oneOf(['play', 'pause', 'seek', 'skip', 'vote'])
    .required(),
  positionMs: yup.number().optional(),
});
export type RoomActionRequest = yup.InferType<typeof roomActionRequestSchema>;
