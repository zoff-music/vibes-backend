import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';
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

    const cachedPromise = IN_FLIGHT_REQUESTS.get(key) as Promise<[Error | null, any]> | undefined;
    let promise: Promise<[Error | null, any]>;
    if (!cachedPromise) {
      promise = api.get('/rooms/{id}', { id: roomId });
      IN_FLIGHT_REQUESTS.set(key, promise);
    } else {
      promise = cachedPromise;
    }

    const [wrapErr, result] = await safeWrapAsync(promise);
    IN_FLIGHT_REQUESTS.delete(key);
    setIsLoading(false);

    if (wrapErr) {
      console.error('Unexpected error fetching room', wrapErr);
      setError(wrapErr);
      return;
    }

    const [err, data] = result as [Error | null, any];
    if (err) {
      setError(err);
      return;
    }

    if (data) setRoom(data);
  }, [roomId, setRoom]);

  const joinRoom = useCallback(
    async (password?: string) => {
      setIsLoading(true);
      const key = `joinRoom:${roomId}`;

      const cachedPromise = IN_FLIGHT_REQUESTS.get(key) as Promise<[Error | null, any]> | undefined;
      let promise: Promise<[Error | null, any]>;
      if (!cachedPromise) {
        promise = api.post(
          '/rooms/{id}/sessions',
          { id: roomId },
          { password },
        );
        IN_FLIGHT_REQUESTS.set(key, promise);
      } else {
        promise = cachedPromise;
      }

      const [wrapErr, result] = await safeWrapAsync(promise);
      IN_FLIGHT_REQUESTS.delete(key);

      if (wrapErr) {
        console.error('Unexpected error joining room', wrapErr);
        setIsLoading(false);
        setError(wrapErr);
        return null;
      }

      const [err, data] = result as [Error | null, any];
      setIsLoading(false);

      if (err) {
        setError(err);
        return null;
      }

      if (data) {
        // userId and nickname might be returned, but we mainly care about isAdmin
        setSession(data.userId, data.isAdmin, data.nickname as any);
        setRoom(data.room);
        return data;
      }

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
