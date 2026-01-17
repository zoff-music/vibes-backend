import { useCallback } from 'react';
import { api } from '../api/client';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';
import { safeWrapAsync } from '../utils/wrap';
import { SourceType } from '@vibez/shared';

export const useQueue = (roomId: string) => {
  const { songs, setSongs, addSong, removeSong, reorderSongs } = useQueueStore();
  const { userId } = useRoomStore();

  const fetchQueue = useCallback(async () => {
    const [data, err] = await safeWrapAsync(api.get(`/rooms/{id}/songs`, { 
      params: { id: roomId } 
    }));
    
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

    const [data, err] = await safeWrapAsync(
      api.post(`/rooms/{id}/songs`, {
        params: { id: roomId },
        request: {
          sourceType,
          sourceId,
          title,
          thumbnailUrl,
          duration,
          artist,
          addedBy: userId,
        }
      })
    );

    if (data) {
      addSong(data);
      return data;
    }
    
    return null;
  }, [roomId, userId, addSong]);

  const removeFromQueue = useCallback(async (songId: string) => {
    // Optimistic update
    removeSong(songId);

    const [_, err] = await safeWrapAsync(
      api.delete(`/rooms/{id}/songs/{songId}`, {
        params: { id: roomId, songId }
      })
    );

    if (err) {
      // Rollback or re-fetch on error
      fetchQueue();
    }
  }, [roomId, removeSong, fetchQueue]);

  const moveInQueue = useCallback(async (songId: string, newPosition: number) => {
    // Optimistic update
    reorderSongs(songId, newPosition);

    const [_, err] = await safeWrapAsync(
      api.patch(`/rooms/{id}/songs/reorder/{songId}`, {
        params: { id: roomId, songId },
        request: { newPosition }
      })
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
