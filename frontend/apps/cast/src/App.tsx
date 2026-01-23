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

// Global flag to prevent multiple Cast receiver initializations
let isCastReceiverInitialized = false;

const App = () => {
  const [roomInfo] = useState<RoomInfo | null>(null);
  const [statusText] = useState('Ready for Casting');

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

      console.log('Initializing Cast Receiver...');
      const context = cast.framework.CastReceiverContext.getInstance();
      const playerManager = context.getPlayerManager();

      contextRef.current = context;
      playerManagerRef.current = playerManager;

      // --- Message Interceptor for LOAD requests ---
      playerManager.setMessageInterceptor(
        cast.framework.messages.MessageType.LOAD,
        (loadRequestData) => {
          console.log('Intercepted LOAD request:', loadRequestData);

          const media = loadRequestData.media;
          if (media?.customData) {
            // Assuming our sender sends song info in customData
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const data = media.customData as any;

            // Handle tokens if present
            if (data.tokens) {
              for (const [provider, tokenData] of Object.entries(
                data.tokens as Record<string, any>,
              )) {
                if (tokenData?.token) {
                  setCachedToken(
                    provider,
                    tokenData.token,
                    tokenData.expiresAt ||
                      new Date(Date.now() + 3600000).toISOString(),
                  );
                }
              }
            }

            const song = data.song || data; // handle if song is nested or root

            setPlaybackState({
              currentSong: song,
              isPlaying: true,
              positionMs: 0,
              updatedAt: new Date().toISOString(),
              serverTimeMs: Date.now(),
            });
            setIsPlaying(true);
          }

          // Attempt to extract YouTube ID if present in contentId
          const videoId = extractYouTubeVideoId(media.contentId);
          if (videoId) {
            console.log('Detected YouTube detected:', videoId);
          }

          return loadRequestData;
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
      initCast();
    }

    // Note: We don't cleanup the Cast receiver on unmount because:
    // 1. It's a singleton that should persist for the entire app lifecycle
    // 2. Stopping and restarting it can cause issues with active cast sessions
  }, [setIsPlaying, setPlaybackState]);

  // Helper from original code
  const extractYouTubeVideoId = (url?: string) => {
    if (!url) return null;
    const regex =
      /(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([^&\n?#]+)/;
    const match = url.match(regex);
    return match ? match[1] : null;
  };

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
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
