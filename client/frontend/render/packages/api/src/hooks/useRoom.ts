import {
  type RoomSettings,
  type RoomUpdate,
  safeWrapAsync,
  useRoomStore,
} from '@vibez/shared';
import { useCallback, useState } from 'react';
import { api } from '../index';
import { USE_SSE_CALLBACKS, useSSE } from './useSSE';

// Simple request deduplication map to handle strict mode double-invocations
const IN_FLIGHT_REQUESTS = new Map<string, Promise<any>>();

export const useRoom = (roomId: string, callbacks?: USE_SSE_CALLBACKS) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const room = useRoomStore((state) => state.room);
  const users = useRoomStore((state) => state.users);
  const userId = useRoomStore((state) => state.userId);
  const setRoom = useRoomStore((state) => state.setRoom);
  const setSession = useRoomStore((state) => state.setSession);
  const reset = useRoomStore((state) => state.reset);

  useSSE(roomId, callbacks);

  const fetchRoom = useCallback(async () => {
    if (!roomId) return;

    setIsLoading(true);
    const key = `fetchRoom:${roomId}`;

    const cachedPromise = IN_FLIGHT_REQUESTS.get(key) as
      | Promise<[Error | null, any]>
      | undefined;
    let promise: Promise<[Error | null, any]>;
    if (cachedPromise) {
      promise = cachedPromise;
    } else {
      promise = api.get('/rooms/{id}', { id: roomId });
      IN_FLIGHT_REQUESTS.set(key, promise);
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

      const cachedPromise = IN_FLIGHT_REQUESTS.get(key) as
        | Promise<[Error | null, any]>
        | undefined;
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
        const sessionId = data.sessionId || data.userId;
        setSession(sessionId, data.isAdmin, data.nickname || undefined);
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
    async (updates: RoomUpdate) => {
      setIsLoading(true);
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

  const updateRoomSettings = useCallback(
    (settings: RoomSettings) => updateRoom({ settings }),
    [updateRoom],
  );

  return {
    room,
    users,
    userId,
    isAdmin: useRoomStore.getState().isAdmin, // Getting it from store
    isLoading,
    error,
    fetchRoom,
    joinRoom,
    updateRoomSettings,
    updateRoom,
    leaveRoom,
  };
};
