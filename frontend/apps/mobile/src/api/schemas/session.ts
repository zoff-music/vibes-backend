import * as yup from 'yup';
import { roomSchema } from './room';

export const roomUserSchema = yup.object({
  id: yup.string().required(),
  nickname: yup.string().optional(),
  isAdmin: yup.boolean().required(),
  joinedAt: yup.string().required(),
  lastSeenAt: yup.string().required(),
});

export const sessionResponseSchema = yup.object({
  userId: yup.string().required(),
  nickname: yup.string().optional(),
  isAdmin: yup.boolean().required(),
  room: roomSchema.required(),
});

export const createSessionRequestSchema = yup.object({
  nickname: yup.string().optional(),
  password: yup.string().optional(),
});
