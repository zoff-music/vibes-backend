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

export const roomSchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  createdAt: yup.string().required(),
  hasPassword: yup.boolean().required(),
  settings: roomSettingsSchema.required(),
  userCount: yup.number().optional(),
});

export const createRoomRequestSchema = yup.object({
  name: yup.string().required(),
  password: yup.string().optional(),
});

export const roomUpdateSchema = createRoomRequestSchema.pick(['name']).partial();
