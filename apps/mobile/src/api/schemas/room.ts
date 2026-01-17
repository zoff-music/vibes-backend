import * as yup from 'yup';

export const roomSettingsSchema = yup.object({
  skipAllowed: yup.boolean().required(),
  democraticSkip: yup.boolean().required(),
  skipVoteThreshold: yup.number().min(0).max(1).required(),
  maxContinuousAdds: yup.number().min(1).max(10).required(),
  removeOnPlay: yup.boolean().required(),
  loopQueue: yup.boolean().required(),
  allowDuplicates: yup.boolean().required(),
});

export const roomSchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  createdAt: yup.string().required(),
  hasPassword: yup.boolean().required(),
  settings: roomSettingsSchema.required(),
  userCount: yup.number().optional(),
});

export const createRoomRequestSchema = yup.object({
  name: yup.string().min(1).max(100).required(),
  adminPassword: yup.string().optional(),
  settings: roomSettingsSchema.optional(),
});

export const createSessionRequestSchema = yup.object({
  nickname: yup.string().optional(),
  password: yup.string().optional(),
});

export const sessionResponseSchema = yup.object({
  userId: yup.string().required(),
  nickname: yup.string().optional().nullable(),
  isAdmin: yup.boolean().required(),
  room: roomSchema.required(),
});
