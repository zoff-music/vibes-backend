import { useCallback, useRef } from 'react';
import { api } from '../api/client';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';
import { usePlaybackStore } from '../stores/playbackStore';
import { SourceType } from '@vibez/shared';

export const useQueue = (roomId: string) => {
  const { songs, setSongs, addSong, removeSong, reorderSongs } = useQueueStore();
  const { userId } = useRoomStore();
  const { setPlaybackState } = usePlaybackStore();
  const lastAddTimestamp = useRef<number>(0);

  const fetchQueue = useCallback(async () => {
    const fetchStartTime = Date.now();
    const [_err, data] = await api.get('/rooms/{id}/songs', { id: roomId });
    
    // If an add operation happened after we started fetching, 
    // this fetch result is stale and should be ignored to prevent 
    // overwriting the optimistic update.
    if (lastAddTimestamp.current > fetchStartTime) {
      console.log('[Queue] Ignoring stale fetch queue result');
      return;
    }

    if (data) {
      setSongs(data);
    }
  }, [roomId, setSongs]);

  const addToQueue = useCallback(async (
    sourceType: SourceType,
    sourceId: string,
    title: string,
    thumbnailUrl: string,
    duration: number,
    artist?: string
  ) => {
    if (!userId) return null;

    const timestamp = Date.now();
    lastAddTimestamp.current = timestamp;

    const [_err, data] = await api.post('/rooms/{id}/songs', 
      { id: roomId },
      {
        sourceType,
        sourceId,
        title,
        thumbnailUrl,
        duration,
        artist,
        addedBy: userId,
      }
    );

    if (data) {
      const shouldAutoPlay = songs.length === 0 && !usePlaybackStore.getState().currentSongId;
      addSong(data);

      if (shouldAutoPlay) {
        const [playErr, playback] = await api.post('/rooms/{id}/action', 
          { id: roomId },
          { action: 'play' }
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
  }, [roomId, userId, addSong]);

  const removeFromQueue = useCallback(async (songId: string) => {
    // Optimistic update
    removeSong(songId);

    const [err, _] = await api.delete('/rooms/{id}/songs/{songId}', { id: roomId, songId });

    if (err) {
      // Rollback or re-fetch on error
      fetchQueue();
    }
  }, [roomId, removeSong, fetchQueue]);

  const moveInQueue = useCallback(async (songId: string, newPosition: number) => {
    // Optimistic update
    reorderSongs(songId, newPosition);

    const [err, _] = await api.patch('/rooms/{id}/songs/reorder/{songId}', 
      { id: roomId, songId },
      { newPosition }
    );

    if (err) {
      // Rollback or re-fetch on error
      fetchQueue();
    }
  }, [roomId, reorderSongs, fetchQueue]);

  return {
    songs,
    fetchQueue,
    addToQueue,
    removeFromQueue,
    moveInQueue,
  };
};

