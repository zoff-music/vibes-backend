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
}

const CastContext = createContext<CastContextType | undefined>(undefined);

export const CastProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [roomInfo, setRoomInfo] = useState<RoomInfo | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [statusText, setStatusText] = useState('Ready for Casting');
  const [roomMode, setRoomMode] = useState<string | null>(null);

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
            setPlaybackState(
              {
                currentSong: normalizeSong(message.currentSong),
                isPlaying: message.isPlaying || false,
                positionMs: message.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              },
              roomMode || undefined,
            );
            setIsPlaying(message.isPlaying || false);
            setStatusText(`Now Playing: ${message.currentSong.title}`);
          }
          if (message.roomInfo) {
            setRoomInfo(message.roomInfo);
          }
          break;

        case 'syncPlayback':
          console.log('Syncing playback:', message);
          if (message.currentSong) {
            setPlaybackState(
              {
                currentSong: normalizeSong(message.currentSong),
                isPlaying: message.isPlaying || false,
                positionMs: message.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              },
              roomMode || undefined,
            );
            setIsPlaying(message.isPlaying || false);
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
    [normalizeSong, roomMode, setIsPlaying, setPlaybackState],
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
            // Handle tokens if present
            const data = media.customData as any;
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
          }

          setStatusText('Waiting for content...');
          return loadRequestData;
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

      const [err] = safeWrap(() => {
        context.start(options);
        isCastReceiverInitialized = true;
        console.log('Cast Receiver started');
      });
      if (err) {
        console.error('Failed to start Cast Receiver', err);
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
