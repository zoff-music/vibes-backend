import * as yup from 'yup';

export const sourceTypeSchema = yup
  .string()
  .oneOf(['youtube', 'spotify', 'soundcloud'])
  .required();
export type SourceType = yup.InferType<typeof sourceTypeSchema>;

export const songSchema = yup.object({
  id: yup.string().required(),
  sourceType: sourceTypeSchema,
  sourceId: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().optional(),
  thumbnailUrl: yup.string().required(),
  duration: yup.number().required(),
  addedBy: yup.string().required(),
  addedByNickname: yup.string().optional(),
  addedAt: yup.string().required(),
  position: yup.number().required(),
  voteCount: yup.number().optional(),
});
export type Song = yup.InferType<typeof songSchema>;

export const addSongRequestSchema = yup.object({
  sourceType: sourceTypeSchema,
  sourceId: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().optional(),
  thumbnailUrl: yup.string().required(),
  duration: yup.number().required(),
});
export type AddSongRequest = yup.InferType<typeof addSongRequestSchema>;

export const songsListSchema = yup.array(songSchema).required();
export type SongsList = yup.InferType<typeof songsListSchema>;

export const reorderSongsRequestSchema = yup.object({
  newPosition: yup.number().required(),
});
export type ReorderSongsRequest = yup.InferType<
  typeof reorderSongsRequestSchema
>;
