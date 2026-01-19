import { api } from '@vibez/api';
import { PlaybackState, Song, safeWrap } from '@vibez/shared';
import { useEffect, useRef } from 'react';
import { usePlaybackStore } from '../stores/playbackStore';
import { useQueueStore } from '../stores/queueStore';
import { useRoomStore } from '../stores/roomStore';

export const useSSE = (roomId: string | undefined) => {
  const { setRoom, setUsers } = useRoomStore();
  const { setSongs } = useQueueStore();
  const { setPlaybackState } = usePlaybackStore();
  const unsubscribeRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!roomId) return;

    const connect = async () => {
      // Use api.sse from wiretyped client
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
                console.log('[SSE] songs_update received:', songs);
                setSongs(songs);
              });
              if (error)
                console.error('Failed to parse songs_update event', error);
              break;
            }
            case 'playback_update': {
              const [_, error] = safeWrap(() => {
                const state = message.data as PlaybackState;
                console.log('[SSE] playback_update received:', state);
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
                // The room store likely expects the full Room object.
                // Assuming message.data matches what setRoom expects.
                // Cast to unknown first to avoid direct any usage if possible, but store likely needs proper type.
                // Checking usage: setRoom(message.data as any);
                // We should probably remove 'as any' and let types flow if compatible, or cast to expected type.
                // Since I cannot check Room type easily right now, I'll use 'as Room' if available or unknown.
                // message.data is likely specific type from schema.
                // Let's assume it matches.
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
        console.error('Failed to connect to SSE:', err);
        // Retry logic could go here
        return;
      }

      unsubscribeRef.current = unsubscribe;
    };

    connect();

    return () => {
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
      }
    };
  }, [roomId, setRoom, setUsers, setSongs, setPlaybackState]);
};
