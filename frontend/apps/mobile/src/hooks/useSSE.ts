import { useEffect, useRef } from 'react';
import { useRoomStore } from '../stores/roomStore';
import { useQueueStore } from '../stores/queueStore';
import { usePlaybackStore } from '../stores/playbackStore';
import { Room, Song, PlaybackState, RoomUser } from '@vibez/shared';

const API_URL = (import.meta as any).env?.VITE_API_URL || 'http://localhost:8080';

export const useSSE = (roomId: string | undefined) => {
  const { setRoom, setUsers } = useRoomStore();
  const { setSongs } = useQueueStore();
  const { setPlaybackState } = usePlaybackStore();
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!roomId) return;

    const connect = () => {
      const url = `${API_URL}/api/v1/rooms/${roomId}/events`;
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.addEventListener('room_state', (e) => {
        try {
          const room = JSON.parse(e.data) as Room;
          setRoom(room);
        } catch (err) {
          console.error('Failed to parse room_state event', err);
        }
      });

      es.addEventListener('songs_update', (e) => {
        try {
          const songs = JSON.parse(e.data) as Song[];
          console.log('[SSE] songs_update received:', songs);
          setSongs(songs);
        } catch (err) {
          console.error('Failed to parse songs_update event', err);
        }
      });

      es.addEventListener('playback_update', (e) => {
        try {
          const state = JSON.parse(e.data) as PlaybackState;
          console.log('[SSE] playback_update received:', state);
          setPlaybackState(state);
        } catch (err) {
          console.error('Failed to parse playback_update event', err);
        }
      });

      es.addEventListener('song_added', (e) => {
        try {
          const song = JSON.parse(e.data) as Song;
          console.log('[SSE] song_added received:', song);
          window.dispatchEvent(new CustomEvent('song-added', { detail: song }));
        } catch (err) {
          console.error('Failed to parse song_added event', err);
        }
      });

      es.addEventListener('users_update', (e) => {
        try {
          const users = JSON.parse(e.data) as RoomUser[];
          setUsers(users);
        } catch (err) {
          console.error('Failed to parse users_update event', err);
        }
      });

      es.onerror = (e) => {
        console.error('SSE connection error', e);
        es.close();
        // Reconnect after 5 seconds
        setTimeout(connect, 5000);
      };
    };

    connect();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, [roomId, setRoom, setUsers, setSongs, setPlaybackState]);
};
