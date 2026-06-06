import {
  type SourceType,
  usePlaybackStore,
  useQueueStore,
} from '@vibez/shared';
import { useCallback, useRef } from 'react';
import { api } from '../index';

export const useQueue = (roomId: string) => {
  const songs = useQueueStore((state) => state.songs);
  const setSongs = useQueueStore((state) => state.setSongs);
  const addSong = useQueueStore((state) => state.addSong);
  const removeSong = useQueueStore((state) => state.removeSong);
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const lastAddTimestamp = useRef<number>(0);

  const fetchQueue = useCallback(async () => {
    const fetchStartTime = Date.now();
    const [_err, data] = await api.get('/rooms/{id}/songs', { id: roomId });

    if (lastAddTimestamp.current > fetchStartTime) {
      console.log('[Queue] Igzoffng stale fetch queue result');
      return;
    }

    if (data) {
      setSongs(data);
    }
  }, [roomId, setSongs]);

  const addToQueue = useCallback(
    async (
      sourceType: SourceType,
      sourceId: string,
      title: string,
      thumbnailUrl: string,
      duration: number,
      artist?: string,
    ) => {
      const timestamp = Date.now();
      lastAddTimestamp.current = timestamp;

      const [_err, data] = await api.post(
        '/rooms/{id}/songs',
        { id: roomId },
        {
          sourceType,
          sourceId,
          title,
          thumbnailUrl,
          duration,
          artist,
        },
      );

      if (data) {
        const shouldAutoPlay =
          songs.length === 0 && !usePlaybackStore.getState().currentSong;
        addSong(data);

        if (shouldAutoPlay) {
          const [playErr, playback] = await api.post(
            '/rooms/{id}',
            { id: roomId },
            { action: 'play' },
          );

          if (playErr) {
            console.error('[Queue] Failed to auto-play after add:', playErr);
          }

          if (playback) {
            setPlaybackState(playback);
          }
        }

        return data;
      }

      return null;
    },
    [roomId, addSong, songs.length, setPlaybackState],
  );

  const removeFromQueue = useCallback(
    async (songId: string) => {
      removeSong(songId);

      const [err, _] = await api.delete('/rooms/{id}/songs/{songId}', {
        id: roomId,
        songId,
      });

      if (err) {
        fetchQueue();
      }
    },
    [roomId, removeSong, fetchQueue],
  );

  const voteSong = useCallback(
    async (songId: string) => {
      const [err, _] = await api.post(
        '/rooms/{id}/songs/{songId}',
        { id: roomId, songId },
        {},
      );

      if (err) {
        if (
          err.message.includes('409') ||
          err.message.includes('already voted')
        ) {
          return 'already_voted' as const;
        }
        console.error('[Queue] Failed to vote for song:', err);
        return 'error' as const;
      }
      return 'success' as const;
    },
    [roomId],
  );

  return {
    songs,
    fetchQueue,
    addToQueue,
    removeFromQueue,
    voteSong,
  };
};
