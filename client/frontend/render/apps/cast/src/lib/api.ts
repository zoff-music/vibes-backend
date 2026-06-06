import { createApiClient } from '@vibez/api';

function getCastHeaders() {
  if (typeof window === 'undefined') return {};
  const params = new URLSearchParams(window.location.search);
  const castToken = params.get('castToken');
  const headers: Record<string, string> = {};
  if (castToken) {
    headers.Authorization = `Bearer ${castToken}`;
  }
  return headers;
}

export const api = createApiClient(getCastHeaders());
