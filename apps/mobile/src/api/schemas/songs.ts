import * as yup from 'yup';

export const songSchema = yup.object({
  id: yup.string().required(),
  sourceType: yup
    .string()
    .oneOf(['youtube', 'spotify', 'soundcloud'])
    .required(),
  sourceId: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().optional().nullable(),
  thumbnailUrl: yup.string().required(),
  duration: yup.number().required(),
  addedBy: yup.string().required(),
  addedByNickname: yup.string().optional().nullable(),
  addedAt: yup.string().required(),
  position: yup.number().required(),
});

export const songsResponseSchema = yup.object({
  songs: yup.array().of(songSchema).required(),
  totalCount: yup.number().required(),
});

export const addSongRequestSchema = yup.object({
  sourceType: yup.string().required(),
  sourceId: yup.string().required(),
});

export const reorderSongsRequestSchema = yup.object({
  songId: yup.string().required(),
  newPosition: yup.number().required(),
});
