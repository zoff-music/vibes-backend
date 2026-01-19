import * as yup from 'yup';

export const youTubeVideoSchema = yup.object({
  id: yup.string().required(),
  title: yup.string().required(),
  channelTitle: yup.string().required(),
  thumbnailUrl: yup.string().required(),
  duration: yup.string().optional(), // ISO 8601 duration
});
export type YouTubeVideo = yup.InferType<typeof youTubeVideoSchema>;

export const youTubeSearchResponseSchema = yup.array(youTubeVideoSchema);
export type YouTubeSearchResponse = yup.InferType<typeof youTubeSearchResponseSchema>;

export const youTubeSearchQuerySchema = yup.object({
  q: yup.string().required(),
});
export type YouTubeSearchQuery = yup.InferType<typeof youTubeSearchQuerySchema>;
