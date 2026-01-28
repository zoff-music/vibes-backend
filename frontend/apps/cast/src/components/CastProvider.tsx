/// <reference types="chromecast-caf-receiver" />

import { api } from '@vibez/api';
import {
  type PlaybackState,
  type Room,
  type Song,
  safeWrap,
  setCachedToken,
  usePlaybackStore,
} from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';

// Types are available globally via @types/chromecast-caf-receiver

export interface RoomInfo {
  name: string;
  participantCount: number;
}

export type QueueItem = Song;

type LocalCastMessage =
  | {
      action: 'updatePlayback';
      currentSong?: QueueItem;
      isPlaying?: boolean;
      positionMs?: number;
      queue?: QueueItem[];
      roomInfo?: RoomInfo;
    }
  | {
      action: 'syncPlayback';
      currentSong?: QueueItem;
      isPlaying?: boolean;
      positionMs?: number;
    }
  | {
      action: 'updateQueue';
      queue?: QueueItem[];
    }
  | {
      action: 'updateRoomInfo';
      roomInfo?: RoomInfo;
    };

// Global flag to prevent multiple Cast receiver initializations
let isCastReceiverInitialized = false;

interface CastContextType {
  roomInfo: RoomInfo | null;
  queue: QueueItem[];
  statusText: string;
  roomMode: string | null;
  currentSong: Song | null;
  actualPositionMs: number;
  updateActualPosition: () => void;
  debugMode: boolean;
}

const CastContext = createContext<CastContextType | undefined>(undefined);

export const CastProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [roomInfo, setRoomInfo] = useState<RoomInfo | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [statusText, setStatusText] = useState('Ready for Casting');
  const [roomMode, setRoomMode] = useState<string | null>(null);
  const [debugMode, setDebugModeState] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('debug') === 'true';
  });
  const debugModeRef = useRef(debugMode);

  const setDebugMode = useCallback((value: boolean) => {
    setDebugModeState(value);
    debugModeRef.current = value;
  }, []);

  // Use global store for playback state to share with components
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setIsPlaying = usePlaybackStore((state) => state.setIsPlaying);
  const updateActualPosition = usePlaybackStore(
    (state) => state.updateActualPosition,
  );
  const actualPositionMs = usePlaybackStore((state) => state.actualPositionMs);
  const currentSong = usePlaybackStore((state) => state.currentSong);

  // Cast Context Ref
  const contextRef = useRef<framework.CastReceiverContext | null>(null);
  const playerManagerRef = useRef<framework.PlayerManager | null>(null);

  const normalizeSong = useCallback((song: Song): Song => {
    return {
      id: song.id,
      sourceType: song.sourceType,
      sourceId: song.sourceId,
      title: song.title,
      artist: song.artist,
      thumbnailUrl: song.thumbnailUrl || '',
      duration: song.duration || 0,
      position: song.position ?? 0,
      addedBy: song.addedBy || 'cast-receiver',
      addedAt: song.addedAt || new Date().toISOString(),
      addedByNickname: song.addedByNickname,
      voteCount: song.voteCount,
    };
  }, []);

  // Helper to update CAF metadata for external controls
  const updateMediaMetadata = useCallback((song: Song) => {
    const playerManager = playerManagerRef.current;
    if (!playerManager) return;

    const [err] = safeWrap(() => {
      const mediaInfo =
        playerManager.getMediaInformation() ||
        new cast.framework.messages.MediaInformation();

      const metadata = new cast.framework.messages.MusicTrackMediaMetadata();
      metadata.title = song.title;
      metadata.artist = song.artist || 'Unknown Artist';
      metadata.images = song.thumbnailUrl
        ? [new cast.framework.messages.Image(song.thumbnailUrl)]
        : [];

      mediaInfo.metadata = metadata;
      mediaInfo.contentId = song.id; // Or sourceId
      mediaInfo.contentType = 'audio/mpeg'; // Generic content type
      mediaInfo.streamType = cast.framework.messages.StreamType.BUFFERED;
      mediaInfo.duration = song.duration || 0;

      playerManager.setMediaInformation(mediaInfo);

      // Force a status broadcast
      // Extending the type locally to avoid 'any' casting for missing method definition
      interface ExtendedPlayerManager extends framework.PlayerManager {
        broadcastStatus(includeMediaStatus: boolean): void;
      }
      (playerManager as ExtendedPlayerManager).broadcastStatus?.(true);
    });

    if (err) {
      console.error('[Cast Receiver] Failed to update metadata', err);
    }
  }, []);

  const handleCastMessage = useCallback(
    (message: LocalCastMessage) => {
      console.log('[Cast Receiver] custom message payload', {
        action: message?.action,
        hasCurrentSong:
          'currentSong' in message ? !!message.currentSong : false,
        hasQueue: 'queue' in message ? Array.isArray(message.queue) : false,
        hasRoomInfo: 'roomInfo' in message ? !!message.roomInfo : false,
      });

      const action = message.action;
      switch (action) {
        case 'updatePlayback':
          console.log('Updating playback state:', message);
          if (message.currentSong) {
            const normalizedSong = normalizeSong(message.currentSong);
            setPlaybackState(
              {
                currentSong: normalizedSong,
                isPlaying: message.isPlaying || false,
                positionMs: message.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              },
              roomMode || undefined,
            );
            setIsPlaying(message.isPlaying || false);
            setStatusText(`Now Playing: ${message.currentSong.title}`);
            updateMediaMetadata(normalizedSong);
          }
          if (message.roomInfo) {
            setRoomInfo(message.roomInfo);
          }
          break;

        case 'syncPlayback':
          console.log('Syncing playback:', message);
          if (message.currentSong) {
            const normalizedSong = normalizeSong(message.currentSong);
            setPlaybackState(
              {
                currentSong: normalizedSong,
                isPlaying: message.isPlaying || false,
                positionMs: message.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              },
              roomMode || undefined,
            );
            setIsPlaying(message.isPlaying || false);
            updateMediaMetadata(normalizedSong);
          }
          break;

        case 'updateQueue':
          console.log('Updating queue:', message);
          if (message.queue) {
            setQueue(message.queue);
          }
          break;

        case 'updateRoomInfo':
          console.log('Updating room info:', message);
          if (message.roomInfo) {
            setRoomInfo(message.roomInfo);
          }
          break;

        default:
          console.log('Unknown message action:', action);
      }
    },
    [
      normalizeSong,
      roomMode,
      setIsPlaying,
      setPlaybackState,
      updateMediaMetadata,
    ],
  );

  useEffect(() => {
    const initCast = () => {
      // Prevent double initialization using global flag
      if (isCastReceiverInitialized) {
        console.log('Cast Receiver already initialized globally, skipping...');
        return;
      }

      console.log('[Cast Receiver] Initializing Cast Receiver...', {
        userAgent: navigator.userAgent,
        location: window.location.href,
      });
      const context = cast.framework.CastReceiverContext.getInstance();
      const playerManager = context.getPlayerManager();

      contextRef.current = context;
      playerManagerRef.current = playerManager;

      playerManager.addEventListener(
        cast.framework.events.EventType.MEDIA_STATUS,
        (event: framework.events.MediaStatusEvent) => {
          console.log('[Cast Receiver] MEDIA_STATUS', {
            playerState: event.mediaStatus?.playerState,
            idleReason: event.mediaStatus?.idleReason,
            currentTime: playerManager.getCurrentTimeSec(),
            duration: playerManager.getDurationSec(),
          });
        },
      );
      playerManager.addEventListener(
        cast.framework.events.EventType.TIME_UPDATE,
        (event: framework.events.MediaElementEvent) => {
          console.log('[Cast Receiver] TIME_UPDATE', {
            currentTime: event.currentMediaTime,
            duration: playerManager.getDurationSec(),
          });
        },
      );
      playerManager.addEventListener(
        cast.framework.events.EventType.ERROR,
        (event: framework.events.ErrorEvent) => {
          console.error('[Cast Receiver] PLAYER_ERROR', {
            detailedErrorCode: event.detailedErrorCode,
            error: event.error,
            reason: event.reason,
            severity: event.severity,
          });
        },
      );

      // --- Message Interceptor for LOAD requests ---
      playerManager.setMessageInterceptor(
        cast.framework.messages.MessageType.LOAD,
        (loadRequestData) => {
          console.log('Intercepted LOAD request:', loadRequestData);

          const media = loadRequestData.media;
          if (media?.customData) {
            const data = media.customData as any;

            // Handle tokens if present
            if (data.tokens) {
              for (const [provider, tokenData] of Object.entries(
                data.tokens as Record<string, any>,
              )) {
                if (tokenData?.token) {
                  console.log('[Cast Receiver] caching token for provider', {
                    provider,
                  });
                  setCachedToken(
                    provider,
                    tokenData.token,
                    tokenData.expiresAt ||
                      new Date(Date.now() + 3600000).toISOString(),
                  );
                }
              }
            }

            // Handle debug mode
            if (data.debug) {
              console.log('[Cast Receiver] Enabling debug mode from sender');
              setDebugMode(true);
            }

            // Handle initial playback state from LOAD request
            // This is critical because the default player cannot play our content
            // so we must ingest the state here and return null to prevent IDLE state
            if (data.currentSong) {
              console.log(
                '[Cast Receiver] Initializing playback from LOAD request',
              );
              const normalizedSong = normalizeSong(data.currentSong);

              setPlaybackState({
                currentSong: normalizedSong,
                isPlaying: true, // Auto-play on load
                positionMs: data.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              });
              setIsPlaying(true);
              setStatusText(`Now Playing: ${normalizedSong.title}`);
              updateMediaMetadata(normalizedSong);
            }
          }

          // Return null to prevent the default player from trying to load the content
          // (which would fail for custom metadata and result in IDLE state)
          return null;
        },
      );

      // --- Custom Message Handler for Vibez messages ---
      context.addCustomMessageListener(
        'urn:x-cast:com.vibez.cast',
        (customEvent) => {
          console.log('Received custom message:', customEvent);

          const message = customEvent.data as LocalCastMessage;
          handleCastMessage(message);
        },
      );

      const options = new cast.framework.CastReceiverOptions();
      options.maxInactivity = 3600;
      options.statusText = 'Vibez Session';
      options.disableIdleTimeout = true; // IMPORTANT for keeping session alive during custom playback

      // --- Debug Logger Initialization ---
      const castDebugLogger = cast.debug.CastDebugLogger.getInstance();
      const LOG_TAG = 'VibezApp';

      // Enable debug logger and show the overlay
      // This is "forcing" dev mode as requested
      castDebugLogger.setEnabled(true);
      castDebugLogger.showDebugLogs(true);

      // Set verbosity level
      castDebugLogger.loggerLevelByEvents = {
        'cast.framework.events.category.CORE': cast.framework.LoggerLevel.INFO,
        'cast.framework.events.EventType.MEDIA_STATUS':
          cast.framework.LoggerLevel.DEBUG,
      };

      // Set custom tags logging
      castDebugLogger.loggerLevelByTags = {
        [LOG_TAG]: cast.framework.LoggerLevel.DEBUG,
      };

      // Helper to safely serialize args
      const serializeArgs = (args: any[]) => {
        return args.map((arg) => {
          if (typeof arg === 'object' && arg !== null) {
            try {
              return JSON.stringify(arg, (key, value) => {
                if (key === 'source' && value?.tagName) return '[DOM Element]'; // Circular DOM refs
                return value;
              });
            } catch (_e) {
              return String(arg);
            }
          }
          return String(arg);
        });
      };

      const originalConsole = {
        log: console.log,
        info: console.info,
        warn: console.warn,
        error: console.error,
        debug: console.debug,
      };

      const sendLogToSender = (level: string, args: any[]) => {
        // Use Ref.current to avoid closure staleness issues
        if (!debugModeRef.current) return;

        // Use checks to ensure we can actually send
        const ctx = cast.framework.CastReceiverContext.getInstance();
        // Check if there are connected senders to avoid useless work
        if (ctx.getSenders().length === 0) return;

        const [err] = safeWrap(() => {
          ctx.sendCustomMessage('urn:x-cast:com.vibez.cast', undefined, {
            action: 'LOG',
            level,
            args: serializeArgs(args),
            timestamp: Date.now(),
          });
        });

        if (err) {
          // Fallback to original console if sending fails (recursion prevention)
          // originalConsole.error('Failed to forward log to sender', err);
        }
      };

      console.log = (...args) => {
        castDebugLogger.info(LOG_TAG, ...args);
        originalConsole.log(...args);
        sendLogToSender('info', args);
      };

      console.info = (...args) => {
        castDebugLogger.info(LOG_TAG, ...args);
        originalConsole.info(...args);
        sendLogToSender('info', args);
      };

      console.warn = (...args) => {
        castDebugLogger.warn(LOG_TAG, ...args);
        originalConsole.warn(...args);
        sendLogToSender('warn', args);
      };

      console.error = (...args) => {
        castDebugLogger.error(LOG_TAG, ...args);
        originalConsole.error(...args);
        sendLogToSender('error', args);
      };

      console.debug = (...args) => {
        castDebugLogger.debug(LOG_TAG, ...args);
        originalConsole.debug(...args);
        sendLogToSender('debug', args);
      };

      const [err] = safeWrap(() => {
        context.start(options);
        isCastReceiverInitialized = true;
        castDebugLogger.info(
          LOG_TAG,
          'Cast Receiver started with Debug Logger enabled',
        );
      });
      if (err) {
        castDebugLogger.error(LOG_TAG, 'Failed to start Cast Receiver', err);
        // Don't reset the global flag on error to prevent retry loops
      }
    };

    // Check availability (CAF is loaded via script tag in index.html)
    if (window.cast?.framework) {
      console.log('[Cast Receiver] CAF framework detected');
      initCast();
    } else {
      console.error('[Cast Receiver] CAF framework not available on window');
      setStatusText('Cast framework not available');
    }

    // Note: We don't cleanup the Cast receiver on unmount because:
    // 1. It's a singleton that should persist for the entire app lifecycle
    // 2. Stopping and restarting it can cause issues with active cast sessions
  }, [setIsPlaying, setPlaybackState]);

  const sseStartedRef = useRef(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const roomId = params.get('roomId');
    if (!roomId) return;
    if (sseStartedRef.current) return;
    sseStartedRef.current = true;

    const casterId = params.get('casterId') || params.get('casterUserId') || '';

    let isMounted = true;
    let unsubscribe: (() => void) | null = null;

    const connect = async () => {
      const [err, stop] = await api.sse(
        '/rooms/{id}/events',
        {
          id: roomId,
        },
        (result: [Error | null, any]) => {
          const [eventError, message] = result;
          if (eventError || !message || !isMounted) {
            return;
          }

          const typedMessage = message as
            | { type: 'connected'; data: unknown }
            | { type: 'playback_update'; data: PlaybackState }
            | { type: 'songs_update'; data: Song[] }
            | { type: 'settings_update'; data: Room }
            | { type: 'users_update'; data: number };

          switch (typedMessage.type) {
            case 'connected':
              setStatusText(`Connected to ${roomId}`);
              break;
            case 'playback_update': {
              const data = typedMessage.data;
              setPlaybackState(
                {
                  ...data,
                  currentSong: data.currentSong
                    ? normalizeSong(data.currentSong)
                    : null,
                },
                roomMode || undefined,
              );
              setIsPlaying(data.isPlaying);
              break;
            }
            case 'songs_update': {
              const data = typedMessage.data;
              setQueue(data);
              break;
            }
            case 'settings_update': {
              const data = typedMessage.data;
              setRoomMode(data.mode);
              setRoomInfo((current) => ({
                name: data.name,
                participantCount: current?.participantCount ?? 0,
              }));
              break;
            }
            case 'users_update': {
              const data = typedMessage.data;
              setRoomInfo((current) => ({
                name: current?.name || roomId,
                participantCount: data,
              }));
              break;
            }
          }
        },
        {
          headers: {
            'X-Cast-Receiver': '1',
            ...(casterId ? { 'X-Cast-Caster-Id': casterId } : {}),
          },
        },
      );

      if (!err && stop) {
        unsubscribe = stop;
        const [songsErr, songs] = await api.get('/rooms/{id}/songs', {
          id: roomId,
        });
        if (!songsErr && songs) {
          setQueue(songs);
        }
      } else if (err) {
        console.error('[Cast Receiver] SSE error', err);
      }
    };

    connect();

    return () => {
      isMounted = false;
      if (unsubscribe) {
        unsubscribe();
      }
      sseStartedRef.current = false;
    };
  }, [normalizeSong, roomMode, setIsPlaying, setPlaybackState]);

  useEffect(() => {
    const interval = setInterval(() => {
      updateActualPosition();
    }, 500);

    return () => clearInterval(interval);
  }, [updateActualPosition]);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      const data = event.data as LocalCastMessage | null;
      if (!data || typeof data !== 'object') return;
      if (!('action' in data)) return;

      console.log('[Local Cast] received message', {
        action: data.action,
      });
      handleCastMessage(data);
    };

    window.addEventListener('message', onMessage);
    return () => {
      window.removeEventListener('message', onMessage);
    };
  }, [handleCastMessage]);

  return (
    <CastContext.Provider
      value={{
        roomInfo,
        queue,
        statusText,
        roomMode,
        currentSong: currentSong
          ? { ...currentSong, position: (currentSong as any).position ?? 0 }
          : null,
        actualPositionMs,
        updateActualPosition,
        debugMode,
      }}
    >
      {children}
    </CastContext.Provider>
  );
};

export const useCast = () => {
  const context = useContext(CastContext);
  if (context === undefined) {
    throw new Error('useCast must be used within a CastProvider');
  }
  return context;
};
