import { api } from '@vibez/api';
import { PlaybackState, Song, safeWrap } from '@vibez/shared';
import { useEffect, useRef } from 'react';
import { usePlaybackStore } from '../stores/playbackStore';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';

const ACTIVE_CONNECTIONS = new Map<
  string,
  { count: number; unsubscribe: () => void }
>();

export const useSSE = (roomId: string | undefined) => {
  const setRoom = useRoomStore((state) => state.setRoom);
  const setUsers = useRoomStore((state) => state.setUsers);
  const setSongs = useQueueStore((state) => state.setSongs);
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);

  // Use a ref to track if *this specific hook instance* is responsible for a subscription
  // This helps handle strict mode double-invocations locally
  const isSubscribedRef = useRef(false);

  useEffect(() => {
    if (!roomId) return;

    // Prevent multiple connections for the same room
    // In strict mode, this effect runs twice.
    // We want to ensure we only have one actual wire connection per room.
    // We can use a reference counting mechanism.

    const setupConnection = async () => {
      let connection = ACTIVE_CONNECTIONS.get(roomId);

      if (!connection) {
        // No active connection, create one
        console.log('[SSE] establishing new connection for', roomId);
        const [err, unsubscribe] = await api.sse(
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

            switch (message.type) {
              case 'connected':
                console.log('[SSE] connected:', message.data);
                break;
              case 'songs_update': {
                const [_, error] = safeWrap(() => {
                  const songs = message.data as Song[];
                  // console.log('[SSE] songs_update received:', songs.length);
                  setSongs(songs);
                });
                if (error)
                  console.error('Failed to parse songs_update event', error);
                break;
              }
              case 'playback_update': {
                const [_, error] = safeWrap(() => {
                  const state = message.data as PlaybackState;
                  // console.log('[SSE] playback_update received', state);
                  setPlaybackState(state);
                });
                if (error)
                  console.error('Failed to parse playback_update event', error);
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
                  console.log('[SSE] settings_update received:', message.data);
                  setRoom(message.data as any);
                });
                if (error)
                  console.error('Failed to parse settings_update event', error);
                break;
              }
            }
          },
        );

        if (err) {
          console.error('[SSE] Failed to connect:', err);
          return;
        }

        connection = { count: 0, unsubscribe };
        ACTIVE_CONNECTIONS.set(roomId, connection);
      } else {
        console.log('[SSE] reusing existing connection for', roomId);
      }

      // Increment ref count
      connection.count++;
      isSubscribedRef.current = true;
    };

    setupConnection();

    return () => {
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
            console.log('[SSE] closing connection for', roomId);
            connection.unsubscribe();
            ACTIVE_CONNECTIONS.delete(roomId);
          }
        }
        isSubscribedRef.current = false;
      }
    };
  }, [roomId, setRoom, setUsers, setSongs, setPlaybackState]);
};
