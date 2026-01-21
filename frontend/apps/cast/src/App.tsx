/// <reference types="chromecast-caf-receiver" />

import { SoundCloudPlayer, SpotifyPlayer, VideoPlayer } from '@vibez/player';
import { safeWrap, setCachedToken, usePlaybackStore } from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import { useEffect, useRef, useState } from 'react';

// Types are available globally via @types/chromecast-caf-receiver

interface RoomInfo {
  name: string;
  participantCount: number;
}

interface AppProps {
  initialData?: any;
}

const App = ({ initialData: _ }: AppProps) => {
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
              // Add required fields with defaults
              currentSongId: song.id || null,
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
        console.log('Cast Receiver started');
      });
      if (err) console.error('Failed to start Cast Receiver', err);
    };

    // Check availability (CAF is loaded via script tag in index.html)
    if (window.cast?.framework) {
      initCast();
    }
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
    <div className="dark relative flex h-screen w-screen animate-fade-in flex-col items-center justify-center overflow-hidden bg-theme text-theme">
      {/* Dynamic Background Elements */}
      <div className="pointer-events-none absolute inset-0 overflow-hidden">
        <div className="absolute top-[-10%] left-[-10%] h-[40%] w-[40%] animate-float rounded-full bg-primary/10 blur-[120px]" />
        <div className="absolute right-[-5%] bottom-[-5%] h-[50%] w-[50%] animate-float-delayed rounded-full bg-primary/5 blur-[100px]" />
      </div>

      {/* Main Content Area */}
      <div className="relative z-10 flex h-full w-full items-center justify-center">
        {/* Render Players based on currentSong type */}
        <div className="absolute inset-0 h-full w-full">
          <VideoPlayer isVisible={currentSong?.sourceType === 'youtube'} />
          <SpotifyPlayer isVisible={currentSong?.sourceType === 'spotify'} />
          <SoundCloudPlayer
            isVisible={currentSong?.sourceType === 'soundcloud'}
          />
        </div>

        {/* Fallback / Idle Screen */}
        {!currentSong && (
          <div className="animate-scale-in text-center">
            <h1
              className="mb-6 font-black text-8xl text-primary tracking-tight drop-shadow-neon-pink"
              style={{ fontFamily: 'Poppins, sans-serif' }}
            >
              Vibez
            </h1>
            <div className="glass mx-auto max-w-sm rounded-3xl px-8 py-4">
              <p className="font-bold text-2xl tracking-wide opacity-90">
                {statusText}
              </p>
              <p className="mt-2 font-medium opacity-50">
                Waiting for music to play...
              </p>
            </div>

            {roomInfo && (
              <div className="glass-elevated mt-10 animate-slide-up rounded-3xl p-8">
                <p className="font-black text-3xl tracking-tight">
                  {roomInfo.name}
                </p>
                <div className="mt-3 flex items-center justify-center gap-2">
                  <div className="h-2 w-2 animate-pulse rounded-full bg-matcha" />
                  <p className="font-bold text-lg text-matcha">
                    {roomInfo.participantCount} active
                  </p>
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
