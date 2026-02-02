import { createApiClientWithBaseUrl } from '@vibez/api';

const LOCAL_API_BASE_URL = 'http://localhost:8080';

export const serverApi = createApiClientWithBaseUrl(LOCAL_API_BASE_URL);
