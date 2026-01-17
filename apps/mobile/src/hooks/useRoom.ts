import { useCallback, useState } from 'react';
import { api } from '../api/client';
import { useRoomStore } from '../stores/roomStore';
import { useSSE } from './useSSE';
import { safeWrapAsync } from '../utils/wrap';

export const useRoom = (roomId: string) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { room, users, setRoom, setSession, reset } = useRoomStore();

  useSSE(roomId);

  const fetchRoom = useCallback(async () => {
    setIsLoading(true);
    const [data, err] = await safeWrapAsync(api.get(`/rooms/{id}`, { params: { id: roomId } }));
    setIsLoading(false);

    if (err) {
      setError(err);
      return;
    }

    if (data) {
      setRoom(data);
    }
  }, [roomId, setRoom]);

  const joinRoom = useCallback(async (nickname?: string, password?: string) => {
    setIsLoading(true);
    const [data, err] = await safeWrapAsync(
      api.post(`/rooms/{id}/sessions`, { 
        params: { id: roomId },
        request: { nickname, password }
      })
    );
    setIsLoading(true); // Keep loading state until we handle result

    if (err) {
      setError(err);
      setIsLoading(false);
      return null;
    }

    if (data) {
      setSession(data.userId, data.isAdmin, data.nickname);
      setRoom(data.room);
      setIsLoading(false);
      return data;
    }
    
    setIsLoading(false);
    return null;
  }, [roomId, setRoom, setSession]);

  const leaveRoom = useCallback(() => {
    reset();
  }, [reset]);

  return {
    room,
    users,
    isLoading,
    error,
    fetchRoom,
    joinRoom,
    leaveRoom,
  };
};
