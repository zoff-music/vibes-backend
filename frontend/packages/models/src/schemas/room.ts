import * as yup from 'yup';

export const roomSettingsSchema = yup.object({
  skipAllowed: yup.boolean().required(),
  democraticSkip: yup.boolean().required(),
  skipVoteThreshold: yup.number().required(),
  maxContinuousAdds: yup.number().required(),
  removeOnPlay: yup.boolean().required(),
  loopQueue: yup.boolean().required(),
  allowDuplicates: yup.boolean().required(),
});
export type RoomSettings = yup.InferType<typeof roomSettingsSchema>;

export const roomSchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  mode: yup.string()
    .transform((value) => (!value ? 'server' : value))
    .oneOf(['server', 'host'])
    .default('server'),
  hostId: yup.string().nullable().optional(),
  createdAt: yup.string().required(),
  hasPassword: yup.boolean().required(),
  settings: roomSettingsSchema.required(),
  userCount: yup.number().optional(),
});
export type Room = yup.InferType<typeof roomSchema>;

export const createRoomRequestSchema = yup.object({
  name: yup.string().required(),
  mode: yup.string().oneOf(['server', 'host']).optional(),
  password: yup.string().optional(),
});
export type CreateRoomRequest = yup.InferType<typeof createRoomRequestSchema>;

export const roomUpdateSchema = yup.object({
  name: yup.string().optional(),
  mode: yup.string().oneOf(['server', 'host']).optional(),
  settings: roomSettingsSchema.partial().optional(),
});
export type RoomUpdate = yup.InferType<typeof roomUpdateSchema>;
