import { api } from '@vibez/api';
import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';

const DEFAULT_SETTINGS = {
  skipAllowed: true,
  democraticSkip: true,
  loopQueue: false,
  removeOnPlay: true,
};

interface CreateRoomProps {
  initialData?: any;
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
      settings,
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
    <div className="min-h-screen overflow-y-auto bg-theme px-4 pt-8 pb-12">
      <div className="mx-auto max-w-xl animate-fade-in">
        {/* Header */}
        <div className="mb-8">
          <Link
            to="/"
            className="group mb-6 inline-flex cursor-pointer items-center gap-2 text-theme-muted transition-colors hover:text-theme"
          >
            <svg
              className="h-5 w-5 transition-transform group-hover:-translate-x-1"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 19l-7-7 7-7"
              />
            </svg>
            <span className="font-bold text-sm tracking-wide">Back</span>
          </Link>
          <h1
            className="mb-2 font-black text-4xl text-theme"
            style={{ fontFamily: 'Poppins' }}
          >
            Create Session
          </h1>
          <p className="mt-2 font-medium text-theme-muted">
            Set up your shared music room
          </p>
          <p className="jp-art mt-1 text-sm text-theme-subtle">
            セッションを作成
          </p>
        </div>

        {/* Form */}
        <div className="space-y-5">
          {/* Room name */}
          <div className="glass rounded-2xl border-2 border-theme p-6">
            <label className="mb-3 block font-bold text-sm text-theme-muted tracking-wide">
              Session Name
            </label>
            <input
              type="text"
              placeholder="Friday Night Vibes"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              className="w-full rounded-xl border-2 border-theme bg-theme-surface px-4 py-4 text-base text-theme transition-all placeholder:text-theme-subtle focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden"
              autoFocus
            />
          </div>

          {/* Room Mode */}
          <div className="glass rounded-2xl border-2 border-theme p-6">
            <label className="mb-3 block font-bold text-sm text-theme-muted tracking-wide">
              Room Mode
            </label>
            <div className="grid grid-cols-2 gap-4">
              <button
                type="button"
                onClick={() => setMode('server')}
                className={`cursor-pointer rounded-xl border-2 p-4 text-left transition-all ${
                  mode === 'server'
                    ? 'border-primary bg-primary/10 text-theme'
                    : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                }`}
              >
                <div className="mb-1 font-bold">Server Mode</div>
                <div className="text-xs opacity-70">
                  Auto-play music 24/7. Perfect for radio stations.
                </div>
              </button>
              <button
                type="button"
                onClick={() => setMode('host')}
                className={`cursor-pointer rounded-xl border-2 p-4 text-left transition-all ${
                  mode === 'host'
                    ? 'border-secondary bg-secondary/10 text-theme'
                    : 'border-theme bg-theme-surface text-theme-muted hover:border-theme-strong'
                }`}
              >
                <div className="mb-1 font-bold">Host Mode</div>
                <div className="text-xs opacity-70">
                  Host controls playback. Great for parties.
                </div>
              </button>
            </div>
          </div>

          {/* Optional password */}
          <div className="glass rounded-2xl border-2 border-theme p-6">
            <label className="mb-3 block font-bold text-sm text-theme-muted tracking-wide">
              Admin Password{' '}
              <span className="font-normal text-theme-subtle">(optional)</span>
            </label>
            <input
              type="password"
              placeholder="For room control"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-xl border-2 border-theme bg-theme-surface px-4 py-4 text-base text-theme transition-all placeholder:text-theme-subtle focus:border-secondary focus:shadow-[0_0_0_3px_rgba(0,217,255,0.1)] focus:outline-hidden"
            />
            <p className="mt-3 font-medium text-theme-subtle text-xs">
              Leave empty to allow anyone to control playback
            </p>
          </div>

          {/* Settings */}
          <div className="mt-8 space-y-3">
            <h2 className="mb-4 font-bold text-sm text-theme-muted uppercase tracking-widest">
              Playback Settings
            </h2>

            <div className="glass group flex items-center justify-between rounded-2xl border-2 border-theme p-5 transition-all hover:shadow-retro">
              <div className="mr-4 flex-1">
                <div className="font-bold text-base text-theme">Allow Skip</div>
                <div className="mt-0.5 font-medium text-sm text-theme-muted">
                  Anyone can skip songs
                </div>
              </div>
              <label className="relative inline-flex cursor-pointer items-center">
                <input
                  type="checkbox"
                  checked={settings.skipAllowed}
                  onChange={(e) =>
                    updateSetting('skipAllowed', e.target.checked)
                  }
                  className="peer sr-only"
                />
                <div className="peer h-7 w-12 rounded-full bg-theme-surface shadow-retro after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full after:bg-theme-muted after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30"></div>
              </label>
            </div>

            <div className="glass group flex items-center justify-between rounded-2xl border-2 border-theme p-5 transition-all hover:shadow-retro">
              <div className="mr-4 flex-1">
                <div className="font-bold text-base text-theme">
                  Democratic Skip
                </div>
                <div className="mt-0.5 font-medium text-sm text-theme-muted">
                  Require votes to skip
                </div>
              </div>
              <label className="relative inline-flex cursor-pointer items-center">
                <input
                  type="checkbox"
                  checked={settings.democraticSkip}
                  onChange={(e) =>
                    updateSetting('democraticSkip', e.target.checked)
                  }
                  className="peer sr-only"
                />
                <div className="peer h-7 w-12 rounded-full bg-theme-surface shadow-retro after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full after:bg-theme-muted after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30"></div>
              </label>
            </div>

            <div className="glass group flex items-center justify-between rounded-2xl border-2 border-theme p-5 transition-all hover:shadow-retro">
              <div className="mr-4 flex-1">
                <div className="font-bold text-base text-theme">Loop Queue</div>
                <div className="mt-0.5 font-medium text-sm text-theme-muted">
                  Restart when queue ends
                </div>
              </div>
              <label className="relative inline-flex cursor-pointer items-center">
                <input
                  type="checkbox"
                  checked={settings.loopQueue}
                  onChange={(e) => updateSetting('loopQueue', e.target.checked)}
                  className="peer sr-only"
                />
                <div className="peer h-7 w-12 rounded-full bg-theme-surface shadow-retro after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full after:bg-theme-muted after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30"></div>
              </label>
            </div>

            <div className="glass group flex items-center justify-between rounded-2xl border-2 border-theme p-5 transition-all hover:shadow-retro">
              <div className="mr-4 flex-1">
                <div className="font-bold text-base text-theme">
                  Remove Played
                </div>
                <div className="mt-0.5 font-medium text-sm text-theme-muted">
                  Removed after play
                </div>
              </div>
              <label className="relative inline-flex cursor-pointer items-center">
                <input
                  type="checkbox"
                  checked={settings.removeOnPlay}
                  onChange={(e) =>
                    updateSetting('removeOnPlay', e.target.checked)
                  }
                  className="peer sr-only"
                />
                <div className="peer h-7 w-12 rounded-full bg-theme-surface shadow-retro after:absolute after:top-[2px] after:left-[2px] after:h-6 after:w-6 after:rounded-full after:bg-theme-muted after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:bg-white peer-focus:outline-hidden peer-focus:ring-2 peer-focus:ring-primary/30"></div>
              </label>
            </div>
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="glass-elevated mt-6 animate-scale-in rounded-2xl border-2 border-error/30 p-5">
            <div className="flex items-start gap-3">
              <svg
                className="mt-0.5 h-5 w-5 shrink-0 text-error"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <p className="flex-1 font-medium text-error text-sm">{error}</p>
            </div>
          </div>
        )}

        {/* Create button */}
        <button
          onClick={handleCreate}
          disabled={!name.trim() || isLoading}
          className="mt-8 flex w-full cursor-pointer items-center justify-center gap-2 rounded-2xl bg-primary py-4 font-black text-base text-white tracking-wide transition-all hover:bg-primary-muted hover:shadow-retro-pink active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-theme-surface disabled:text-theme-subtle"
          style={{ fontFamily: 'Poppins' }}
        >
          {isLoading ? (
            <>
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              <span>Creating...</span>
            </>
          ) : (
            <>
              <span>Create Session</span>
              <svg
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M13 7l5 5m0 0l-5 5m5-5H6"
                />
              </svg>
            </>
          )}
        </button>
      </div>
    </div>
  );
};

export default CreateRoom;
