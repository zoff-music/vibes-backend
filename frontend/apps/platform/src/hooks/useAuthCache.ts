import { api } from '@vibez/api';
import { create } from 'zustand';

interface AuthCacheState {
  providers: string[] | null;
  fetchProviders: () => Promise<string[]>;
}

// Promises to track in-flight requests
let providersPromise: Promise<string[]> | null = null;


// Store results in memory so different components (Modal, AuthPrompt) share them
// and don't re-fetch unnecessarily
export const useAuthCache = create<AuthCacheState>((set, get) => ({
  providers: null,

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
}));
