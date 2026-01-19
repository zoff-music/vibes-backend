import * as yup from 'yup';

export const searchResultSchema = yup.object({
  id: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().required(),
  duration: yup.string().optional(),
  thumbnail: yup.string().optional(),
  url: yup.string().optional(),
});
export type SearchResult = yup.InferType<typeof searchResultSchema>;

export const searchResponseSchema = yup.array(searchResultSchema).required();
export type SearchResponse = yup.InferType<typeof searchResponseSchema>;

export const searchQuerySchema = yup.object({
  q: yup.string().required(),
});
export type SearchQuery = yup.InferType<typeof searchQuerySchema>;
