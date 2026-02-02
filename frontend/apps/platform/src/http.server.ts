import { createApiClientWithBaseUrl } from '@vibez/api';

const FALLBACK_API_BASE_URL = 'http://localhost:8080';

function resolveServerApiBaseUrl(request?: Request) {
  const internalApiUrl =
    typeof process !== 'undefined'
      ? process.env?.VITE_API_URL_INTERNAL
      : undefined;
  if (internalApiUrl) {
    return internalApiUrl;
  }

  if (request) {
    try {
      const originUrl = new URL(request.url);
      if (
        (originUrl.hostname === 'localhost' ||
          originUrl.hostname === '127.0.0.1') &&
        (originUrl.port === '' || originUrl.port === '80')
      ) {
        return 'http://localhost:8080';
      }
      return originUrl.origin;
    } catch {
      // Fall through to fallback.
    }
  }

  const runtimeApiUrl =
    typeof process !== 'undefined' ? process.env?.VITE_API_URL : undefined;
  if (runtimeApiUrl) {
    return runtimeApiUrl;
  }

  return FALLBACK_API_BASE_URL;
}

export function getServerApi(request?: Request) {
  const baseUrl = resolveServerApiBaseUrl(request);
  console.log('[serverApi] resolved baseUrl', {
    baseUrl,
    requestUrl: request?.url,
    apiUrl: process?.env?.VITE_API_URL,
    apiUrlInternal: process?.env?.VITE_API_URL_INTERNAL,
  });
  return createApiClientWithBaseUrl(baseUrl);
}
