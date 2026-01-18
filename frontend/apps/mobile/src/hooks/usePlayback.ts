import { useCallback, useEffect } from 'react';
import { api } from '../api/client';
import { usePlaybackStore } from '../stores/playbackStore';

export const usePlayback = (roomId: string) => {
  const playback = usePlaybackStore();

  // Update actual position every 100ms for smooth UI progress bars
  useEffect(() => {
    const interval = setInterval(() => {
      playback.updateActualPosition();
    }, 100);
    
    return () => clearInterval(interval);
  }, [playback]);

  const performAction = useCallback(async (action: 'play' | 'pause' | 'skip' | 'vote' | 'seek', positionMs?: number) => {
    let err, data;

    if (action === 'play' || action === 'pause' || action === 'seek') {
      [err, data] = await api.put('/rooms/{id}/states', 
        { id: roomId },
        { action, positionMs }
      );
    } else if (action === 'skip' || action === 'vote') {
      [err, data] = await api.post('/rooms/{id}/skips', { id: roomId }, {});
    }

    if (data) {
      playback.setPlaybackState(data);
    }
  }, [roomId, playback]);

  const play = useCallback(() => performAction('play'), [performAction]);
  const pause = useCallback(() => performAction('pause'), [performAction]);
  const seek = useCallback((positionMs: number) => performAction('seek', positionMs), [performAction]);
  const skip = useCallback(() => performAction('skip'), [performAction]);
  const vote = useCallback(() => performAction('vote'), [performAction]);

  return {
    ...playback,
    play,
    pause,
    seek,
    skip,
    vote,
  };
};

