import {
  type PlaybackState,
  usePlaybackStore,
  useRoomStore,
} from '@vibez/shared';

import { useCallback, useEffect } from 'react';
import { api, getHttpError } from '../index';
import { USE_SSE_CALLBACKS } from './useSSE';

export function usePlayback(roomId: string, callbacks?: USE_SSE_CALLBACKS) {
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
        // Optimistic update for play/pause in server/host mode
        if (room?.mode && (action === 'play' || action === 'pause')) {
          setLocalPlayingState(action === 'play', room.mode);
        }

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
      if (error) {
        let message = '';

        if (action === 'skip') {
          const msgHost = 'only hosts can skip in host mode';
          const msgDisabled = 'skipping is disabled in this room';

          if (
            error.message?.includes(msgHost) ||
            getHttpError(error)?.response.status === 403
          ) {
            message = 'Only hosts can skip in host mode';
          } else if (error.message?.includes(msgDisabled)) {
            message = 'Skipping is disabled in this room';
          }
        } else if (action === 'play' || action === 'pause') {
          message = `Failed to ${action}: ${error.message || 'Unknown error'}`;
        }

        if (message) {
          if (callbacks?.onToast) {
            callbacks.onToast(message, 'error');
          } else if (
            typeof window !== 'undefined' &&
            window.dispatchEvent &&
            typeof CustomEvent !== 'undefined'
          ) {
            // Web fallback
            window.dispatchEvent(
              new CustomEvent('show-toast', {
                detail: { message, type: 'error' },
              }),
            );
          } else {
            // Mobile/Environment without CustomEvent fallback?
            // Ideally the callback handles it.
            console.error('[usePlayback] Error:', message);
          }
        }
      }

      if (data) {
        // Handle wrapped responses (SkipActionResponse/VoteActionResponse)
        // Handle wrapped responses (SkipActionResponse/VoteActionResponse)
        const state = ((data as { playback?: PlaybackState }).playback ||
          data) as PlaybackState;
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

  const fetchPlayback = useCallback(async () => {
    if (!roomId) return;

    const [err, data] = await api.get('/rooms/{id}/states', { id: roomId });
    if (data) {
      setPlaybackState(data, room?.mode);
    }
    return [err, data];
  }, [roomId, setPlaybackState, room?.mode]);

  return {
    ...playback,
    play,
    pause,
    seek,
    skip,
    vote,
    fetchPlayback,
  };
}
