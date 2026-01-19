import * as yup from 'yup';

export const emptyObjectSchema = yup.object({}).nullable().optional();
export type EmptyObject = yup.InferType<typeof emptyObjectSchema>;

export const connectedSchema = yup.object({
  time: yup.number().required(),
});
export type Connected = yup.InferType<typeof connectedSchema>;
