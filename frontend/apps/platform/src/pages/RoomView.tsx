import { getHttpError, usePlayback, useQueue, useRoom } from '@vibez/api';
import {
  type Song,
  usePlaybackStore,
  useQueueStore,
  useRoomStore,
} from '@vibez/shared';
import {
  AlertCircleIcon,
  PlayerControls,
  QueueList,
  SoundCloudPlayer,
  SpotifyPlayer,
  Toast,
  VideoPlayer,
} from '@vibez/ui';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import type { SSRInitialData } from '../App';
import { DeviceSelector } from '../components/cast/DeviceSelector';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { RoomHeader } from '../components/room/RoomHeader';
import { useCasting } from '../hooks/useCasting';
import { useProviderToken } from '../hooks/useProviderToken';
import { useThemeStore } from '../stores/themeStore';

interface RoomViewProps {
  initialData?: SSRInitialData;
}

interface ToastEventDetail {
  message: string;
  type: 'success' | 'error' | 'info';
}

export default function RoomView({ initialData }: RoomViewProps) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { currentSong, skip, isPlaying, play, pause } = usePlayback(id || '');
  const { songs, fetchQueue, voteSong } = useQueue(id || '');
  const { isConnected, castDeviceName } = useCasting(id || '');

  const headerRef = useRef<HTMLDivElement | null>(null);

  // Use songs from store, fallback to initial data for SSR
  const displaySongs = songs.length > 0 ? songs : initialData?.songs || [];

  // Use playing state from store, fallback to initial data for SSR
  const displayIsPlaying =
    isPlaying !== undefined
      ? isPlaying
      : initialData?.playback?.isPlaying || false;

  // Use current song from store, fallback to initial data for SSR
  const displayCurrentSong = useMemo(() => {
    if (currentSong) return currentSong;
    if (initialData?.playback?.currentSong)
      return initialData.playback.currentSong;
    // No fallback needed since currentSong comes from playback state
    return null;
  }, [
    currentSong,
    initialData?.playback?.currentSong,
    initialData?.songs?.length,
  ]);

  const {
    room,
    fetchRoom,
    isLoading,
    error,
    joinRoom,
    userId,
    users,
    updateRoomSettings,
    updateRoom,
  } = useRoom(id || '');
  const { toggleDarkMode } = useThemeStore();

  // Use theme from server-side initialData to avoid hydration mismatch
  const [isDarkMode, setIsDarkMode] = useState(initialData?.theme === 'dark');

  // Track if we're in SSR mode to disable animations
  const [isSSR, setIsSSR] = useState(true);

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

  // Sync with theme store when toggling
  const handleToggleDarkMode = () => {
    toggleDarkMode();
    setIsDarkMode(!isDarkMode);
  };

  // Detect client-side hydration
  useEffect(() => {
    setIsSSR(false);
  }, []);

  // Handle SSR initial data
  const { setRoom, isAdmin: storeIsAdmin } = useRoomStore();
  const { setSongs } = useQueueStore();
  const { setPlaybackState } = usePlaybackStore();
  const [initialized, setInitialized] = useState(false);

  // Use consistent room data between server and client
  const displayRoom = useMemo(() => {
    return room || initialData?.room || null;
  }, [room, initialData?.room]);

  const isAdmin = storeIsAdmin || initialData?.room?.isAdmin || false;

  useEffect(() => {
    if (initialData && !initialized) {
      console.log(
        '[SSR] Initializing room with server-provided data',
        initialData,
      );
      if (initialData.room) {
        setRoom(initialData.room);
        console.log('[SSR] Set room. isAdmin:', initialData.room.isAdmin);
      }
      if (initialData.songs) {
        console.log('[SSR] Setting songs:', initialData.songs);
        setSongs(initialData.songs);
      }
      if (initialData.playback) {
        console.log('[SSR] Setting playback state:', initialData.playback);
        setPlaybackState(initialData.playback);
      }
      setInitialized(true);
    }
  }, [initialData, initialized, setRoom, setSongs, setPlaybackState]);

  const [isAddModalVisible, setIsAddModalVisible] = useState(false);
  const [showRoomInfo, setShowRoomInfo] = useState(false);
  const [showShare, setShowShare] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showDeviceSelector, setShowDeviceSelector] = useState(false);
  const [toasts, setToasts] = useState<
    { id: string; message: string; type: 'success' | 'info' | 'error' }[]
  >([]);
  const [adminPassword, setAdminPassword] = useState('');
  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const shareUrl = typeof window === 'undefined' ? '' : window.location.href;
  const settingsButtonRef = useRef<HTMLButtonElement | null>(null);
  const settingsMenuRef = useRef<HTMLDivElement | null>(null);

  const fetchAttemptedRef = useRef<string | null>(null);
  useEffect(() => {
    if (id && fetchAttemptedRef.current !== id) {
      fetchAttemptedRef.current = id;

      // Only fetch if we don't have initial data for this specific room
      if (!initialData || initialData.room?.id !== id) {
        fetchRoom();
        fetchQueue();
      }
    }
  }, [id, fetchRoom, fetchQueue, initialData]);

  const formatTime = (ms: number) => {
    const seconds = Math.floor(ms / 1000);
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  const { actualPositionMs } = usePlayback(id || '');
  const progress = displayCurrentSong
    ? actualPositionMs / (displayCurrentSong.duration * 1000)
    : 0;

  useEffect(() => {
    const handleSongAdded = (event: Event) => {
      const customEvent = event as CustomEvent<Song>;
      console.log('[UI] song-added event received:', customEvent.detail);
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

  const joinAttemptedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!id || userId) return;

    if (joinAttemptedRef.current !== id) {
      joinAttemptedRef.current = id;
      const checkJoin = async () => {
        // Implicit session creation happens via middleware on any request.
        // We just need to fetch the room to be sure.
        await fetchRoom();
      };

      checkJoin();
    }
  }, [id, userId, fetchRoom]);

  // Render logic moved inside main return

  /* Player Controls Handlers */
  const handleAddSong = useCallback(() => setIsAddModalVisible(true), []);

  /* Queue Handlers */
  const handleVote = useCallback(
    async (songId: string) => {
      const result = await voteSong(songId);
      if (result === 'success') {
        setToasts((prev) => [
          ...prev,
          {
            id: Math.random().toString(36).substr(2, 9),
            message: 'Vote recorded!',
            type: 'success',
          },
        ]);
      } else if (result === 'already_voted') {
        setToasts((prev) => [
          ...prev,
          {
            id: Math.random().toString(36).substr(2, 9),
            message: 'You have already voted for this song',
            type: 'info',
          },
        ]);
      }
    },
    [voteSong],
  );

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

  /* Spotify Proactive Auth Logic */
  const { token: spotifyToken, fetchToken: fetchSpotifyToken } =
    useProviderToken();

  const hasSpotifySongs = useMemo(
    () => displaySongs.some((s) => s.sourceType === 'spotify'),
    [displaySongs],
  );

  useEffect(() => {
    if (hasSpotifySongs) {
      // Check auth status if we have Spotify songs
      fetchSpotifyToken('spotify');
    }
  }, [hasSpotifySongs, fetchSpotifyToken]);

  // Auto-redirect to create room if room not found
  useEffect(() => {
    if (error && id) {
      console.log('[RoomView] Error detected:', error);
      console.log('[RoomView] Error message:', error.message);
      console.log('[RoomView] Room ID:', id);

      // Check if it's an HTTP error using wiretyped's getHttpError
      const httpError = getHttpError(error);
      console.log('[RoomView] HTTP error:', httpError);

      let isRoomNotFound = false;

      if (httpError?.response) {
        console.log('[RoomView] HTTP status:', httpError.response.status);
        isRoomNotFound = httpError.response.status === 404;
      } else {
        // Fallback to message checking if getHttpError doesn't work
        isRoomNotFound =
          error.message.includes('not found') ||
          error.message.includes('404') ||
          error.message.includes('Room does not exist') ||
          error.message.toLowerCase().includes('room not found');
      }

      console.log('[RoomView] Is room not found?', isRoomNotFound);

      if (isRoomNotFound) {
        console.log(
          '[RoomView] Room not found, redirecting to create with name:',
          id,
        );
        const timer = setTimeout(() => {
          const createUrl = `/rooms/create?name=${encodeURIComponent(id)}`;
          console.log('[RoomView] Navigating to:', createUrl);
          navigate(createUrl);
        }, 2000); // Wait 2 seconds to show the error message

        return () => clearTimeout(timer);
      }
    }
  }, [error, id, navigate]);

  const handleConnectSpotify = useCallback(() => {
    const width = 600;
    const height = 800;
    const left = window.screen.width / 2 - width / 2;
    const top = window.screen.height / 2 - height / 2;

    const popup = window.open(
      '/api/v1/authorizations/spotify',
      'SpotifyAuth',
      `width=${width},height=${height},left=${left},top=${top}`,
    );

    const handleMessage = (event: MessageEvent) => {
      if (
        event.data?.type === 'oauth-success' &&
        event.data?.provider === 'spotify'
      ) {
        fetchSpotifyToken('spotify', true);
        popup?.close();
        window.removeEventListener('message', handleMessage);
      }
    };

    window.addEventListener('message', handleMessage);

    const timer = setInterval(() => {
      if (popup?.closed) {
        window.removeEventListener('message', handleMessage);
        clearInterval(timer);
        fetchSpotifyToken('spotify', true);
      }
    }, 1000);
  }, [fetchSpotifyToken]);

  return (
    <div
      className={`relative min-h-screen overflow-hidden bg-theme text-theme ${
        !isSSR ? 'animate-fade-in' : ''
      }`}
    >
      <div className="synth-sky pointer-events-none fixed inset-0" />
      <div className="synth-haze pointer-events-none fixed inset-0" />
      <div className="vhs-scanlines pointer-events-none fixed inset-0" />
      <div className="retro-grid pointer-events-none fixed bottom-0 left-1/2 h-[45vh] w-[140%]" />

      <div className="relative z-10 flex min-h-screen flex-col">
        {/* Header */}
        <RoomHeader
          headerRef={headerRef}
          displayRoom={displayRoom}
          roomId={id || ''}
          users={users}
          showRoomInfo={showRoomInfo}
          onToggleRoomInfo={() => setShowRoomInfo(!showRoomInfo)}
          isSSR={isSSR}
          showShare={showShare}
          onToggleShare={() => setShowShare(!showShare)}
          shareUrl={shareUrl}
          onCopyShareLink={handleCopyShareLink}
          isDarkMode={isDarkMode}
          onToggleDarkMode={handleToggleDarkMode}
          showSettings={showSettings}
          onToggleSettings={() => setShowSettings(!showSettings)}
          onCloseSettings={() => setShowSettings(false)}
          settingsButtonRef={settingsButtonRef}
          settingsMenuRef={settingsMenuRef}
          room={room}
          isAdmin={isAdmin}
          updateRoomSettings={updateRoomSettings}
          updateRoom={updateRoom}
          adminPassword={adminPassword}
          onAdminPasswordChange={setAdminPassword}
          onJoinAdmin={handleJoinAdmin}
          isAuthenticating={isAuthenticating}
        />

        {/* Main content */}
        {/* Main content - Conditionally rendered */}
        {isLoading && !room && !initialData?.room ? (
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
          <div className="flex flex-1 flex-col items-center justify-center px-4">
            <div className="panel-surface w-full max-w-md animate-scale-in rounded-[28px] p-8 text-center">
              <div className="mb-5 inline-flex h-20 w-20 items-center justify-center rounded-2xl border border-error/50 bg-error/10">
                <AlertCircleIcon className="h-10 w-10 text-error" />
              </div>
              <h2 className="mb-2 font-display text-lg text-theme">
                Connection Failed
              </h2>
              <p className="mb-6 text-sm text-theme-muted">{error.message}</p>
              <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
                <button
                  onClick={() => fetchRoom()}
                  className="cursor-pointer rounded-xl border border-theme bg-theme-surface px-6 py-3 text-theme text-xs transition-all hover:border-theme-strong"
                >
                  Try Again
                </button>
                {(() => {
                  const httpError = getHttpError(error);
                  const isRoomNotFound =
                    httpError?.response?.status === 404 ||
                    error.message.includes('not found') ||
                    error.message.includes('404') ||
                    error.message.includes('Room does not exist') ||
                    error.message.toLowerCase().includes('room not found');

                  return (
                    isRoomNotFound && (
                      <button
                        onClick={() =>
                          navigate(
                            `/rooms/create?name=${encodeURIComponent(id || '')}`,
                          )
                        }
                        className="cursor-pointer rounded-xl border border-primary/60 bg-primary/80 px-6 py-3 text-white text-xs shadow-[0_0_18px_rgba(255,46,151,0.4)] transition-all hover:bg-primary"
                      >
                        Create Room
                      </button>
                    )
                  );
                })()}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto lg:overflow-hidden">
            <div className="mx-auto max-w-7xl items-start gap-8 px-4 py-8 lg:grid lg:h-[calc(100vh-var(--room-header-height))] lg:grid-cols-[1.3fr_0.7fr] lg:py-6">
              {/* Player Section */}
              <div className="space-y-6 lg:flex lg:h-full lg:flex-col">
                {/* Player - Reserve height to prevent CLS */}
                <div className="crt-frame relative flex aspect-video min-h-[280px] w-full overflow-hidden rounded-[28px] bg-black sm:min-h-[340px] lg:aspect-auto lg:min-h-0 lg:flex-1">
                  <div className="vhs-scanlines pointer-events-none absolute inset-0" />
                  {isConnected && (
                    <div className="crt-overlay !absolute !z-10 pointer-events-none inset-0" />
                  )}
                  {isConnected && castDeviceName && (
                    <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center">
                      <div className="panel-surface flex items-center gap-3 rounded-full px-5 py-2 text-sm text-theme shadow-[0_0_22px_rgba(0,0,0,0.28)]">
                        <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-primary" />
                        <span className="font-medium">
                          Casting to {castDeviceName}
                        </span>
                      </div>
                    </div>
                  )}
                  {displayCurrentSong ? (
                    displayCurrentSong.sourceType === 'spotify' ? (
                      <SpotifyPlayer
                        onEnded={
                          displayRoom?.mode === 'host' ? skip : undefined
                        }
                        isVisible={!isConnected}
                      />
                    ) : displayCurrentSong.sourceType === 'soundcloud' ? (
                      <SoundCloudPlayer
                        onEnded={
                          displayRoom?.mode === 'host' ? skip : undefined
                        }
                        isVisible={!isConnected}
                      />
                    ) : (
                      <VideoPlayer
                        onEnded={
                          displayRoom?.mode === 'host' ? skip : undefined
                        }
                        isVisible={!isConnected}
                      />
                    )
                  ) : displaySongs.length > 0 ? (
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="text-center">
                        <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full border border-theme bg-theme-surface">
                          <div className="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                        </div>
                        <p className="text-sm text-theme-muted">
                          Loading song...
                        </p>
                      </div>
                    </div>
                  ) : (
                    <div className="absolute inset-0 flex items-center justify-center overflow-hidden">
                      <div className="text-center">
                        <div className="mb-6 inline-flex items-center rounded-full border border-theme px-4 py-2 text-[10px] text-theme-muted tracking-[0.35em]">
                          NO SIGNAL
                        </div>
                        <h3 className="mb-2 font-display text-base text-theme">
                          Add a song to light up the room
                        </h3>
                        <p className="text-theme-muted text-xs">
                          Tap "Add Song" to start the music flow.
                        </p>
                      </div>
                    </div>
                  )}
                </div>

                {/* Controls (always below video) */}
                <PlayerControls
                  isPlaying={displayIsPlaying}
                  canPlay={Boolean(
                    displayCurrentSong || displaySongs.length > 0,
                  )}
                  canSkip={Boolean(displayCurrentSong)}
                  onPlay={play}
                  onPause={pause}
                  onSkip={skip}
                  onAddSong={handleAddSong}
                  onOpenCast={() => setShowDeviceSelector(true)}
                  isCasting={isConnected}
                  castDeviceName={castDeviceName}
                  showSpotifyConnect={hasSpotifySongs && !spotifyToken}
                  onConnectSpotify={handleConnectSpotify}
                />
              </div>

              {/* Queue & Now Playing Section */}
              <div className="mt-8 space-y-8 lg:mt-0 lg:h-full lg:overflow-y-auto lg:pr-2">
                <div className="lg:pb-6">
                  {/* Now Playing (Integrated into list style) */}
                  {displayCurrentSong && (
                    <div className="mb-8">
                      <div className="mb-3 flex items-center gap-2">
                        <div
                          className={`h-2 w-2 rounded-full ${
                            displayIsPlaying
                              ? 'animate-pulse bg-secondary shadow-[0_0_10px_rgba(0,217,255,0.6)]'
                              : 'bg-white/30'
                          }`}
                        />
                        <span className="font-display text-[10px] text-theme-muted tracking-[0.3em]">
                          {displayIsPlaying ? 'Now Playing' : 'Paused'}
                        </span>
                      </div>

                      <div className="group/card panel-surface no-box relative overflow-hidden rounded-2xl p-4">
                        <div className="vhs-scanlines pointer-events-none absolute inset-0" />

                        {displayCurrentSong.thumbnailUrl && (
                          <div className="relative z-10 shrink-0">
                            <img
                              src={displayCurrentSong.thumbnailUrl}
                              alt=""
                              className="h-16 w-16 rounded-xl border border-theme object-cover shadow-xs transition-transform group-hover/card:scale-105"
                            />
                          </div>
                        )}
                        <div className="relative z-10 min-w-0 flex-1">
                          <h3 className="truncate font-display text-theme text-xs">
                            {displayCurrentSong.title}
                          </h3>
                          <p className="truncate text-theme-muted text-xs">
                            {displayCurrentSong.artist || 'Unknown Artist'} •{' '}
                            {formatTime(displayCurrentSong.duration * 1000)}
                          </p>
                        </div>
                      </div>
                      <progress
                        className="progress-bar mt-3 h-1 w-full"
                        value={isSSR ? 0 : progress}
                        max={1}
                      />

                      <div className="mt-8 mb-4 h-px bg-theme-surface" />
                    </div>
                  )}

                  {/* Up Next List */}
                  <div>
                    <h3 className="mb-4 font-display text-[10px] text-theme-muted tracking-[0.3em]">
                      Up Next (
                      {
                        displaySongs.filter(
                          (s) => s.id !== displayCurrentSong?.id,
                        ).length
                      }
                      )
                    </h3>
                    <QueueList
                      songs={displaySongs.filter(
                        (s) => s.id !== displayCurrentSong?.id,
                      )}
                      roomId={id || ''}
                      onVote={handleVote}
                    />
                  </div>
                </div>
              </div>
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
