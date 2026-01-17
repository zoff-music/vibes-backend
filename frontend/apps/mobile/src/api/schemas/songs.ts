import * as yup from 'yup';

export const sourceTypeSchema = yup.string().oneOf(['youtube', 'spotify', 'soundcloud']).required();

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
});

export const addSongRequestSchema = yup.object({
  sourceType: sourceTypeSchema,
  sourceId: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().optional(),
  thumbnailUrl: yup.string().required(),
  duration: yup.number().required(),
  addedBy: yup.string().required(),
});

export const songsListSchema = yup.array(songSchema).required();

export const reorderSongsRequestSchema = yup.object({
  newPosition: yup.number().required(),
});
