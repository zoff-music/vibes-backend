import { api, setCachedToken } from '@vibez/api';
import { type Song, safeWrap, usePlaybackStore } from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import { setGlobalDebug } from '../logging';

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
    setGlobalDebug(value);
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

  // Helper to normalize song which might come from untyped sources
  const normalizeSong = useCallback((song: Song): Song => {
    return {
      id: song.id,
      sourceType: song.sourceType,
      sourceId: song.sourceId,
      title: song.title,
      artist: song.artist,
      thumbnailUrl: song.thumbnailUrl || '',
      duration: song.duration || 0,
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
      const currentState = usePlaybackStore.getState();
      const currentSongId = currentState.currentSong?.id;
      const currentActualPosition = currentState.actualPositionMs;

      switch (action) {
        case 'updatePlayback':
          console.log('Updating playback state:', message);
          if (message.currentSong) {
            const normalizedSong = normalizeSong(message.currentSong);
            const isSameSong = currentSongId === normalizedSong.id;
            // Prevent reset to 0 if we are already playing this song and have a position
            const shouldPreservePosition =
              isSameSong &&
              (!message.positionMs || message.positionMs === 0) &&
              currentActualPosition > 1000;
            const positionMs = shouldPreservePosition
              ? currentActualPosition
              : message.positionMs || 0;

            if (shouldPreservePosition) {
              console.log('[Cast Receiver] Preserving local position', {
                local: currentActualPosition,
                incoming: message.positionMs,
              });
            }

            setPlaybackState(
              {
                currentSong: normalizedSong,
                isPlaying: message.isPlaying || false,
                positionMs: positionMs,
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
            const isSameSong = currentSongId === normalizedSong.id;
            // Prevent reset to 0 if we are already playing this song and have a position
            const shouldPreservePosition =
              isSameSong &&
              (!message.positionMs || message.positionMs === 0) &&
              currentActualPosition > 1000;
            const positionMs = shouldPreservePosition
              ? currentActualPosition
              : message.positionMs || 0;

            if (shouldPreservePosition) {
              console.log('[Cast Receiver] Preserving local position', {
                local: currentActualPosition,
                incoming: message.positionMs,
              });
            }

            setPlaybackState(
              {
                currentSong: normalizedSong,
                isPlaying: message.isPlaying || false,
                positionMs: positionMs,
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
        debug: debugMode,
      });

      const context = cast.framework.CastReceiverContext.getInstance();

      // ENABLE DEBUG LOGGING
      if (debugMode) {
        context.setLoggerLevel(cast.framework.LoggerLevel.DEBUG);
      }

      const playerManager = context.getPlayerManager();

      contextRef.current = context;
      playerManagerRef.current = playerManager;

      // REMOVED noisy listeners that crash inspector on low-power devices
      /* 
      playerManager.addEventListener(cast.framework.events.EventType.MEDIA_STATUS, ...)
      playerManager.addEventListener(cast.framework.events.EventType.TIME_UPDATE, ...)
      */

      playerManager.addEventListener(
        cast.framework.events.EventType.ERROR,
        (event: framework.events.ErrorEvent) => {
          console.error('[Cast Receiver] PLAYER_ERROR', {
            detailedErrorCode: event.detailedErrorCode,
            error: event.error,
          });
        },
      );

      // --- Message Interceptor for LOAD requests ---
      playerManager.setMessageInterceptor(
        cast.framework.messages.MessageType.LOAD,
        (loadRequestData) => {
          console.log('Intercepted LOAD request:', loadRequestData);
          // Return the request data as-is if it's our technical URL
          // but if it's a "real" media request from the sender, we handle it
          // Returning null CAN cause session_error on some senders,
          // but it's the official way to tell CAF "I got this, don't play anything".

          const media = loadRequestData.media;
          if (media?.customData) {
            const data = media.customData as any;
            if (data.tokens) {
              for (const [provider, tokenData] of Object.entries(data.tokens)) {
                if ((tokenData as any)?.token) {
                  setCachedToken(
                    provider,
                    (tokenData as any).token,
                    (tokenData as any).expiresAt ||
                      new Date(Date.now() + 3600000).toISOString(),
                  );
                }
              }
            }

            if (data.debug) setDebugMode(true);

            if (data.currentSong) {
              const normalizedSong = normalizeSong(data.currentSong);
              setPlaybackState({
                currentSong: normalizedSong,
                isPlaying: true,
                positionMs: data.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              });
              setIsPlaying(true);
              setStatusText(`Now Playing: ${normalizedSong.title}`);
              updateMediaMetadata(normalizedSong);
            }
          }

          return null as any;
        },
      );

      // --- Custom Message Handler for Vibez messages ---
      context.addCustomMessageListener(
        'urn:x-cast:com.vibez.cast',
        (customEvent) => {
          const message = customEvent.data as LocalCastMessage;
          handleCastMessage(message);
        },
      );

      const options = new cast.framework.CastReceiverOptions();
      options.maxInactivity = 3600;
      options.statusText = 'Zoff';
      options.disableIdleTimeout = true;

      const [startErr] = safeWrap(() => {
        context.start(options);
        isCastReceiverInitialized = true;
      });

      if (startErr) {
        console.error('[Cast Receiver] Start error', startErr);
      }
    };

    if (window.cast?.framework) {
      initCast();
    }
  }, [setIsPlaying, setPlaybackState]);

  const sseStartedRef = useRef(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const roomId = params.get('roomId');
    if (!roomId || sseStartedRef.current) return;

    sseStartedRef.current = true;
    const casterId = params.get('casterId') || params.get('casterUserId') || '';

    let isMounted = true;
    let unsubscribe: (() => void) | null = null;

    const connect = async () => {
      console.log('[Cast Receiver] Connect sequence started', {
        roomId,
        casterId,
        debug: debugMode,
      });

      if (!roomId) {
        console.error(
          '[Cast Receiver] Room ID is missing from URL parameters!',
        );
        return;
      }

      // Fetch initial state immediately, don't wait for SSE
      console.log(`[Cast Receiver] Fetching queue & playback from: ${roomId}`);

      Promise.all([
        api.get('/rooms/{id}/songs', { id: roomId }),
        api.get('/rooms/{id}/states', { id: roomId }),
      ])
        .then(([queueRes, playbackRes]) => {
          const [songsErr, songs] = queueRes;
          const [playbackErr, playbackState] = playbackRes;

          console.log('[Cast Receiver] Initial fetch results:', {
            queueErr: songsErr ? songsErr.message : null,
            queueCount: songs?.length,
            playbackErr: playbackErr ? playbackErr.message : null,
            hasPlayback: !!playbackState,
          });

          if (!songsErr && songs) setQueue(songs);

          if (!playbackErr && playbackState && playbackState.currentSong) {
            const normalizedSong = normalizeSong(playbackState.currentSong);
            setPlaybackState(
              {
                ...playbackState,
                currentSong: normalizedSong,
              },
              roomMode || undefined,
            );
            setIsPlaying(playbackState.isPlaying);
            setStatusText(`Now Playing: ${normalizedSong.title}`);
            updateMediaMetadata(normalizedSong);
          }
        })
        .catch((unexpectedErr) => {
          console.error(
            '[Cast Receiver] Unexpected fetch error:',
            unexpectedErr,
          );
        });

      const [err, stop] = await api.sse(
        '/rooms/{id}/events',
        { id: roomId },
        (result: [Error | null, any]) => {
          const [eventError, message] = result;
          if (eventError || !message || !isMounted) return;

          const typedMessage = message as any;

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
                // Use a ref or get the value directly to avoid effect restart
                undefined,
              );
              setIsPlaying(data.isPlaying);
              break;
            }
            case 'songs_update':
              console.log(
                '[Cast Receiver] Received songs update (SSE):',
                typedMessage.data,
              );
              setQueue(typedMessage.data);
              break;
            case 'settings_update':
              setRoomMode(typedMessage.data.mode);
              setRoomInfo((current) => ({
                name: typedMessage.data.name,
                participantCount: current?.participantCount ?? 0,
              }));
              break;
            case 'users_update':
              setRoomInfo((current) => ({
                name: current?.name || roomId,
                participantCount: typedMessage.data,
              }));
              break;
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
      }
    };

    connect();

    return () => {
      isMounted = false;
      if (unsubscribe) unsubscribe();
      // WE DO NOT reset sseStartedRef.current here to avoid infinite loops on re-renders
    };
  }, [normalizeSong, setIsPlaying, setPlaybackState]);

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
        currentSong,
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
