import { api } from '@vibez/api';
import { create } from 'zustand';

interface AuthCacheState {
  providers: string[] | null;
  authorizations: string[] | null;
  
  fetchProviders: () => Promise<string[]>;
  fetchAuthorizations: () => Promise<string[]>;
}

// Promises to track in-flight requests
let providersPromise: Promise<string[]> | null = null;
let authPromise: Promise<string[]> | null = null;

// Store results in memory so different components (Modal, AuthPrompt) share them
// and don't re-fetch unnecessarily
export const useAuthCache = create<AuthCacheState>((set, get) => ({
  providers: null,
  authorizations: null,

  fetchProviders: async () => {
    const { providers } = get();
    if (providers) return providers;

    if (providersPromise) return providersPromise;

    providersPromise = (async () => {
        const [err, data] = await api.get('/providers', null);
        providersPromise = null;
        if (!err && data) {
            set({ providers: data });
            return data;
        }
        return [];
    })();

    return providersPromise;
  },

  fetchAuthorizations: async () => {
    const { authorizations } = get();
    if (authorizations) return authorizations;

    if (authPromise) return authPromise;

    authPromise = (async () => {
        const [err, data] = await api.get('/authorizations', null);
        authPromise = null;
        if (!err && data) {
            set({ authorizations: data });
            return data;
        }
        return [];
    })();

    return authPromise;
  }
}));
