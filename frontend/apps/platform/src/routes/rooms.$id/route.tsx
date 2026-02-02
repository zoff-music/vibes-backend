import { getHttpError, usePlayback, useQueue, useRoom } from '@vibez/api';
import {
  type Song,
  usePlaybackStore,
  useQueueStore,
  useRoomStore,
} from '@vibez/shared';
import { Toast } from '@vibez/ui';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLoaderData, useNavigate, useParams } from 'react-router';
import { DeviceSelector } from '../../components/cast/DeviceSelector';
import { AddToQueueModal } from '../../components/queue/AddToQueueModal';
import { RoomErrorView } from '../../components/room/RoomErrorView';
import { RoomHeader } from '../../components/room/RoomHeader';
import { RoomPlayer } from '../../components/room/RoomPlayer';
import { RoomQueue } from '../../components/room/RoomQueue';
import { useThemeDisplay } from '../../hooks/useThemeDisplay';
import { useThemeStore } from '../../stores/themeStore';
import type { RoomLoaderData } from './loader';
import { loader } from './loader';

export { loader };

interface ToastEventDetail {
  message: string;
  type: 'success' | 'error' | 'info';
}

export default function Room() {
  const loaderData = useLoaderData() as RoomLoaderData;
  /* 1. Refs */
  const headerRef = useRef<HTMLDivElement | null>(null);
  const fetchAttemptedRef = useRef<string | null>(null);
  const joinAttemptedRef = useRef<string | null>(null);
  const primeAttemptedRef = useRef<string | null>(null);
  const playbackAttemptedRef = useRef<string | null>(null);
  const settingsButtonRef = useRef<HTMLButtonElement | null>(null);
  const settingsMenuRef = useRef<HTMLDivElement | null>(null);
  const originalTitleRef = useRef<string | null>(null);

  /* 2. Hooks */
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toggleDarkMode } = useThemeStore();
  const { room, fetchRoom, isLoading, error, joinRoom, userId } = useRoom(
    id || '',
  );
  const { fetchPlayback } = usePlayback(id || '');

  // Set warping state based on loading
  useEffect(() => {
    // Keep room loading state local to the page; no background warp.
  }, []);

  const { fetchQueue } = useQueue(id || '');
  const primeQueue = useCallback(async () => {
    if (!id) return;
    if (!loaderData || loaderData.room?.id !== id || !loaderData.playback) {
      await fetchPlayback();
    }
    fetchQueue();
  }, [id, loaderData, fetchPlayback, fetchQueue]);

  // Granular store setters (subscription-free/minimized re-renders)
  const setRoom = useRoomStore((state) => state.setRoom);
  const setSongs = useQueueStore((state) => state.setSongs);
  const songsFromStore = useQueueStore((state) => state.songs);
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const currentSongFromStore = usePlaybackStore((state) => state.currentSong);

  // Granular store query (only re-render if isAdmin changes)
  // storeIsAdmin removed as it was unused

  /* 3. State */
  const [initialized, setInitialized] = useState(false);
  const [isSSR, setIsSSR] = useState(true);
  const { themeId, currentTheme } = useThemeDisplay();
  const [isAddModalVisible, setIsAddModalVisible] = useState(false);
  const [showShare, setShowShare] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showDeviceSelector, setShowDeviceSelector] = useState(false);
  const [adminPassword, setAdminPassword] = useState('');
  const [isAuthenticating, setIsAuthenticating] = useState(false);

  const [toasts, setToasts] = useState<
    { id: string; message: string; type: 'success' | 'info' | 'error' }[]
  >([]);

  /* 4. Computed / Derived */
  const shareUrl = typeof window === 'undefined' ? '' : window.location.href;
  const displayRoom = useMemo(
    () => room || loaderData?.room || null,
    [room, loaderData?.room],
  );

  /* 5. Handlers (Arrow Functions) */
  const handleToggleDarkMode = useCallback(() => {
    toggleDarkMode();
  }, [toggleDarkMode]);

  const handleAddSong = useCallback(() => setIsAddModalVisible(true), []);

  const handleCopyShareLink = useCallback(() => {
    if (!shareUrl) return;
    navigator.clipboard.writeText(shareUrl);
    setToasts((prev) => [
      ...prev,
      {
        id: Math.random().toString(36).substr(2, 9),
        message: 'Link copied!',
        type: 'success',
      },
    ]);
    setShowShare(false);
  }, [shareUrl]);

  const handleJoinAdmin = useCallback(async () => {
    if (!adminPassword) return;
    setIsAuthenticating(true);
    const data = await joinRoom(adminPassword);
    setIsAuthenticating(false);

    if (data) {
      setToasts((prev) => [
        ...prev,
        {
          id: Math.random().toString(36).substr(2, 9),
          message: room?.hasPassword
            ? 'Logged in as admin!'
            : 'Password set and admin granted!',
          type: 'success',
        },
      ]);
      setAdminPassword('');
    } else {
      setToasts((prev) => [
        ...prev,
        {
          id: Math.random().toString(36).substr(2, 9),
          message: 'Failed to authenticate. Incorrect password?',
          type: 'error',
        },
      ]);
    }
  }, [adminPassword, joinRoom, room?.hasPassword]);

  const handleLeave = useCallback(() => {
    navigate('/', { state: { fromRoom: true } });
  }, [navigate]);

  /* 6. Effects */

  // Client-side detection for animations/hydration
  useEffect(() => {
    setIsSSR(false);
  }, []);

  // Header height syncing
  useEffect(() => {
    const header = headerRef.current;
    if (!header) return;

    const updateHeaderHeight = () => {
      const height = header.getBoundingClientRect().height;
      document.documentElement.style.setProperty(
        '--room-header-height',
        `${height}px`,
      );
    };

    updateHeaderHeight();

    const resizeObserver = new ResizeObserver(() => {
      updateHeaderHeight();
    });
    resizeObserver.observe(header);

    window.addEventListener('resize', updateHeaderHeight);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener('resize', updateHeaderHeight);
    };
  }, []);

  // Document title (now playing) and restore on leave
  useEffect(() => {
    if (typeof document === 'undefined') return;
    if (!originalTitleRef.current) {
      originalTitleRef.current = document.title;
    }

    const baseTitle = originalTitleRef.current;
    if (currentSongFromStore?.title) {
      const roomName = displayRoom?.name ? ` · ${displayRoom.name}` : '';
      document.title = `${currentSongFromStore.title}${roomName}`;
    } else if (baseTitle) {
      document.title = baseTitle;
    }

    return () => {
      if (originalTitleRef.current) {
        document.title = originalTitleRef.current;
      }
    };
  }, [currentSongFromStore?.title, displayRoom?.name]);

  // SSR Initialization
  useEffect(() => {
    if (loaderData && !initialized) {
      console.log(
        '[SSR] Initializing room with server-provided data',
        loaderData,
      );
      if (loaderData.room) {
        setRoom(loaderData.room);
      }
      if (loaderData.songs) {
        setSongs(loaderData.songs);
      }
      if (loaderData.playback) {
        setPlaybackState(loaderData.playback);
      }
      setInitialized(true);
    }
  }, [loaderData, initialized, setRoom, setSongs, setPlaybackState]);

  // Initial fetch and session join
  useEffect(() => {
    if (!id) return;

    if (fetchAttemptedRef.current !== id) {
      fetchAttemptedRef.current = id;
      const shouldFetchRoom = !loaderData || loaderData.room?.id !== id;
      const shouldPrimeQueue =
        shouldFetchRoom || !loaderData?.playback || !loaderData?.songs;

      if (shouldFetchRoom) {
        fetchRoom();
      }

      if (shouldPrimeQueue && primeAttemptedRef.current !== id) {
        primeAttemptedRef.current = id;
        void primeQueue();
      }
    }

    if (!userId && joinAttemptedRef.current !== id) {
      joinAttemptedRef.current = id;
      fetchRoom();
      if (primeAttemptedRef.current !== id) {
        primeAttemptedRef.current = id;
        void primeQueue();
      }
    }
  }, [id, userId, fetchRoom, loaderData, primeQueue]);

  // If queue is present but playback is missing, retry fetching playback once per queue size.
  useEffect(() => {
    if (!id) return;
    if (currentSongFromStore || songsFromStore.length === 0) return;

    const attemptKey = `${id}:${songsFromStore.length}`;
    if (playbackAttemptedRef.current === attemptKey) return;
    playbackAttemptedRef.current = attemptKey;

    void fetchPlayback();
  }, [id, currentSongFromStore, songsFromStore.length, fetchPlayback]);

  // Global events (Toast, Song Added)
  useEffect(() => {
    const handleSongAdded = (event: Event) => {
      const customEvent = event as CustomEvent<Song>;
      const song = customEvent.detail;
      setToasts((prev) => [
        ...prev,
        {
          id: Math.random().toString(36).substr(2, 9),
          message: `"${song.title}" added to queue`,
          type: 'success',
        },
      ]);
    };

    const handleShowToast = (event: Event) => {
      const customEvent = event as CustomEvent<ToastEventDetail>;
      const { message, type } = customEvent.detail;
      setToasts((prev) => [
        ...prev,
        {
          id: Math.random().toString(36).substr(2, 9),
          message,
          type,
        },
      ]);
    };

    window.addEventListener('song-added', handleSongAdded);
    window.addEventListener('show-toast', handleShowToast);

    return () => {
      window.removeEventListener('song-added', handleSongAdded);
      window.removeEventListener('show-toast', handleShowToast);
    };
  }, []);

  // Settings menu outside click
  useEffect(() => {
    if (!showSettings) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (settingsMenuRef.current?.contains(target)) return;
      if (settingsButtonRef.current?.contains(target)) return;
      setShowSettings(false);
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('touchstart', handlePointerDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('touchstart', handlePointerDown);
    };
  }, [showSettings]);

  // Navigation on error (Room not found)
  useEffect(() => {
    if (error && id) {
      const httpError = getHttpError(error);
      const isRoomNotFound =
        httpError?.response?.status === 404 ||
        error.message.includes('not found') ||
        error.message.includes('404');

      if (isRoomNotFound) {
        const timer = setTimeout(() => {
          navigate(`/rooms/create?name=${encodeURIComponent(id)}`);
        }, 2000);
        return () => clearTimeout(timer);
      }
    }
  }, [error, id, navigate]);

  return (
    <div
      className={`relative min-h-screen overflow-x-hidden lg:h-screen lg:overflow-hidden ${
        !isSSR ? 'animate-fade-in' : ''
      }`}
    >
      <div className="relative z-10 flex min-h-screen flex-col overflow-x-hidden lg:h-screen lg:overflow-hidden">
        {/* Header */}
        <RoomHeader
          headerRef={headerRef}
          displayRoom={displayRoom}
          roomId={id || ''}
          showShare={showShare}
          onToggleShare={() => setShowShare(!showShare)}
          shareUrl={shareUrl}
          onCopyShareLink={handleCopyShareLink}
          themeId={themeId}
          currentTheme={currentTheme}
          onToggleDarkMode={handleToggleDarkMode}
          showSettings={showSettings}
          onToggleSettings={() => setShowSettings(!showSettings)}
          onCloseSettings={() => setShowSettings(false)}
          settingsButtonRef={settingsButtonRef}
          settingsMenuRef={settingsMenuRef}
          adminPassword={adminPassword}
          onAdminPasswordChange={setAdminPassword}
          onJoinAdmin={handleJoinAdmin}
          isAuthenticating={isAuthenticating}
          onLeave={handleLeave}
        />

        {/* Main content */}
        {/* Main content - Conditionally rendered */}
        {isLoading && !room && !loaderData?.room ? (
          <div className="flex flex-1 items-center justify-center">
            <div className="animate-fade-in text-center">
              <div className="mb-5 inline-flex h-20 w-20 items-center justify-center rounded-2xl border border-theme bg-theme-surface shadow-[0_0_20px_rgba(255,46,151,0.25)]">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              </div>
              <p className="text-sm text-theme-muted">Loading session...</p>
              <p className="jp-art mt-1 text-theme-subtle text-xs">
                読み込み中
              </p>
            </div>
          </div>
        ) : error ? (
          <RoomErrorView
            error={error}
            roomId={id || ''}
            onRetry={() => fetchRoom()}
          />
        ) : (
          <div className="flex-1 overflow-visible lg:overflow-hidden">
            <div className="mx-auto max-w-7xl items-start gap-8 px-4 py-8 lg:grid lg:h-[calc(100vh-var(--room-header-height))] lg:grid-cols-[1.3fr_0.7fr] lg:py-6">
              {/* Player Section */}
              <RoomPlayer
                roomId={id || ''}
                displayRoom={displayRoom}
                onAddSong={handleAddSong}
                onOpenCast={() => setShowDeviceSelector(true)}
                initialPlayback={loaderData?.playback}
              />

              {/* Queue & Now Playing Section */}
              <RoomQueue
                roomId={id || ''}
                isSSR={isSSR}
                initialPlayback={loaderData?.playback}
                initialSongs={loaderData?.songs}
              />
            </div>
          </div>
        )}

        {/* Toasts */}
        {toasts.map((toast) => (
          <Toast
            key={toast.id}
            message={toast.message}
            type={
              toast.type === 'success'
                ? 'success'
                : toast.type === 'error'
                  ? 'error'
                  : 'info'
            }
            onClose={() =>
              setToasts((prev) => prev.filter((t) => t.id !== toast.id))
            }
          />
        ))}

        {/* Device Selector Modal */}
        <DeviceSelector
          isOpen={showDeviceSelector}
          onClose={() => setShowDeviceSelector(false)}
        />

        {/* Add Song Modal */}
        <AddToQueueModal
          roomId={id || ''}
          isVisible={isAddModalVisible}
          onClose={() => setIsAddModalVisible(false)}
        />
      </div>
    </div>
  );
}
