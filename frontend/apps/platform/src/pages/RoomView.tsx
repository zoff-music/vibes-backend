import type { PlaybackState, Room } from '@vibez/models';
import { type Song, usePlaybackStore } from '@vibez/shared';
import {
  PlayerControls,
  SoundCloudPlayer,
  SpotifyPlayer,
  Toast,
  VideoPlayer,
} from '@vibez/ui';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { getHttpError } from 'wiretyped';
import { DeviceSelector } from '../components/cast/DeviceSelector';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { QueueList } from '../components/queue/QueueList';
import { RoomHeader } from '../components/room/RoomHeader';
import { useCasting } from '../hooks/useCasting';
import { usePlayback } from '../hooks/usePlayback';
import { useProviderToken } from '../hooks/useProviderToken';
import { useQueue } from '../hooks/useQueue';
import { useRoom } from '../hooks/useRoom';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';
import { useThemeStore } from '../stores/themeStore';

interface RoomViewProps {
  initialData?: {
    room?: Room;
    songs?: Song[];
    playback?: PlaybackState;
    theme?: 'light' | 'dark';
  };
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
      className={`flex min-h-screen flex-col ${!isSSR ? 'animate-fade-in' : ''}`}
    >
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
            <div className="mb-5 inline-flex h-20 w-20 items-center justify-center rounded-2xl border-4 border-ink/20 bg-white shadow-retro">
              <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary/30 border-t-primary" />
            </div>
            <p className="font-medium text-ink/60">Loading session...</p>
            <p className="jp-art mt-1 text-ink/40 text-sm">読み込み中</p>
          </div>
        </div>
      ) : error ? (
        <div className="flex flex-1 flex-col items-center justify-center px-4">
          <div className="w-full max-w-md animate-scale-in text-center">
            <div className="mb-5 inline-flex h-20 w-20 items-center justify-center rounded-2xl border-4 border-error bg-white shadow-retro">
              <svg
                className="h-10 w-10 text-error"
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
            </div>
            <h2
              className="mb-2 font-black text-2xl text-ink"
              style={{ fontFamily: 'Poppins' }}
            >
              Connection Failed
            </h2>
            <p className="mb-6 font-medium text-ink/60">{error.message}</p>
            <div className="flex gap-4">
              <button
                onClick={() => fetchRoom()}
                className="glass-elevated cursor-pointer rounded-xl border-2 border-ink/10 px-8 py-3.5 font-bold text-ink transition-all hover:shadow-retro"
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
                      className="cursor-pointer rounded-xl border-2 border-primary bg-primary px-8 py-3.5 font-bold text-white transition-all hover:bg-primary-muted hover:shadow-retro-pink"
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
        <div className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-7xl items-start px-4 py-8 lg:grid lg:grid-cols-[1fr_400px] lg:gap-12">
            {/* Player Section */}
            <div className="space-y-6">
              {/* Player - Reserve height to prevent CLS */}
              <div
                className="relative w-full overflow-hidden rounded-xl bg-gray-100 dark:bg-gray-800"
                style={{ aspectRatio: '16/9', minHeight: '200px' }}
              >
                {displayCurrentSong ? (
                  // Show actual player when there's a current song
                  displayCurrentSong.sourceType === 'spotify' ? (
                    <SpotifyPlayer
                      onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                      isVisible={true}
                    />
                  ) : displayCurrentSong.sourceType === 'soundcloud' ? (
                    <SoundCloudPlayer
                      onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                      isVisible={true}
                    />
                  ) : (
                    <VideoPlayer
                      onEnded={displayRoom?.mode === 'host' ? skip : undefined}
                      isVisible={true}
                    />
                  )
                ) : displaySongs.length > 0 ? (
                  // State 2: Queue exists but nothing playing - show loading
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="text-center">
                      <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 dark:bg-primary/20">
                        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
                      </div>
                      <p className="font-medium text-ink/70 dark:text-dark-text-muted">
                        Loading song...
                      </p>
                    </div>
                  </div>
                ) : (
                  // State 1: No queue - show dancing 8-bit guy
                  <div className="absolute inset-0 flex items-center justify-center overflow-hidden">
                    <div className="text-center">
                      {/* Dancing 8-bit character */}
                      <div className="mb-6">
                        <svg
                          width="160"
                          height="140"
                          viewBox="0 0 160 140"
                          className="mx-auto"
                        >
                          <style>
                            {`
                              @keyframes dance-left-leg {
                                0%, 24% { transform: translateX(0px); }
                                25%, 49% { transform: translateX(-3px); }
                                50%, 74% { transform: translateX(0px); }
                                75%, 100% { transform: translateX(0px); }
                              }
                              @keyframes dance-right-leg {
                                0%, 24% { transform: translateX(0px); }
                                25%, 49% { transform: translateX(0px); }
                                50%, 74% { transform: translateX(0px); }
                                75%, 100% { transform: translateX(3px); }
                              }
                              @keyframes dance-arms {
                                0%, 49% { transform: rotate(0deg); }
                                50%, 99% { transform: rotate(8deg); }
                                100% { transform: rotate(0deg); }
                              }
                              @keyframes dance-body {
                                0%, 49% { transform: translateY(0px); }
                                50%, 99% { transform: translateY(-2px); }
                                100% { transform: translateY(0px); }
                              }
                              @keyframes blink {
                                0%, 90% { height: 8px; }
                                95%, 100% { height: 2px; }
                              }
                              .dance-left-leg { animation: dance-left-leg 1.2s steps(4, end) infinite; }
                              .dance-right-leg { animation: dance-right-leg 1.2s steps(4, end) infinite; }
                              .dance-arms { animation: dance-arms 1s steps(2, end) infinite; }
                              .dance-body { animation: dance-body 1.4s steps(2, end) infinite; }
                              .blink { animation: blink 3s steps(2, end) infinite; }
                            `}
                          </style>

                          {/* 8-bit dancing character with leg movement animation */}
                          <g className="dance-body">
                            {/* Head */}
                            <rect
                              x="60"
                              y="20"
                              width="40"
                              height="40"
                              fill="#FFB366"
                            />
                            <rect
                              x="55"
                              y="25"
                              width="10"
                              height="10"
                              fill="#FFB366"
                            />
                            <rect
                              x="95"
                              y="25"
                              width="10"
                              height="10"
                              fill="#FFB366"
                            />

                            {/* Eyes - with blinking animation */}
                            <rect
                              x="65"
                              y="30"
                              width="8"
                              height="8"
                              fill="#000"
                              className="blink"
                            />
                            <rect
                              x="87"
                              y="30"
                              width="8"
                              height="8"
                              fill="#000"
                              className="blink"
                              style={{ animationDelay: '0.1s' }}
                            />

                            {/* Smile */}
                            <rect
                              x="70"
                              y="45"
                              width="8"
                              height="4"
                              fill="#000"
                            />
                            <rect
                              x="78"
                              y="45"
                              width="4"
                              height="4"
                              fill="#000"
                            />
                            <rect
                              x="82"
                              y="45"
                              width="8"
                              height="4"
                              fill="#000"
                            />

                            {/* Body */}
                            <rect
                              x="55"
                              y="60"
                              width="50"
                              height="35"
                              fill="#4ECDC4"
                            />

                            {/* Arms - animated */}
                            <g
                              className="dance-arms"
                              style={{ transformOrigin: '45px 69px' }}
                            >
                              <rect
                                x="35"
                                y="65"
                                width="20"
                                height="8"
                                fill="#FFB366"
                              />
                            </g>
                            <g
                              className="dance-arms"
                              style={{
                                transformOrigin: '115px 69px',
                                animationDelay: '0.4s',
                              }}
                            >
                              <rect
                                x="105"
                                y="65"
                                width="20"
                                height="8"
                                fill="#FFB366"
                              />
                            </g>

                            {/* Left Leg - animated independently */}
                            <g className="dance-left-leg">
                              <rect
                                x="65"
                                y="95"
                                width="12"
                                height="20"
                                fill="#2C3E50"
                              />
                              {/* Left Foot */}
                              <rect
                                x="60"
                                y="115"
                                width="20"
                                height="5"
                                fill="#000"
                              />
                            </g>

                            {/* Right Leg - animated independently */}
                            <g className="dance-right-leg">
                              <rect
                                x="83"
                                y="95"
                                width="12"
                                height="20"
                                fill="#2C3E50"
                              />
                              {/* Right Foot */}
                              <rect
                                x="85"
                                y="115"
                                width="20"
                                height="5"
                                fill="#000"
                              />
                            </g>
                          </g>

                          {/* Musical notes floating around - repositioned to stay within bounds */}
                          <g className="animate-ping">
                            <circle
                              cx="35"
                              cy="40"
                              r="3"
                              fill="#FF6B6B"
                              opacity="0.7"
                            />
                            <rect
                              x="33"
                              y="35"
                              width="2"
                              height="8"
                              fill="#FF6B6B"
                              opacity="0.7"
                            />
                          </g>
                          <g
                            className="animate-ping"
                            style={{ animationDelay: '0.5s' }}
                          >
                            <circle
                              cx="125"
                              cy="50"
                              r="3"
                              fill="#4ECDC4"
                              opacity="0.7"
                            />
                            <rect
                              x="123"
                              y="45"
                              width="2"
                              height="8"
                              fill="#4ECDC4"
                              opacity="0.7"
                            />
                          </g>
                          <g
                            className="animate-ping"
                            style={{ animationDelay: '1s' }}
                          >
                            <circle
                              cx="25"
                              cy="70"
                              r="3"
                              fill="#FFE66D"
                              opacity="0.7"
                            />
                            <rect
                              x="23"
                              y="65"
                              width="2"
                              height="8"
                              fill="#FFE66D"
                              opacity="0.7"
                            />
                          </g>
                        </svg>
                      </div>

                      <h3 className="mb-2 font-bold text-ink text-xl dark:text-dark-text">
                        Add a song to get the party started!
                      </h3>
                      <p className="text-ink/60 text-sm dark:text-dark-text-muted">
                        Click the "Add Song" button to start the music
                      </p>
                    </div>
                  </div>
                )}
              </div>

              {/* Controls (always below video) */}
              <PlayerControls
                isPlaying={displayIsPlaying}
                canPlay={Boolean(displayCurrentSong || displaySongs.length > 0)}
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
            <div className="mt-8 space-y-8 lg:mt-0">
              <div>
                {/* Now Playing (Integrated into list style) */}
                {displayCurrentSong && (
                  <div className="mb-8">
                    <div className="mb-3 flex items-center gap-2">
                      <div
                        className={`h-2 w-2 rounded-full ${displayIsPlaying ? 'animate-pulse bg-matcha shadow-neon-pink' : 'bg-ink/30'}`}
                      />
                      <span className="font-black text-[10px] text-ink/60 uppercase tracking-[0.2em] dark:text-dark-text-muted">
                        {displayIsPlaying ? 'Now Playing' : 'Paused'}
                      </span>
                    </div>

                    <div className="glass-elevated group/card relative flex items-center gap-4 overflow-hidden rounded-2xl border-2 border-primary/20 bg-white/40 p-4 shadow-retro-pink/20 backdrop-blur-sm dark:bg-dark-surface/60">
                      {/* Progress Background - only animate on client */}
                      <div
                        className="pointer-events-none absolute inset-y-0 left-0 bg-primary/10 dark:bg-primary/20"
                        style={{ width: isSSR ? '0%' : `${progress * 100}%` }}
                      />

                      {displayCurrentSong.thumbnailUrl && (
                        <div className="relative z-10 shrink-0">
                          <img
                            src={displayCurrentSong.thumbnailUrl}
                            alt=""
                            className="h-16 w-16 rounded-xl border-2 border-ink/10 object-cover shadow-xs transition-transform group-hover/card:scale-105 dark:border-primary/20"
                          />
                        </div>
                      )}
                      <div className="relative z-10 min-w-0 flex-1">
                        <h3 className="truncate font-bold text-ink text-sm dark:text-dark-text">
                          {displayCurrentSong.title}
                        </h3>
                        <p className="truncate font-medium text-ink/60 text-xs dark:text-dark-text-muted">
                          {displayCurrentSong.artist || 'Unknown Artist'} •{' '}
                          {formatTime(displayCurrentSong.duration * 1000)}
                        </p>
                      </div>
                    </div>

                    <div className="mt-8 mb-4 h-px bg-ink/10 dark:bg-primary/20" />
                  </div>
                )}

                {/* Up Next List */}
                <div>
                  <h3 className="mb-4 font-black text-[10px] text-ink/60 uppercase tracking-[0.2em] dark:text-dark-text-muted">
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
  );
}
