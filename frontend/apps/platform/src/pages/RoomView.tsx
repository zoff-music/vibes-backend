import { SoundCloudPlayer, SpotifyPlayer, VideoPlayer } from '@vibez/player';
import { Song, usePlaybackStore } from '@vibez/shared';
import { AnimatePresence, motion } from 'framer-motion';
import { QRCodeSVG } from 'qrcode.react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { getHttpError } from 'wiretyped';
import { PlayerControls } from '../components/player/PlayerControls';
import { AddToQueueModal } from '../components/queue/AddToQueueModal';
import { QueueList } from '../components/queue/QueueList';
import { UserCount } from '../components/room/UserCount';
import { Toast } from '../components/ui/Toast';
import { usePlayback } from '../hooks/usePlayback';
import { useProviderToken } from '../hooks/useProviderToken';
import { useQueue } from '../hooks/useQueue';
import { useRoom } from '../hooks/useRoom';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';
import { useThemeStore } from '../stores/themeStore';

interface RoomViewProps {
  initialData?: {
    room?: any;
    songs?: any[];
    playback?: any;
    theme?: 'light' | 'dark';
  };
}

export default function RoomView({ initialData }: RoomViewProps) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { currentSong, skip, isPlaying } = usePlayback(id || '');
  const { songs, fetchQueue, voteSong } = useQueue(id || '');

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
  const [toasts, setToasts] = useState<
    { id: string; message: string; type: 'success' | 'info' | 'error' }[]
  >([]);
  const [adminPassword, setAdminPassword] = useState('');
  const [isAuthenticating, setIsAuthenticating] = useState(false);

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
    const handleSongAdded = (e: any) => {
      console.log('[UI] song-added event received:', e.detail);
      const song = e.detail as Song;
      setToasts((prev) => [
        ...prev,
        {
          id: Math.random().toString(36).substr(2, 9),
          message: `"${song.title}" added to queue`,
          type: 'success',
        },
      ]);
    };

    const handleShowToast = (e: any) => {
      const { message, type } = e.detail;
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
      <div
        ref={headerRef}
        className="sticky top-0 z-20 border-ink/10 border-b-4 bg-white/95 px-4 py-5 shadow-retro backdrop-blur-lg transition-colors duration-300 dark:border-primary/20 dark:bg-dark-paper/95"
      >
        <div className="relative mx-auto flex max-w-7xl items-center justify-between">
          <Link
            to="/"
            className="group inline-flex cursor-pointer items-center gap-2 text-ink/60 transition-colors hover:text-ink dark:text-dark-text-muted dark:hover:text-dark-text"
          >
            <svg
              className="h-5 w-5 transition-transform group-hover:-translate-x-1"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 19l-7-7 7-7"
              />
            </svg>
            <span className="font-bold text-sm tracking-wide">Leave</span>
          </Link>

          {/* Room name - absolutely positioned to stay centered */}
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
            <button
              onClick={() => setShowRoomInfo(!showRoomInfo)}
              className="flex cursor-pointer items-center justify-center gap-2 transition-opacity hover:opacity-70"
            >
              <h1
                className="truncate whitespace-nowrap font-black text-ink text-lg dark:text-dark-text"
                style={{ fontFamily: 'Poppins' }}
              >
                {displayRoom?.name || 'Loading...'}
              </h1>
              <svg
                className={`h-4 w-4 text-ink/50 transition-transform dark:text-dark-text-muted ${showRoomInfo ? 'rotate-180' : ''}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </button>
          </div>

          <div className="flex items-center gap-2">
            <UserCount />

            {/* Dark Mode Toggle */}
            <div className="hidden sm:block">
              <button
                onClick={handleToggleDarkMode}
                className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${
                  isDarkMode
                    ? 'border-primary bg-primary text-white shadow-neon-pink'
                    : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink'
                }`}
                title={
                  isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'
                }
              >
                {isDarkMode ? (
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
                      d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
                    />
                  </svg>
                ) : (
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
                      d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
                    />
                  </svg>
                )}
              </button>
            </div>

            <div className="relative hidden sm:block">
              <button
                onClick={() => setShowShare(!showShare)}
                className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${showShare ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary' : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'}`}
                title="Share Room"
              >
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
                    d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"
                  />
                </svg>
              </button>

              <AnimatePresence>
                {showShare && (
                  <motion.div
                    initial={{ opacity: 0, y: 10, scale: 0.95 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    exit={{ opacity: 0, y: 10, scale: 0.95 }}
                    className="absolute right-0 z-50 mt-3 w-72 rounded-3xl border-4 border-ink bg-white p-4 shadow-2xl dark:border-primary dark:bg-dark-surface"
                  >
                    <div className="space-y-4 text-center">
                      <div className="inline-block rounded-2xl bg-sakura/20 p-4 ring-2 ring-ink/5">
                        <QRCodeSVG
                          value={window.location.href}
                          size={180}
                          bgColor="#fff0f2"
                          fgColor="#2d3142"
                          level="H"
                        />
                      </div>
                      <div>
                        <p className="mb-1 font-black text-ink text-sm dark:text-dark-text">
                          Invite Friends
                        </p>
                        <div className="flex items-center gap-2 rounded-xl bg-ink/5 p-2 dark:bg-dark-surfaceElevated">
                          <p className="flex-1 truncate text-left font-mono text-[10px] text-ink/60 dark:text-dark-text-muted">
                            {window.location.href}
                          </p>
                          <button
                            onClick={() => {
                              navigator.clipboard.writeText(
                                window.location.href,
                              );
                              setToasts((prev) => [
                                ...prev,
                                {
                                  id: Math.random().toString(36).substr(2, 9),
                                  message: 'Link copied!',
                                  type: 'success',
                                },
                              ]);
                              setShowShare(false);
                            }}
                            className="cursor-pointer rounded-lg bg-ink p-1 px-2 font-bold text-[10px] text-white transition-all hover:scale-105 active:scale-95 dark:bg-primary"
                          >
                            Copy
                          </button>
                        </div>
                      </div>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            <div className="relative ml-1">
              <button
                onClick={() => setShowSettings(!showSettings)}
                className={`cursor-pointer rounded-xl border-2 p-2.5 transition-all ${showSettings ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary' : 'border-ink/10 text-ink/60 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'}`}
                title="Room Settings"
              >
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
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                </svg>
              </button>

              <AnimatePresence>
                {showSettings && (
                  <motion.div
                    initial={{ opacity: 0, y: 10, scale: 0.95 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    exit={{ opacity: 0, y: 10, scale: 0.95 }}
                    className="fixed top-[var(--room-header-height)] right-0 left-0 z-50 h-[calc(100dvh-var(--room-header-height))] w-full overflow-y-auto overscroll-contain border-ink border-t-4 bg-white p-5 shadow-2xl sm:absolute sm:top-auto sm:right-0 sm:left-auto sm:mt-3 sm:h-auto sm:w-72 sm:overflow-visible sm:rounded-3xl sm:border-4 dark:border-primary dark:bg-dark-surface"
                  >
                    <div className="space-y-4">
                      <div className="space-y-3 sm:hidden">
                        <div className="flex items-center gap-3">
                          <button
                            onClick={() => setShowShare(!showShare)}
                            className={`flex flex-1 items-center justify-center gap-2 rounded-xl border-2 p-3 font-bold text-xs transition-all ${
                              showShare
                                ? 'border-ink bg-ink text-white dark:border-primary dark:bg-primary'
                                : 'border-ink/10 text-ink/70 hover:border-ink/20 hover:text-ink dark:border-primary/20 dark:text-dark-text-muted dark:hover:text-dark-text'
                            }`}
                            title="Share Room"
                          >
                            <svg
                              className="h-4 w-4"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                              strokeWidth={2.5}
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"
                              />
                            </svg>
                            Share
                          </button>

                          <button
                            onClick={handleToggleDarkMode}
                            className={`flex flex-1 items-center justify-center gap-2 rounded-xl border-2 p-3 font-bold text-xs transition-all ${
                              isDarkMode
                                ? 'border-primary bg-primary text-white shadow-neon-pink'
                                : 'border-ink/10 text-ink/70 hover:border-ink/20 hover:text-ink'
                            }`}
                            title={
                              isDarkMode
                                ? 'Switch to Light Mode'
                                : 'Switch to Dark Mode'
                            }
                          >
                            {isDarkMode ? (
                              <svg
                                className="h-4 w-4"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={2.5}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
                                />
                              </svg>
                            ) : (
                              <svg
                                className="h-4 w-4"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={2.5}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
                                />
                              </svg>
                            )}
                            Theme
                          </button>
                        </div>

                        {showShare && (
                          <div className="rounded-2xl border-2 border-ink/10 bg-white p-4 dark:border-primary/20 dark:bg-dark-surfaceElevated">
                            <div className="space-y-4 text-center">
                              <div className="inline-block rounded-2xl bg-sakura/20 p-4 ring-2 ring-ink/5">
                                <QRCodeSVG
                                  value={window.location.href}
                                  size={180}
                                  bgColor="#fff0f2"
                                  fgColor="#2d3142"
                                  level="H"
                                />
                              </div>
                              <div>
                                <p className="mb-1 font-black text-ink text-sm dark:text-dark-text">
                                  Invite Friends
                                </p>
                                <div className="flex items-center gap-2 rounded-xl bg-ink/5 p-2 dark:bg-dark-surface">
                                  <p className="flex-1 truncate text-left font-mono text-[10px] text-ink/60 dark:text-dark-text-muted">
                                    {window.location.href}
                                  </p>
                                  <button
                                    onClick={() => {
                                      navigator.clipboard.writeText(
                                        window.location.href,
                                      );
                                      setToasts((prev) => [
                                        ...prev,
                                        {
                                          id: Math.random()
                                            .toString(36)
                                            .substr(2, 9),
                                          message: 'Link copied!',
                                          type: 'success',
                                        },
                                      ]);
                                      setShowShare(false);
                                    }}
                                    className="cursor-pointer rounded-lg bg-ink p-1 px-2 font-bold text-[10px] text-white transition-all hover:scale-105 active:scale-95 dark:bg-primary"
                                  >
                                    Copy
                                  </button>
                                </div>
                              </div>
                            </div>
                          </div>
                        )}
                      </div>

                      <h4 className="border-ink/5 border-b-2 pb-2 font-black text-ink text-sm uppercase tracking-wider dark:border-primary/20 dark:text-dark-text">
                        Room Control
                      </h4>

                      {/* Host Mode Disclaimer */}
                      {room?.mode === 'host' && (
                        <div className="rounded-lg border border-secondary/20 bg-secondary/10 p-3 dark:border-secondary/30 dark:bg-secondary/20">
                          <div className="flex items-center gap-2">
                            <div className="h-2 w-2 rounded-full bg-secondary"></div>
                            <span className="font-bold text-secondary text-sm dark:text-secondary">
                              Host Mode Active
                            </span>
                          </div>
                          <p className="mt-1 text-ink/70 text-xs dark:text-dark-text-muted">
                            In host mode, only the host can skip songs. Skip
                            settings are disabled.
                          </p>
                        </div>
                      )}

                      <div className="group flex items-center justify-between">
                        <div className="flex flex-col">
                          <span
                            className={`font-bold text-sm ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                          >
                            Allow Skip
                          </span>
                          <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
                            {room?.mode === 'host'
                              ? 'Host controls skipping'
                              : 'Anyone can skip'}
                          </span>
                        </div>
                        <button
                          disabled={
                            (room?.hasPassword && !isAdmin) ||
                            room?.mode === 'host'
                          }
                          onClick={() =>
                            room &&
                            updateRoomSettings({
                              ...room.settings,
                              skipAllowed: !room.settings.skipAllowed,
                            })
                          }
                          className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.skipAllowed ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                        >
                          <div
                            className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.skipAllowed ? 'right-1.5' : 'left-1.5'}`}
                          />
                        </button>
                      </div>

                      <div className="group flex items-center justify-between">
                        <div className="flex flex-col">
                          <span
                            className={`font-bold text-sm ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                          >
                            Democratic Skip
                          </span>
                          <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
                            {room?.mode === 'host'
                              ? 'Host decides skipping'
                              : 'Require votes'}
                          </span>
                        </div>
                        <button
                          disabled={
                            (room?.hasPassword && !isAdmin) ||
                            room?.mode === 'host'
                          }
                          onClick={() =>
                            room &&
                            updateRoomSettings({
                              ...room.settings,
                              democraticSkip: !room.settings.democraticSkip,
                            })
                          }
                          className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.democraticSkip ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${(room?.hasPassword && !isAdmin) || room?.mode === 'host' ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                        >
                          <div
                            className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.democraticSkip ? 'right-1.5' : 'left-1.5'}`}
                          />
                        </button>
                      </div>

                      <div className="group flex items-center justify-between">
                        <div className="flex flex-col">
                          <span
                            className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                          >
                            Loop Queue
                          </span>
                          <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
                            Cycled back to end
                          </span>
                        </div>
                        <button
                          disabled={room?.hasPassword && !isAdmin}
                          onClick={() =>
                            room &&
                            updateRoomSettings({
                              ...room.settings,
                              loopQueue: !room.settings.loopQueue,
                            })
                          }
                          className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.loopQueue ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                        >
                          <div
                            className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.loopQueue ? 'right-1.5' : 'left-1.5'}`}
                          />
                        </button>
                      </div>

                      <div className="group flex items-center justify-between">
                        <div className="flex flex-col">
                          <span
                            className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                          >
                            Allow Duplicates
                          </span>
                          <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
                            Same song multiple times
                          </span>
                        </div>
                        <button
                          disabled={room?.hasPassword && !isAdmin}
                          onClick={() =>
                            room &&
                            updateRoomSettings({
                              ...room.settings,
                              allowDuplicates: !room.settings.allowDuplicates,
                            })
                          }
                          className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.allowDuplicates ? 'bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                        >
                          <div
                            className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.allowDuplicates ? 'right-1.5' : 'left-1.5'}`}
                          />
                        </button>
                      </div>

                      <div className="group flex items-center justify-between">
                        <div className="flex flex-col">
                          <span
                            className={`font-bold text-sm ${room?.hasPassword && !isAdmin ? 'text-ink/30 dark:text-dark-text-subtle' : 'text-ink dark:text-dark-text'}`}
                          >
                            Remove Played
                          </span>
                          <span className="text-[10px] text-ink/40 dark:text-dark-text-subtle">
                            Removed after play
                          </span>
                        </div>
                        <button
                          disabled={room?.hasPassword && !isAdmin}
                          onClick={() =>
                            room &&
                            updateRoomSettings({
                              ...room.settings,
                              removeOnPlay: !room.settings.removeOnPlay,
                            })
                          }
                          className={`relative h-6 w-12 cursor-pointer rounded-full border-2 border-ink transition-colors dark:border-primary ${room?.settings.removeOnPlay ? 'bg-ink dark:bg-primary' : 'bg-ink/5 opacity-50 dark:bg-dark-surfaceElevated'} ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                        >
                          <div
                            className={`absolute top-1 h-2 w-2 rounded-full bg-white shadow-xs transition-all ${room?.settings.removeOnPlay ? 'right-1.5' : 'left-1.5'}`}
                          />
                        </button>
                      </div>

                      <div className="border-ink/5 border-t-2 pt-4 dark:border-primary/20">
                        <h5 className="mb-3 font-black text-ink text-xs uppercase tracking-wider dark:text-dark-text">
                          Room Mode
                        </h5>

                        <div className="space-y-2">
                          <button
                            disabled={room?.hasPassword && !isAdmin}
                            onClick={() =>
                              room && updateRoom({ mode: 'server' })
                            }
                            className={`w-full cursor-pointer rounded-xl border-2 p-3 text-left transition-all ${
                              room?.mode === 'server'
                                ? 'border-primary bg-primary/10 text-ink dark:border-primary dark:bg-primary/20 dark:text-dark-text'
                                : 'border-ink/10 bg-surface text-ink/60 hover:border-ink/20 dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text-muted dark:hover:border-primary/30'
                            } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                          >
                            <div className="mb-1 font-bold text-sm">
                              Server Mode
                            </div>
                            <div className="text-[10px] opacity-70">
                              Auto-play music 24/7. Perfect for radio stations.
                            </div>
                          </button>

                          <button
                            disabled={room?.hasPassword && !isAdmin}
                            onClick={() => room && updateRoom({ mode: 'host' })}
                            className={`w-full cursor-pointer rounded-xl border-2 p-3 text-left transition-all ${
                              room?.mode === 'host'
                                ? 'border-secondary bg-secondary/10 text-ink dark:border-secondary dark:bg-secondary/20 dark:text-dark-text'
                                : 'border-ink/10 bg-surface text-ink/60 hover:border-ink/20 dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text-muted dark:hover:border-primary/30'
                            } ${room?.hasPassword && !isAdmin ? 'cursor-not-allowed opacity-30 grayscale' : ''}`}
                          >
                            <div className="mb-1 font-bold text-sm">
                              Host Mode
                            </div>
                            <div className="text-[10px] opacity-70">
                              Host controls playback. Great for parties.
                            </div>
                          </button>
                        </div>
                      </div>

                      {!isAdmin && (
                        <div className="group mt-6 flex flex-col gap-2 border-ink/5 border-t-2 pt-4 dark:border-primary/20">
                          <span className="font-bold text-ink text-sm dark:text-dark-text">
                            Admin Access
                          </span>
                          <div className="flex gap-2">
                            <input
                              type="password"
                              value={adminPassword}
                              onChange={(e) => setAdminPassword(e.target.value)}
                              placeholder={
                                displayRoom?.hasPassword
                                  ? 'Login as admin'
                                  : 'Add password'
                              }
                              className="flex-1 rounded-xl border-2 border-ink/10 bg-white px-3 py-2 text-ink text-sm outline-none transition-all focus:border-primary/50 dark:border-primary/20 dark:bg-dark-paper dark:text-dark-text"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') handleJoinAdmin();
                              }}
                            />
                            <button
                              onClick={handleJoinAdmin}
                              disabled={isAuthenticating || !adminPassword}
                              className="cursor-pointer rounded-xl bg-ink p-2 px-4 font-bold text-white text-xs transition-all hover:bg-primary disabled:cursor-not-allowed disabled:opacity-50 dark:bg-primary"
                            >
                              {isAuthenticating ? '...' : 'Go'}
                            </button>
                          </div>
                        </div>
                      )}

                      {isAdmin && (
                        <div className="group mt-6 border-ink/5 border-t-2 pt-4 text-center dark:border-primary/20">
                          <span className="font-bold text-matcha text-sm">
                            ✓ You are an Admin
                          </span>
                        </div>
                      )}

                      <p className="pt-2 text-center font-black text-[10px] text-ink/30 italic dark:text-dark-text-subtle">
                        Settings sync enabled
                      </p>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
        </div>

        {/* Room info dropdown */}
        {showRoomInfo && (
          <div
            className={`glass-elevated mt-4 rounded-2xl border-2 border-ink/10 p-5 dark:border-primary/20 ${!isSSR ? 'animate-slide-down' : ''}`}
          >
            <div className="space-y-4">
              <div>
                <p className="mb-2 font-bold text-ink/60 text-xs uppercase tracking-widest dark:text-dark-text-muted">
                  Room Code
                </p>
                <div className="flex items-center gap-2">
                  <code className="rounded-xl border-2 border-ink/20 bg-surface px-4 py-2 font-bold font-mono text-ink text-sm dark:border-primary/20 dark:bg-dark-surfaceElevated dark:text-dark-text">
                    {id}
                  </code>
                  <button
                    onClick={() => navigator.clipboard.writeText(id || '')}
                    className="cursor-pointer rounded-lg border-2 border-transparent p-2 transition-colors hover:border-ink/10 hover:bg-ink/5"
                    title="Copy code"
                  >
                    <svg
                      className="h-5 w-5 text-ink/60"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                      />
                    </svg>
                  </button>
                </div>
              </div>
              {users && users.length > 0 && (
                <div>
                  <p className="mb-2 font-bold text-ink/60 text-xs uppercase tracking-widest">
                    <span className="hidden sm:inline">Listeners </span>(
                    {users.length})
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {users.slice(0, 8).map((user: any) => (
                      <div
                        key={user.id}
                        className="rounded-full border border-sakura/50 bg-sakura/30 px-3 py-1.5 font-medium text-ink/70 text-xs"
                      >
                        {user.nickname || `User ${user.id.slice(0, 4)}`}
                      </div>
                    ))}
                    {users.length > 8 && (
                      <div className="rounded-full bg-ink/10 px-3 py-1.5 font-medium text-ink/50 text-xs">
                        +{users.length - 8} more
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

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
                roomId={id || ''}
                hasSongsInQueue={displaySongs && displaySongs.length > 0}
                onAddSong={handleAddSong}
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

      {/* Add Song Modal */}
      <AddToQueueModal
        roomId={id || ''}
        isVisible={isAddModalVisible}
        onClose={() => setIsAddModalVisible(false)}
      />
    </div>
  );
}
