import { SoundCloudPlayer, SpotifyPlayer, VideoPlayer } from '@vibez/ui';
import React from 'react';
import { useCast } from './CastProvider';

export const PlayerLayer: React.FC = () => {
  const { currentSong } = useCast();

  return (
    <div className="absolute inset-0 h-full w-full">
      <VideoPlayer
        isVisible={currentSong?.sourceType === 'youtube'}
        fill
        appContext="cast"
      />
      <SpotifyPlayer isVisible={currentSong?.sourceType === 'spotify'} fill />
      <SoundCloudPlayer
        isVisible={currentSong?.sourceType === 'soundcloud'}
        fill
      />
    </div>
  );
};
