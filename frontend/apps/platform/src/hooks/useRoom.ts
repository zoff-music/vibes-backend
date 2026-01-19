import { api } from '@vibez/api';
import { useCallback, useState } from 'react';
import { useRoomStore } from '../stores/roomStore';
import { useSSE } from './useSSE';

// Simple request deduplication map to handle strict mode double-invocations
const IN_FLIGHT_REQUESTS = new Map<string, Promise<any>>();

export const useRoom = (roomId: string) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { room, users, userId, setRoom, setSession, reset } = useRoomStore();

  useSSE(roomId);

  const fetchRoom = useCallback(async () => {
    if (!roomId) return;
    
    setIsLoading(true);
    const key = `fetchRoom:${roomId}`;
    
    try {
        let promise = IN_FLIGHT_REQUESTS.get(key);
        if (!promise) {
            promise = api.get('/rooms/{id}', { id: roomId });
            IN_FLIGHT_REQUESTS.set(key, promise);
        }

        const [err, data] = await promise;
        
        // Only clear if it's the same promise we waited for (though simplistic for shared map)
        // Better to clear always after done
        IN_FLIGHT_REQUESTS.delete(key);

        if (err) {
            setError(err);
            return;
        }

        if (data) {
            setRoom(data);
        }
    } catch (e) {
        console.error('Unexpected error fetching room', e);
        IN_FLIGHT_REQUESTS.delete(key);
    } finally {
        setIsLoading(false);
    }
  }, [roomId, setRoom]);

  const joinRoom = useCallback(
    async (nickname?: string, password?: string) => {
      // If we already have a session for this room/user, skip joining
      // This is a basic check; real logic depends on if backend session matches
      const currentUserId = useRoomStore.getState().userId;
      if (currentUserId && useRoomStore.getState().room?.id === roomId) {
          console.log('[Room] Already joined, skipping join request');
          return { userId: currentUserId };
      }

      setIsLoading(true);
      const key = `joinRoom:${roomId}`;

      try {
          let promise = IN_FLIGHT_REQUESTS.get(key);
          if (!promise) {
              promise = api.post(
                '/rooms/{id}/sessions',
                { id: roomId },
                { nickname, password },
              );
              IN_FLIGHT_REQUESTS.set(key, promise);
          }
          
          const [err, data] = await promise;
          IN_FLIGHT_REQUESTS.delete(key);

          if (err) {
            setError(err);
            setIsLoading(false);
            return null;
          }

          if (data) {
            setSession(data.userId, data.isAdmin, data.nickname as any);
            setRoom(data.room);
            setIsLoading(false);
            return data;
          }
      } catch(e) {
           console.error('Unexpected error joining room', e);
           IN_FLIGHT_REQUESTS.delete(key);
      }
      
      setIsLoading(false);
      return null;
    },
    [roomId, setRoom, setSession],
  );

  const leaveRoom = useCallback(() => {
    reset();
  }, [reset]);

  const updateRoom = useCallback(
    async (updates: any) => {
      setIsLoading(true);
      // updates can contain { settings: {...} } or { mode: '...' } etc.
      const [err, data] = await api.patch(
        '/rooms/{id}/settings',
        { id: roomId },
        updates,
      );
      setIsLoading(false);

      if (err) {
        setError(err);
        return null;
      }

      if (data) {
        setRoom(data);
        return data;
      }
      return null;
    },
    [roomId, setRoom],
  );

  // Keep backward compatibility if needed, or just update usages.
  const updateRoomSettings = useCallback(
    (settings: any) => updateRoom({ settings }),
    [updateRoom],
  );

  return {
    room,
    users,
    userId,
    isLoading,
    error,
    fetchRoom,
    joinRoom,
    updateRoomSettings,
    updateRoom,
    leaveRoom,
  };
};
