/// <reference types="chromecast-caf-receiver" />

import { SoundCloudPlayer, SpotifyPlayer, VideoPlayer } from '@vibez/player';
import { usePlaybackStore } from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import { useEffect, useRef, useState } from 'react';

// Types are available globally via @types/chromecast-caf-receiver

interface RoomInfo {
  name: string;
  participantCount: number;
}

// QueueItem removed as unused

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
            const song = media.customData as any;
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

      try {
        context.start(options);
        console.log('Cast Receiver started');
      } catch (e) {
        console.error('Failed to start Cast Receiver', e);
      }
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
    <div className="relative flex h-screen w-screen flex-col items-center justify-center overflow-hidden bg-gradient-to-br from-indigo-900 to-purple-900 text-white">
      {/* Main Content Area */}
      <div className="relative flex h-full w-full items-center justify-center">
        {/* Render Players based on currentSong type */}
        {/* They check usePlaybackStore internally */}

        <div className="absolute inset-0 h-full w-full">
          <VideoPlayer isVisible={currentSong?.sourceType === 'youtube'} />
          <SpotifyPlayer isVisible={currentSong?.sourceType === 'spotify'} />
          <SoundCloudPlayer
            isVisible={currentSong?.sourceType === 'soundcloud'}
          />
        </div>

        {/* Fallback / Idle Screen */}
        {!currentSong && (
          <div className="z-10 text-center">
            <h1 className="mb-4 font-black text-6xl">Vibez</h1>
            <p className="text-2xl opacity-70">{statusText}</p>
            {roomInfo && (
              <div className="mt-8 rounded-xl bg-white/10 p-6 backdrop-blur-md">
                <p className="font-bold text-xl">{roomInfo.name}</p>
                <p className="opacity-60">{roomInfo.participantCount} active</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
