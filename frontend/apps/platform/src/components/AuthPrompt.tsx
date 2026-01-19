import { useEffect, useState } from 'react';
import { useAuthCache } from '../hooks/useAuthCache';

interface AuthPromptProps {
  activeSources?: string[];
}

const PROVIDERS = {
  spotify: {
    name: 'Spotify',
    description: 'Connect to listen to full tracks and control playback.',
    color: 'bg-green-500 hover:bg-green-600',
  },
  youtube: {
    name: 'YouTube',
    description: 'Connect to search and add videos more easily.',
    color: 'bg-red-500 hover:bg-red-600',
  },
  soundcloud: {
    name: 'SoundCloud',
    description: 'Connect for a premium audio experience.',
    color: 'bg-orange-500 hover:bg-orange-600',
  },
};

export default function AuthPrompt({ activeSources = [] }: AuthPromptProps) {
  const [dismissed, setDismissed] = useState<string[]>([]);
  const { authorizations, fetchAuthorizations } = useAuthCache();
  const userAuths = authorizations || [];

  useEffect(() => {
    fetchAuthorizations();
  }, []);

  // Determine which providers are needed but missing
  const neededProviders = activeSources.filter(
    (source) =>
      !userAuths.includes(source) &&
      !dismissed.includes(source) &&
      source in PROVIDERS,
  );

  if (neededProviders.length === 0) return null;

  const handleAuth = (provider: string) => {
    const width = 500;
    const height = 600;
    const left = window.screen.width / 2 - width / 2;
    const top = window.screen.height / 2 - height / 2;

    const popup = window.open(
      `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/api/v1/authorizations/${provider}`,
      `Auth ${provider}`,
      `width=${width},height=${height},left=${left},top=${top}`,
    );

    // Listen for success message from popup
    const handleMessage = (event: MessageEvent) => {
      if (
        event.data?.type === 'oauth-success' &&
        event.data?.provider === provider
      ) {
        // Reload page to refresh state or invalidate query
        // For now, simpler to just reload, or we could invalidate query
        window.location.reload();
        popup?.close();
        window.removeEventListener('message', handleMessage);
      }
    };

    window.addEventListener('message', handleMessage);
  };

  return (
    <div className="fixed right-4 bottom-4 z-50 flex flex-col gap-2">
      {neededProviders.map((provider) => {
        const info = PROVIDERS[provider as keyof typeof PROVIDERS];
        return (
          <div
            key={provider}
            className="slide-in-from-right fade-in max-w-sm animate-in rounded-lg border border-slate-800 bg-slate-900 p-4 shadow-xl duration-300"
          >
            <div className="mb-2 flex items-start justify-between">
              <h3 className="font-semibold text-white">Connect {info.name}</h3>
              <button
                onClick={() => setDismissed((prev) => [...prev, provider])}
                className="text-slate-400 hover:text-white"
                aria-label="Dismiss"
              >
                ✕
              </button>
            </div>
            <p className="mb-4 text-slate-400 text-sm">{info.description}</p>
            <button
              onClick={() => handleAuth(provider)}
              className={`w-full rounded px-4 py-2 font-medium text-white transition-colors ${info.color}`}
            >
              Connect {info.name}
            </button>
          </div>
        );
      })}
    </div>
  );
}
