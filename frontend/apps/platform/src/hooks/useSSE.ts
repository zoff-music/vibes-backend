import { api } from '@vibez/api';
import { PlaybackState, Song, safeWrap } from '@vibez/shared';
import { useEffect, useRef } from 'react';
import { usePlaybackStore } from '@vibez/shared';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';

const ACTIVE_CONNECTIONS = new Map<
  string,
  { count: number; unsubscribe: () => void }
>();

const IN_FLIGHT_CONNECTIONS = new Map<string, Promise<any>>();

// Track pending cleanups to allow "grace period" for reconnection
const PENDING_CLEANUPS = new Map<string, ReturnType<typeof setTimeout>>();

export const useSSE = (roomId: string | undefined) => {
  const setRoom = useRoomStore((state) => state.setRoom);
  const setUsers = useRoomStore((state) => state.setUsers);
  const setUsersCount = useRoomStore((state) => state.setUsersCount);
  const setSongs = useQueueStore((state) => state.setSongs);
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);

  // Use a ref to track if *this specific hook instance* is responsible for a subscription
  const isSubscribedRef = useRef(false);

  useEffect(() => {
    if (!roomId) return;
    let isMounted = true;

    const setupConnection = async () => {
      // 1. Cancel any pending cleanup for this room
      if (PENDING_CLEANUPS.has(roomId)) {
        console.log('[SSE] canceling pending cleanup for', roomId);
        clearTimeout(PENDING_CLEANUPS.get(roomId));
        PENDING_CLEANUPS.delete(roomId);
      }

      let connection = ACTIVE_CONNECTIONS.get(roomId);

      if (!connection) {
        // Check if there is an in-flight connection
        let inFlight = IN_FLIGHT_CONNECTIONS.get(roomId);

        if (!inFlight) {
          console.log('[SSE] establishing new connection for', roomId);
          inFlight = api.sse(
            '/rooms/{id}/events',
            {
              id: roomId,
            },
            (result) => {
              const [err, message] = result;
              if (err) {
                console.error('SSE Error:', err);
                return;
              }

              if (!message) return;

              switch ((message as any).type) {
                case 'connected':
                  console.log('[SSE] connected:', message.data);
                  break;
                case 'songs_update': {
                  const [_, error] = safeWrap(() => {
                    const songs = message.data as Song[];
                    setSongs(songs);
                  });
                  if (error)
                    console.error('Failed to parse songs_update event', error);
                  break;
                }
                case 'playback_update': {
                  const [_, error] = safeWrap(() => {
                    const state = message.data as PlaybackState;
                    setPlaybackState(state);
                  });
                  if (error)
                    console.error(
                      'Failed to parse playback_update event',
                      error,
                    );
                  break;
                }
                case 'song_added': {
                  const [_, error] = safeWrap(() => {
                    const song = message.data as Song;
                    console.log('[SSE] song_added received:', song);
                    window.dispatchEvent(
                      new CustomEvent('song-added', { detail: song }),
                    );
                  });
                  if (error)
                    console.error('Failed to parse song_added event', error);
                  break;
                }
                case 'settings_update': {
                  const [_, error] = safeWrap(() => {
                    console.log(
                      '[SSE] settings_update received:',
                      message.data,
                    );
                    setRoom(message.data as any);
                  });
                  if (error)
                    console.error(
                      'Failed to parse settings_update event',
                      error,
                    );
                  break;
                }
                case 'users_update': {
                  const [_, error] = safeWrap(() => {
                    const count = message.data as number;
                    setUsersCount(count);
                  });
                  if (error)
                    console.error('Failed to parse users_update event', error);
                  break;
                }
              }
            },
          );
          IN_FLIGHT_CONNECTIONS.set(roomId, inFlight);
        } else {
          console.log('[SSE] waiting for in-flight connection for', roomId);
        }

        if (!inFlight) return; // Should not happen but satisfies TS

        const [err, unsubscribe] = await inFlight;

        // Remove from in-flight since it's resolved
        if (IN_FLIGHT_CONNECTIONS.get(roomId) === inFlight) {
          IN_FLIGHT_CONNECTIONS.delete(roomId);
        }

        if (err) {
          console.error('[SSE] Failed to connect:', err);
          return;
        }

        if (!isMounted) {
          console.log(
            '[SSE] component unmounted during connection, parking connection',
            roomId,
          );
          // Don't unsubscribe immediately!
          // Another component might be waiting on this same promise (Strict Mode).
          // Park it in ACTIVE_CONNECTIONS with count 0 and schedule cleanup.
          if (unsubscribe && !ACTIVE_CONNECTIONS.has(roomId)) {
            connection = { count: 0, unsubscribe };
            ACTIVE_CONNECTIONS.set(roomId, connection);

            // Schedule cleanup since no one is officially holding it yet
            if (PENDING_CLEANUPS.has(roomId))
              clearTimeout(PENDING_CLEANUPS.get(roomId));
            const timeout = setTimeout(() => {
              console.log(
                '[SSE] executing cleanup for parked connection',
                roomId,
              );
              const currentConn = ACTIVE_CONNECTIONS.get(roomId);
              if (currentConn && currentConn.count <= 0) {
                if (currentConn.unsubscribe) currentConn.unsubscribe();
                ACTIVE_CONNECTIONS.delete(roomId);
              }
              PENDING_CLEANUPS.delete(roomId);
            }, 2000);
            PENDING_CLEANUPS.set(roomId, timeout);
          }
          return;
        }

        // Check again if ACTIVE_CONNECTIONS was set by another race
        if (!ACTIVE_CONNECTIONS.has(roomId) && unsubscribe) {
          connection = { count: 0, unsubscribe };
          ACTIVE_CONNECTIONS.set(roomId, connection);
        } else {
          connection = ACTIVE_CONNECTIONS.get(roomId);
        }
      } else {
        console.log('[SSE] reusing existing connection for', roomId);
      }

      // Vital: Clear any pending cleanup now that we're about to use it!
      // This handles the case where we just "parked" it in the race above, or it was pending from before
      if (PENDING_CLEANUPS.has(roomId)) {
        console.log('[SSE] canceling pending cleanup (late) for', roomId);
        clearTimeout(PENDING_CLEANUPS.get(roomId));
        PENDING_CLEANUPS.delete(roomId);
      }

      if (connection && isMounted) {
        connection.count++;
        isSubscribedRef.current = true;
        console.log('[SSE] ref count incremented to', connection.count);
      }
    };

    setupConnection();

    return () => {
      isMounted = false;
      if (isSubscribedRef.current) {
        const connection = ACTIVE_CONNECTIONS.get(roomId);
        if (connection) {
          connection.count--;
          console.log(
            '[SSE] connection ref count decreased for',
            roomId,
            'to',
            connection.count,
          );

          if (connection.count <= 0) {
            // GRACE PERIOD: Don't close immediately. Wait a bit.
            console.log('[SSE] scheduling cleanup for', roomId);

            // Clear any existing timeout just in case
            if (PENDING_CLEANUPS.has(roomId)) {
              clearTimeout(PENDING_CLEANUPS.get(roomId));
            }

            const timeout = setTimeout(() => {
              console.log('[SSE] executing cleanup for', roomId);
              // Check again if count is still 0 (should be, unless logic error)
              const currentConn = ACTIVE_CONNECTIONS.get(roomId);
              if (currentConn && currentConn.count <= 0) {
                if (currentConn.unsubscribe) currentConn.unsubscribe();
                ACTIVE_CONNECTIONS.delete(roomId);
              }
              PENDING_CLEANUPS.delete(roomId);
            }, 2000); // 2 second grace period

            PENDING_CLEANUPS.set(roomId, timeout);
          }
        }
        isSubscribedRef.current = false;
      }
    };
  }, [roomId, setRoom, setUsers, setSongs, setPlaybackState]);
};
