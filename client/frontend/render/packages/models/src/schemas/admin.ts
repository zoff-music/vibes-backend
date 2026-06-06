import * as yup from 'yup';

export const adminRoomSummarySchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  userCount: yup.number().required(),
  songCount: yup.number().required(),
  activeSources: yup.array(yup.string().required()).required(),
});
export type AdminRoomSummary = yup.InferType<typeof adminRoomSummarySchema>;

export const adminRoomsSchema = yup.array(adminRoomSummarySchema).required();
export type AdminRooms = yup.InferType<typeof adminRoomsSchema>;

export const adminLoginRequestSchema = yup.object({
  password: yup.string().required(),
});
export type AdminLoginRequest = yup.InferType<typeof adminLoginRequestSchema>;

export const adminSessionResponseSchema = yup.object({
  authorized: yup.boolean().required(),
});
export type AdminSessionResponse = yup.InferType<
  typeof adminSessionResponseSchema
>;
