import { api } from '@vibez/api';
import { usePlaybackStore } from '@vibez/shared';
import { useCallback, useEffect } from 'react';
import { useRoomStore } from '../stores/roomStore';

export const usePlayback = (roomId: string) => {
  const playback = usePlaybackStore();
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setLocalPlayingState = usePlaybackStore((state) => state.setLocalPlayingState);
  const room = useRoomStore((state) => state.room);

  // Update actual position every 100ms for smooth UI progress bars
  useEffect(() => {
    const interval = setInterval(() => {
      playback.updateActualPosition();
    }, 100);

    return () => clearInterval(interval);
  }, [playback]);

  const performAction = useCallback(
    async (
      action: 'play' | 'pause' | 'skip' | 'vote' | 'seek',
      positionMs?: number,
    ) => {
      let data;
      let error;

      if (action === 'play' || action === 'pause' || action === 'seek') {
        const [err, result] = await api.put(
          '/rooms/{id}/states',
          { id: roomId },
          { action, positionMs },
        );
        data = result;
        error = err;
        
        // In Server mode, handle local playing state
        if (data && room?.mode === 'server' && (action === 'play' || action === 'pause')) {
          setLocalPlayingState(data.isPlaying, room.mode);
        }
      } else if (action === 'skip' || action === 'vote') {
        const [err, result] = await api.post(
          '/rooms/{id}/skips',
          { id: roomId },
          {},
        );
        data = result;
        error = err;
      }

      // Handle host mode skip error
      if (error && action === 'skip') {
        // Check if it's a 403 Forbidden error (host mode restriction or skip disabled)
        if (error.message?.includes('only hosts can skip in host mode') || 
            (error as any)?.status === 403) {
          // Dispatch custom event for toast notification
          window.dispatchEvent(new CustomEvent('show-toast', {
            detail: {
              message: 'Only hosts can skip in host mode',
              type: 'error'
            }
          }));
          return;
        }
        
        if (error.message?.includes('skipping is disabled in this room')) {
          // Dispatch custom event for toast notification
          window.dispatchEvent(new CustomEvent('show-toast', {
            detail: {
              message: 'Skipping is disabled in this room',
              type: 'error'
            }
          }));
          return;
        }
      }

      if (data) {
        setPlaybackState(data, room?.mode);
      }
    },
    [roomId, setPlaybackState, setLocalPlayingState, room?.mode],
  );

  const play = useCallback(() => performAction('play'), [performAction]);
  const pause = useCallback(() => performAction('pause'), [performAction]);
  const seek = useCallback(
    (positionMs: number) => performAction('seek', positionMs),
    [performAction],
  );
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
