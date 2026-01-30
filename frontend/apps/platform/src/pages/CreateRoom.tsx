import { api } from '@vibez/api';
import {
  AlertCircleIcon,
  SoundCloudIcon,
  SpotifyIcon,
  Toggle,
  YouTubeIcon,
} from '@vibez/ui';
import React, { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';
import { ArrowLeftIcon } from '../components/icons/ArrowLeftIcon';
import { ArrowRightIcon } from '../components/icons/ArrowRightIcon';

const DEFAULT_SETTINGS = {
  skipAllowed: true,
  democraticSkip: true,
  loopQueue: false,
  removeOnPlay: true,
  allowDuplicates: false,
  enabledSources: ['youtube', 'spotify', 'soundcloud'],
};

import type { SSRInitialData } from '../App';

interface CreateRoomProps {
  initialData?: Pick<SSRInitialData, 'createRoomName'>;
}

const CreateRoom: React.FC<CreateRoomProps> = ({ initialData }) => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  // Initialize name - prioritize SSR data, then URL params
  const [name, setName] = useState(() => {
    // During SSR, use the initial data if available
    if (initialData?.createRoomName) {
      return initialData.createRoomName;
    }

    // During client-side, try URL params (but only if we're not in SSR)
    if (typeof window !== 'undefined') {
      const urlParams = new URLSearchParams(window.location.search);
      const urlName = urlParams.get('name');
      if (urlName) {
        return urlName;
      }
    }

    return '';
  });

  const [mode, setMode] = useState<'server' | 'host'>('server');
  const [password, setPassword] = useState('');
  const [settings, setSettings] = useState(DEFAULT_SETTINGS);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isHydrated, setIsHydrated] = useState(false);

  // Handle hydration
  useEffect(() => {
    setIsHydrated(true);

    // Fix hydration mismatch: ensure client state matches server state
    if (initialData?.createRoomName && name !== initialData.createRoomName) {
      setName(initialData.createRoomName);
      return; // Don't check URL params if we have SSR data
    }

    // After hydration, check if we need to update from URL params (only if no SSR data)
    if (!initialData?.createRoomName) {
      const urlParams = new URLSearchParams(window.location.search);
      const urlName = urlParams.get('name');

      if (urlName && urlName !== name) {
        setName(urlName);
      }
    }
  }, []); // Run only once on mount

  // Handle client-side URL changes (for navigation)
  useEffect(() => {
    if (!isHydrated) return; // Wait for hydration

    // Only update from URL if we don't have SSR data
    if (initialData?.createRoomName) {
      return;
    }

    const urlParams = new URLSearchParams(window.location.search);
    const urlName = urlParams.get('name');

    if (urlName && urlName !== name) {
      setName(urlName);
    }
  }, [searchParams, isHydrated, initialData?.createRoomName, name]);

  const handleCreate = async () => {
    if (!name.trim() || isLoading) return;

    setIsLoading(true);
    setError(null);

    const [err, room] = await api.post('/rooms', null, {
      name: name.trim(),
      password: password || undefined,
      mode,
      settings: {
        ...settings,
        // Ensure enabledSources is passed correctly, though backend schema handles it
        enabledSources: settings.enabledSources,
      },
    });

    if (err) {
      console.error('Failed to create room:', err);
      setError(err.message || 'Failed to create room');
      setIsLoading(false);
      return;
    }

    if (room) {
      const createdAt = new Date(room.createdAt);
      const now = new Date();
      const isExisting = now.getTime() - createdAt.getTime() > 10000;

      if (isExisting) {
        alert('Welcome! That room already exists, welcome!');
      }

      navigate(`/rooms/${room.id}`, { replace: true });
    }
  };

  const updateSetting = <K extends keyof typeof settings>(
    key: K,
    value: (typeof settings)[K],
  ) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="relative min-h-screen overflow-hidden bg-theme text-theme">
      <div className="synth-sky pointer-events-none fixed inset-0" />
      <div className="synth-haze pointer-events-none fixed inset-0" />
      <div className="vhs-scanlines pointer-events-none fixed inset-0" />
      <div className="sun-hero sunset-orb pointer-events-none fixed" />
      <div className="retro-grid pointer-events-none fixed bottom-0 left-1/2 z-10 h-[45vh] w-[140%]" />

      <div className="relative z-10 mx-auto flex min-h-screen w-full max-w-6xl flex-col justify-center px-6 py-10">
        <div className="mb-8 flex items-center justify-between">
          <Link
            to="/"
            className="group inline-flex cursor-pointer items-center gap-2 text-theme-muted transition-colors hover:text-theme"
          >
            <ArrowLeftIcon className="h-5 w-5 transition-transform group-hover:-translate-x-1" />
            <span className="font-pixel text-xs tracking-[0.3em]">Back</span>
          </Link>
          <div className="text-right font-pixel text-theme-muted text-xs tracking-[0.3em]">
            CREATE A SESSION
          </div>
        </div>

        <div className="crt-frame rounded-[36px] p-6 sm:p-10">
          <div className="mb-10 text-center">
            <h1 className="font-pixel text-3xl text-theme sm:text-4xl">
              CREATE A SESSION
            </h1>
            <p className="mt-3 font-pixel text-sm text-theme-muted">
              Build a neon listening room in seconds.
            </p>
            <p className="jp-art mt-2 text-theme-subtle text-xs">
              セッションを作成
            </p>
          </div>

          <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
            <div className="space-y-6">
              <div className="panel-surface rounded-[24px] p-6">
                <label className="mb-3 block font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
                  SESSION NAME
                </label>
                <input
                  type="text"
                  placeholder="Friday Night Vibes"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                  className="w-full rounded-2xl border border-theme bg-theme-surface px-4 py-4 text-base text-theme placeholder:text-theme-subtle focus:border-secondary focus:outline-hidden focus:ring-2 focus:ring-secondary/30"
                  autoFocus
                />
              </div>

              <div className="panel-surface rounded-[24px] p-6">
                <label className="mb-4 block font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
                  ROOM MODE
                </label>
                <div className="grid gap-4 sm:grid-cols-2">
                  <button
                    type="button"
                    onClick={() => setMode('server')}
                    className={`cursor-pointer rounded-2xl border px-4 py-4 text-left transition-all ${
                      mode === 'server'
                        ? 'border-secondary/60 bg-secondary/10 text-theme shadow-[0_0_18px_rgba(0,217,255,0.35)]'
                        : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                    }`}
                  >
                    <div className="mb-2 font-pixel text-xs tracking-[0.2em]">
                      SERVER MODE
                    </div>
                    <div className="text-theme-muted text-xs">
                      Auto-play music 24/7 for radio rooms.
                    </div>
                  </button>
                  <button
                    type="button"
                    onClick={() => setMode('host')}
                    className={`cursor-pointer rounded-2xl border px-4 py-4 text-left transition-all ${
                      mode === 'host'
                        ? 'border-primary/60 bg-primary/10 text-theme shadow-[0_0_18px_rgba(255,46,151,0.35)]'
                        : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                    }`}
                  >
                    <div className="mb-2 font-pixel text-xs tracking-[0.2em]">
                      HOST MODE
                    </div>
                    <div className="text-theme-muted text-xs">
                      Host controls playback for parties.
                    </div>
                  </button>
                </div>
              </div>

              <div className="panel-surface rounded-[24px] p-6">
                <label className="mb-3 block font-pixel text-[10px] text-theme-muted tracking-[0.3em]">
                  ADMIN PASSWORD
                  <span className="ml-2 text-theme-subtle">(optional)</span>
                </label>
                <input
                  type="password"
                  placeholder="For room control"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-2xl border border-theme bg-theme-surface px-4 py-4 text-base text-theme placeholder:text-theme-subtle focus:border-primary focus:outline-hidden focus:ring-2 focus:ring-primary/30"
                />
                <p className="mt-3 text-theme-subtle text-xs">
                  Leave empty to allow anyone to control playback.
                </p>
              </div>
            </div>

            <div className="panel-surface rounded-[24px] p-6">
              <h2 className="mb-6 font-pixel text-[11px] text-theme-muted tracking-[0.4em]">
                SOURCES
              </h2>
              <div className="grid grid-cols-3 gap-3">
                {[
                  {
                    id: 'youtube',
                    Icon: YouTubeIcon,
                    color:
                      'bg-red-500/20 border-red-500/40 text-red-500 shadow-[0_0_10px_rgba(239,68,68,0.2)]',
                  },
                  {
                    id: 'spotify',
                    Icon: SpotifyIcon,
                    color:
                      'bg-green-500/20 border-green-500/40 text-green-500 shadow-[0_0_10px_rgba(34,197,94,0.2)]',
                  },
                  {
                    id: 'soundcloud',
                    Icon: SoundCloudIcon,
                    color:
                      'bg-orange-500/20 border-orange-500/40 text-orange-500 shadow-[0_0_10px_rgba(249,115,22,0.2)]',
                  },
                ].map(({ id, Icon, color }) => {
                  const isEnabled = settings.enabledSources.includes(id);
                  return (
                    <button
                      key={id}
                      onClick={() => {
                        const newSources = isEnabled
                          ? settings.enabledSources.filter((s) => s !== id)
                          : [...settings.enabledSources, id];
                        updateSetting('enabledSources', newSources);
                      }}
                      className={`group relative flex cursor-pointer items-center justify-center rounded-xl border py-4 transition-all ${
                        isEnabled
                          ? color
                          : 'border-theme bg-theme-surface text-theme-muted opacity-40 hover:border-theme-strong hover:opacity-60'
                      }`}
                      title={`${isEnabled ? 'Disable' : 'Enable'} ${id}`}
                    >
                      <Icon className="h-6 w-6" />
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="panel-surface rounded-[24px] p-6">
              <h2 className="mb-6 font-pixel text-[11px] text-theme-muted tracking-[0.4em]">
                PLAYBACK SETTINGS
              </h2>
              <div className="space-y-4">
                <Toggle
                  label="ALLOW SKIP"
                  description="Anyone can skip songs"
                  checked={settings.skipAllowed}
                  onChange={(checked) => updateSetting('skipAllowed', checked)}
                />

                <Toggle
                  label="DEMOCRATIC SKIP"
                  description="Require votes to skip"
                  checked={settings.democraticSkip}
                  onChange={(checked) =>
                    updateSetting('democraticSkip', checked)
                  }
                />

                <Toggle
                  label="LOOP QUEUE"
                  description="Restart when queue ends"
                  checked={settings.loopQueue}
                  onChange={(checked) => updateSetting('loopQueue', checked)}
                />

                <Toggle
                  label="REMOVE PLAYED"
                  description="Removed after play"
                  checked={settings.removeOnPlay}
                  onChange={(checked) => updateSetting('removeOnPlay', checked)}
                />

                <Toggle
                  label="ALLOW DUPLICATES"
                  description="Same song multiple times"
                  checked={settings.allowDuplicates}
                  onChange={(checked) =>
                    updateSetting('allowDuplicates', checked)
                  }
                />
              </div>
            </div>
          </div>

          {/* Error */}
          {error && (
            <div className="mt-6 rounded-2xl border border-error/40 bg-error/10 p-5 text-error text-sm">
              <div className="flex items-start gap-3">
                <AlertCircleIcon className="mt-0.5 h-5 w-5 shrink-0 text-error" />
                <p className="flex-1">{error}</p>
              </div>
            </div>
          )}

          {/* Create button */}
          <button
            onClick={handleCreate}
            disabled={!name.trim() || isLoading}
            className="mt-8 flex w-full cursor-pointer items-center justify-center gap-3 rounded-2xl bg-primary px-6 py-4 font-pixel text-sm text-white shadow-[0_0_24px_rgba(255,46,151,0.45)] transition-all hover:-translate-y-0.5 hover:bg-primary-muted disabled:cursor-not-allowed disabled:bg-theme-surface disabled:text-theme-subtle"
          >
            {isLoading ? (
              <>
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                <span>Creating...</span>
              </>
            ) : (
              <>
                <span>Start Session</span>
                <ArrowRightIcon className="h-5 w-5" />
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default CreateRoom;
