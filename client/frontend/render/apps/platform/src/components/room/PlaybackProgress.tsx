import { usePlaybackPosition } from '@vibez/api';
import React from 'react';

interface PlaybackProgressProps {
  durationMs: number;
  isSSR: boolean;
}

export const PlaybackProgress: React.FC<PlaybackProgressProps> = React.memo(
  ({ durationMs, isSSR }) => {
    const actualPositionMs = usePlaybackPosition();

    const progress = durationMs > 0 ? actualPositionMs / durationMs : 0;

    const formatTime = (ms: number) => {
      const seconds = Math.floor(ms / 1000);
      const m = Math.floor(seconds / 60);
      const s = seconds % 60;
      return `${m}:${s.toString().padStart(2, '0')}`;
    };

    return (
      <div className="mt-3">
        <progress
          className="progress-bar h-1 w-full"
          value={isSSR ? 0 : Math.min(progress, 1)}
          max={1}
        />
        <div className="mt-1 flex justify-between font-mono text-[10px] text-theme-subtle">
          <span>{formatTime(actualPositionMs)}</span>
          <span>{formatTime(durationMs)}</span>
        </div>
      </div>
    );
  },
);
