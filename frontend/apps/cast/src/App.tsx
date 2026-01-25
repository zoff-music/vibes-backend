/// <reference types="chromecast-caf-receiver" />

import { safeWrap, setCachedToken, usePlaybackStore } from '@vibez/shared';
import { SoundCloudPlayer, SpotifyPlayer, VideoPlayer } from '@vibez/ui';
import type { framework } from 'chromecast-caf-receiver';
import { useEffect, useRef, useState } from 'react';

// Types are available globally via @types/chromecast-caf-receiver

interface RoomInfo {
  name: string;
  participantCount: number;
}

interface QueueItem {
  id: string;
  title: string;
  artist: string;
  sourceType: string;
  sourceId: string;
  thumbnailUrl?: string;
  duration?: number;
}

// Global flag to prevent multiple Cast receiver initializations
let isCastReceiverInitialized = false;

const App = () => {
  const [roomInfo, setRoomInfo] = useState<RoomInfo | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [statusText, setStatusText] = useState('Ready for Casting');

  // Use global store for playback state to share with components
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setIsPlaying = usePlaybackStore((state) => state.setIsPlaying);
  const currentSong = usePlaybackStore((state) => state.currentSong);

  // Cast Context Ref
  const contextRef = useRef<framework.CastReceiverContext | null>(null);
  const playerManagerRef = useRef<framework.PlayerManager | null>(null);

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

          const message = customEvent.data;
          console.log('[Cast Receiver] custom message payload', {
            action: message?.action,
            hasCurrentSong: !!message?.currentSong,
            hasQueue: Array.isArray(message?.queue),
            hasRoomInfo: !!message?.roomInfo,
          });

          switch (message.action) {
            case 'updatePlayback':
              console.log('Updating playback state:', message);
              if (message.currentSong) {
                setPlaybackState({
                  currentSong: message.currentSong,
                  isPlaying: message.isPlaying || false,
                  positionMs: message.positionMs || 0,
                  updatedAt: new Date().toISOString(),
                  serverTimeMs: Date.now(),
                });
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
                setPlaybackState({
                  currentSong: message.currentSong,
                  isPlaying: message.isPlaying || false,
                  positionMs: message.positionMs || 0,
                  updatedAt: new Date().toISOString(),
                  serverTimeMs: Date.now(),
                });
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
              console.log('Unknown message action:', message.action);
          }
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

  return (
    <div className="relative flex min-h-screen w-screen animate-fade-in items-center justify-center overflow-hidden bg-theme text-theme">
      <div className="synth-sky absolute inset-0" />
      <div className="vhs-scanlines pointer-events-none absolute inset-0" />
      <div className="sun-hero opacity-80" />
      <div className="retro-grid opacity-70" />

      <div className="relative z-10 flex h-full w-full items-center justify-center">
        <div className="absolute inset-0 h-full w-full">
          <VideoPlayer isVisible={currentSong?.sourceType === 'youtube'} />
          <SpotifyPlayer isVisible={currentSong?.sourceType === 'spotify'} />
          <SoundCloudPlayer
            isVisible={currentSong?.sourceType === 'soundcloud'}
          />
        </div>

        {!currentSong && (
          <div className="relative z-10 flex max-w-3xl flex-col items-center gap-8 px-6 text-center">
            <div className="panel-frame panel-surface w-full px-10 py-12">
              <div className="mb-6 flex flex-col items-center gap-3">
                <span
                  className="vhs-tear-strong glow-text font-display text-6xl text-readable text-theme uppercase tracking-[0.22em] md:text-7xl"
                  data-text="ノリ"
                >
                  ノリ
                </span>
                <p className="font-mono text-theme-subtle text-xs lowercase tracking-[0.4em]">
                  nori
                </p>
              </div>
              <p className="font-display text-2xl text-readable text-theme">
                {statusText}
              </p>
              <p className="mt-3 text-base text-theme-muted">
                Waiting for music to play...
              </p>
            </div>

            {roomInfo && (
              <div className="panel-frame panel-surface w-full px-8 py-6">
                <p className="font-display text-3xl text-theme">
                  {roomInfo.name}
                </p>
                <div className="mt-3 flex items-center justify-center gap-2 text-theme-muted">
                  <span className="h-2 w-2 animate-pulse rounded-full bg-[var(--color-secondary)]" />
                  <span className="text-sm uppercase tracking-[0.25em]">
                    {roomInfo.participantCount} active
                  </span>
                </div>
              </div>
            )}

            {queue.length > 0 && (
              <div className="panel-frame panel-surface w-full px-8 py-6">
                <h3 className="mb-4 font-display text-theme text-xl uppercase tracking-[0.15em]">
                  Up Next
                </h3>
                <div className="space-y-3">
                  {queue.slice(0, 3).map((song, index) => (
                    <div
                      key={song.id}
                      className="flex items-center gap-4 rounded-lg bg-black bg-opacity-20 p-3 backdrop-blur-sm dark:bg-white dark:bg-opacity-10"
                    >
                      {song.thumbnailUrl && (
                        <img
                          src={song.thumbnailUrl}
                          alt={song.title}
                          className="h-12 w-12 rounded-md object-cover"
                        />
                      )}
                      <div className="flex-1 text-left">
                        <p className="truncate font-medium text-sm text-theme">
                          {song.title}
                        </p>
                        <p className="truncate text-theme-muted text-xs">
                          {song.artist}
                        </p>
                      </div>
                      <div className="text-theme-subtle text-xs">
                        #{index + 1}
                      </div>
                    </div>
                  ))}
                  {queue.length > 3 && (
                    <div className="text-center text-sm text-theme-muted">
                      +{queue.length - 3} more songs
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
