import * as yup from 'yup';

export const authorizationsListSchema = yup.array(yup.string().required()).required();
export type AuthorizationsList = yup.InferType<typeof authorizationsListSchema>;

export const spotifyTokenSchema = yup.object({
  access_token: yup.string().required(),
  token_type: yup.string().required(),
  scope: yup.string().required(),
  expires_in: yup.number().required(),
  refresh_token: yup.string().optional(),
});
export type SpotifyToken = yup.InferType<typeof spotifyTokenSchema>;
