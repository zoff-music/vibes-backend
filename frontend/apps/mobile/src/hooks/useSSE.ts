import { useEffect, useRef } from 'react';
import { useRoomStore } from '../stores/roomStore';
import { useQueueStore } from '../stores/queueStore';
import { usePlaybackStore } from '../stores/playbackStore';
import { Song, PlaybackState } from '@vibez/shared';
import { api } from '../api/client';

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
          id: roomId
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
            case 'songs_update':
              try {
                const songs = message.data as Song[];
                console.log('[SSE] songs_update received:', songs);
                setSongs(songs);
              } catch (err) {
                console.error('Failed to parse songs_update event', err);
              }
              break;
            case 'playback_update':
              try {
                const state = message.data as PlaybackState;
                console.log('[SSE] playback_update received:', state);
                setPlaybackState(state);
              } catch (err) {
                console.error('Failed to parse playback_update event', err);
              }
              break;
            case 'song_added':
              try {
                const song = message.data as Song;
                console.log('[SSE] song_added received:', song);
                window.dispatchEvent(new CustomEvent('song-added', { detail: song }));
              } catch (err) {
                console.error('Failed to parse song_added event', err);
              }
              break;
            case 'settings_update':
              try {
                console.log('[SSE] settings_update received:', message.data);
                setRoom(message.data as any);
              } catch (err) {
                console.error('Failed to parse settings_update event', err);
              }
              break;
          }
        }
      );

      if (err) {
          console.error("Failed to connect to SSE:", err)
          // Retry logic could go here
          return
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
