import { api } from '@vibez/api';
import { useCallback, useEffect } from 'react';
import { usePlaybackStore } from '../stores/playbackStore';
import { useRoomStore } from '../stores/roomStore';
import { USE_SSE_CALLBACKS } from './useSSE';

export const usePlayback = (roomId: string, callbacks?: USE_SSE_CALLBACKS) => {
  const playback = usePlaybackStore();
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setLocalPlayingState = usePlaybackStore(
    (state) => state.setLocalPlayingState,
  );
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

        if (
          data &&
          room?.mode === 'server' &&
          (action === 'play' || action === 'pause')
        ) {
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
        const msgHost = 'only hosts can skip in host mode';
        const msgDisabled = 'skipping is disabled in this room';

        let message = '';
        if (
          error.message?.includes(msgHost) ||
          (error as any)?.status === 403
        ) {
          message = 'Only hosts can skip in host mode';
        } else if (error.message?.includes(msgDisabled)) {
          message = 'Skipping is disabled in this room';
        }

        if (message) {
          if (callbacks?.onToast) {
            callbacks.onToast(message, 'error');
          } else if (typeof window !== 'undefined' && window.dispatchEvent) {
            window.dispatchEvent(
              new CustomEvent('show-toast', {
                detail: { message, type: 'error' },
              }),
            );
          }
        }
      }

      if (data) {
        // Handle wrapped responses (SkipActionResponse/VoteActionResponse)
        const state = (data as any).playback || data;
        setPlaybackState(state, room?.mode);
      }
    },
    [roomId, setPlaybackState, setLocalPlayingState, room?.mode, callbacks],
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
