import * as yup from 'yup';

export const emptyObjectSchema = yup.object({}).nullable().optional();

export const connectedSchema = yup.object({
  time: yup.number().required(),
});
