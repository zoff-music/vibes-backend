import { useCallback, useState } from 'react';
import { api } from '../api/client';
import { useRoomStore } from '../stores/roomStore';
import { useSSE } from './useSSE';

export const useRoom = (roomId: string) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const { room, users, userId, setRoom, setSession, reset } = useRoomStore();

  useSSE(roomId);

  const fetchRoom = useCallback(async () => {
    setIsLoading(true);
    const [err, data] = await api.get('/rooms/{id}', { id: roomId });
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
    const [err, data] = await api.post('/rooms/{id}/sessions', 
        { id: roomId },
        { nickname, password }
      );
    setIsLoading(true); // Keep loading state until we handle result

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
    
    setIsLoading(false);
    return null;
  }, [roomId, setRoom, setSession]);

  const leaveRoom = useCallback(() => {
    reset();
  }, [reset]);

  const updateRoom = useCallback(async (updates: any) => {
    setIsLoading(true);
    // updates can contain { settings: {...} } or { mode: '...' } etc.
    const [err, data] = await api.patch('/rooms/{id}/settings', { id: roomId }, updates);
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
  }, [roomId, setRoom]);
  
  // Keep backward compatibility if needed, or just update usages.
  const updateRoomSettings = useCallback((settings: any) => updateRoom({ settings }), [updateRoom]);

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

