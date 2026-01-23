import { api } from '@vibez/api';
import { type ProviderToken } from '@vibez/models';
import { useCallback, useEffect, useRef, useState } from 'react';
import { safeWrapAsync } from '../utils/wrap';

const tokenCache: Record<string, { token: string; expiresAt: string }> = {};
const pendingRequests: Record<string, Promise<ProviderToken> | undefined> = {};
const listeners = new Set<(provider: string, token: string | null) => void>();

const emitChange = (provider: string, token: string | null) => {
  listeners.forEach((listener) => {
    listener(provider, token);
  });
};

export const setCachedToken = (
  provider: string,
  token: string,
  expiresAt: string,
) => {
  tokenCache[provider] = { token, expiresAt };
  emitChange(provider, token);
};

export const getToken = (provider: string): string | null => {
  return tokenCache[provider]?.token || null;
};

export function useProviderToken() {
  const [token, setToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const lastProvider = useRef<string | null>(null);

  useEffect(() => {
    const handleUpdate = (provider: string, newToken: string | null) => {
      // Only update if this hook instance is interested in this provider
      if (lastProvider.current === provider) {
        setToken(newToken);
        if (newToken) setError(null);
      }
    };

    listeners.add(handleUpdate);
    return () => {
      listeners.delete(handleUpdate);
    };
  }, []);

  const fetchToken = useCallback(async (provider: string, force = false) => {
    lastProvider.current = provider;

    // Check cache first
    if (!force && tokenCache[provider]) {
      const { token, expiresAt } = tokenCache[provider];
      // Simple date expiration check
      if (new Date(expiresAt) > new Date()) {
        setToken(token);
        setError(null);
        return token;
      }
    }

    // Check pending requests
    const pending = pendingRequests[provider];
    if (pending) {
      const [err, data] = await safeWrapAsync<ProviderToken>(pending);
      if (err) {
        setError('Failed to join pending request');
        return null;
      }
      // data is strictly ProviderToken here if err is null
      if (data) {
        setToken(data.accessToken);
        setError(null);
        // Emit change for other listeners
        emitChange(provider, data.accessToken);
        return data.accessToken;
      }
      return null;
    }

    // Create new request promise
    const tokenRequest = async () => {
      const [err, data] = await api.get('/tokens/{provider}', { provider });
      if (err) {
        if (err.message.includes('Status: 403')) {
          throw new Error("You don't seem to have premium");
        }
        throw new Error(err.message || 'Failed to fetch token');
      }
      if (!data) {
        throw new Error('Failed to fetch token');
      }
      return data;
    };

    const requestPromise = tokenRequest();
    pendingRequests[provider] = requestPromise;

    const [err, tokenData] = await safeWrapAsync<ProviderToken>(requestPromise);
    delete pendingRequests[provider];

    if (err) {
      setError(err.message);
      return null;
    }

    if (tokenData) {
      tokenCache[provider] = {
        token: tokenData.accessToken,
        expiresAt: tokenData.expiresAt,
      };

      setToken(tokenData.accessToken);
      setError(null);
      emitChange(provider, tokenData.accessToken);
      return tokenData.accessToken;
    }
    return null;
  }, []);

  return { token, error, fetchToken };
}
