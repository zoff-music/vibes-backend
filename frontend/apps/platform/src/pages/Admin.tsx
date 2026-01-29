import { api } from '@vibez/api';
import type { AdminRoomSummary } from '@vibez/models';
import { SoundCloudIcon, SpotifyIcon, YouTubeIcon } from '@vibez/ui';
import { JSX, useEffect, useMemo, useRef, useState } from 'react';

import type { SSRInitialData } from '../App';

interface AdminProps {
  initialData?: Pick<SSRInitialData, 'adminRooms' | 'adminAuthorized'>;
}

const sourceIcons: Record<string, JSX.Element> = {
  youtube: <YouTubeIcon className="h-4 w-4 text-red-500" />,
  spotify: <SpotifyIcon className="h-4 w-4 text-green-500" />,
  soundcloud: <SoundCloudIcon className="h-4 w-4 text-orange-500" />,
};

type AdminSSEMessage = {
  type: string;
  data: unknown;
};

export default function Admin({ initialData }: AdminProps) {
  const [rooms, setRooms] = useState<AdminRoomSummary[]>(
    initialData?.adminRooms ?? [],
  );
  const [isAuthorized, setIsAuthorized] = useState<boolean>(
    initialData?.adminAuthorized ?? false,
  );
  const [isCheckingAuth, setIsCheckingAuth] = useState<boolean>(
    !initialData?.adminAuthorized,
  );
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const unsubscribeRef = useRef<null | (() => void)>(null);

  const hasRooms = rooms.length > 0;

  const totalViewers = useMemo(
    () => rooms.reduce((acc, room) => acc + room.userCount, 0),
    [rooms],
  );

  const fetchRooms = async () => {
    const [err, data] = await api.get('/admin/rooms', null);
    if (err || !data) {
      return;
    }
    setRooms(data);
  };

  const connectSSE = async () => {
    const [err, unsubscribe] = await api.sse(
      '/admin/events',
      null,
      (result: [Error | null, AdminSSEMessage | null]) => {
        const [eventError, message] = result;
        if (eventError || !message) {
          return;
        }
        if (message.type === 'admin_rooms_update') {
          if (Array.isArray(message.data)) {
            setRooms(message.data as AdminRoomSummary[]);
          }
        }
      },
    );

    if (!err && unsubscribe) {
      unsubscribeRef.current = unsubscribe;
    }
  };

  useEffect(() => {
    let isMounted = true;

    if (!isAuthorized) {
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
      return;
    }

    const setup = async () => {
      await fetchRooms();
      if (!isMounted) return;
      await connectSSE();
    };

    setup();

    return () => {
      isMounted = false;
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
    };
  }, [isAuthorized]);

  useEffect(() => {
    let isMounted = true;

    if (initialData?.adminAuthorized) {
      setIsCheckingAuth(false);
      return;
    }

    const checkAuth = async () => {
      const [err, data] = await api.get('/admin/rooms', null);
      if (!isMounted) {
        return;
      }
      if (!err && data) {
        setRooms(data);
        setIsAuthorized(true);
      }
      setIsCheckingAuth(false);
    };

    checkAuth();

    return () => {
      isMounted = false;
    };
  }, [initialData?.adminAuthorized]);

  const handleLogin = async () => {
    if (!password.trim() || isLoading) {
      return;
    }

    setIsLoading(true);
    setErrorMessage(null);

    const [loginError, response] = await api.post('/admin/sessions', null, {
      password: password.trim(),
    });

    setIsLoading(false);

    if (loginError || !response?.authorized) {
      setErrorMessage('Invalid admin password.');
      return;
    }

    setIsAuthorized(true);
    setPassword('');
  };

  const handleLogout = async () => {
    setIsLoading(true);
    setErrorMessage(null);

    const [logoutError] = await api.delete('/admin/sessions', null);

    setIsLoading(false);

    if (logoutError) {
      setErrorMessage('Failed to sign out.');
      return;
    }

    setIsAuthorized(false);
    setRooms([]);
  };

  if (isCheckingAuth) {
    return (
      <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-6 py-12">
        <div className="glass rounded-3xl border-2 border-ink/10 p-8 text-center dark:border-gray-700">
          <div className="mb-4 h-10 w-10 animate-spin rounded-full border-2 border-ink/20 border-t-primary" />
          <p className="text-ink/60 text-sm dark:text-gray-400">
            Checking admin session...
          </p>
        </div>
      </div>
    );
  }

  if (!isAuthorized) {
    return (
      <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-6 py-12">
        <div className="relative z-10 w-full max-w-md">
          <div className="glass rounded-3xl border-2 border-ink/10 p-8 text-center dark:border-gray-700">
            <h1
              className="mb-2 font-black text-3xl text-ink tracking-tight dark:text-white"
            >
              Admin Access
            </h1>
            <p className="mb-6 text-ink/60 text-sm dark:text-gray-400">
              Enter the admin password to view live room stats.
            </p>

            <div className="space-y-4 text-left">
              <label className="block font-semibold text-ink/80 text-xs uppercase tracking-widest dark:text-gray-400">
                Admin Password
              </label>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                onKeyDown={(event) => event.key === 'Enter' && handleLogin()}
                className="w-full rounded-xl border-2 border-ink/20 bg-surface px-4 py-3 font-mono text-base text-ink transition-all placeholder:text-ink/40 focus:border-primary focus:shadow-[0_0_0_3px_rgba(255,46,151,0.1)] focus:outline-hidden dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-500"
                placeholder="Password"
                disabled={isLoading}
              />
              {errorMessage && (
                <p className="text-red-500 text-sm">{errorMessage}</p>
              )}
              <button
                onClick={handleLogin}
                disabled={!password.trim() || isLoading}
                className="w-full rounded-xl bg-primary py-3 font-bold text-white transition-all hover:bg-primary/90 hover:shadow-retro-pink active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-ink/10 disabled:text-ink/30 dark:disabled:bg-gray-700 dark:disabled:text-gray-500"
              >
                {isLoading ? 'Checking...' : 'Enter Dashboard'}
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-surface px-6 py-10 text-ink dark:bg-gray-900 dark:text-white">
      <div className="mx-auto max-w-5xl space-y-8">
        <header className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <h1
              className="font-black text-3xl tracking-tight"
            >
              Live Rooms
            </h1>
            <p className="text-ink/60 text-sm dark:text-gray-400">
              {hasRooms
                ? `${rooms.length} rooms, ${totalViewers} viewers online`
                : 'No active rooms yet'}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchRooms}
              className="rounded-xl border border-ink/15 px-4 py-2 font-semibold text-sm transition-all hover:border-primary hover:text-primary dark:border-gray-700 dark:hover:border-primary-light dark:hover:text-primary-light"
            >
              Refresh
            </button>
            <button
              onClick={handleLogout}
              className="rounded-xl bg-ink/10 px-4 py-2 font-semibold text-sm transition-all hover:bg-ink/20 dark:bg-gray-800 dark:hover:bg-gray-700"
            >
              Sign Out
            </button>
          </div>
        </header>

        <div className="grid gap-4">
          {rooms.map((room) => (
            <div
              key={room.id}
              className="glass flex flex-col gap-4 rounded-2xl border-2 border-ink/10 p-5 transition-all hover:border-primary/30 md:flex-row md:items-center md:justify-between dark:border-gray-700"
            >
              <div>
                <div className="flex items-center gap-3">
                  <h2
                    className="font-bold text-ink text-xl dark:text-white"
                  >
                    {room.name}
                  </h2>
                  <span className="rounded-full bg-ink/10 px-3 py-1 font-mono text-ink/60 text-xs uppercase tracking-wider dark:bg-gray-800 dark:text-gray-400">
                    {room.id}
                  </span>
                </div>
                <div className="mt-3 flex items-center gap-3 text-ink/60 text-sm dark:text-gray-400">
                  <span className="rounded-lg bg-ink/5 px-2 py-1 font-semibold dark:bg-gray-800">
                    {room.userCount} viewers
                  </span>
                  <span className="rounded-lg bg-ink/5 px-2 py-1 font-semibold dark:bg-gray-800">
                    {room.songCount} songs
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-3">
                {room.activeSources.length === 0 ? (
                  <span className="text-ink/50 text-xs uppercase tracking-widest dark:text-gray-500">
                    No sources
                  </span>
                ) : (
                  room.activeSources.map((source) => (
                    <span
                      key={source}
                      className="rounded-lg bg-ink/5 p-2 dark:bg-gray-800"
                      title={source}
                    >
                      {sourceIcons[source]}
                    </span>
                  ))
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
