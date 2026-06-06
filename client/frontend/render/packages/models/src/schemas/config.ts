import * as yup from 'yup';

export const providersSchema = yup.array(yup.string().required()).required();
export type Providers = yup.InferType<typeof providersSchema>;
