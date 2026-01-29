import { createApiClient } from '@vibez/api';

function getCastHeaders() {
  if (typeof window === 'undefined') return {};
  const params = new URLSearchParams(window.location.search);
  const casterId = params.get('casterId') || params.get('casterUserId');
  const headers: Record<string, string> = {};

  if (params.has('roomId')) {
    // roomId is a strong indicator we are in a receiver environment
    headers['X-Cast-Receiver'] = '1';
  }

  if (casterId) {
    headers['X-Cast-Caster-Id'] = casterId;
  }

  return headers;
}

export const api = createApiClient(getCastHeaders());
