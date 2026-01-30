import type { Song } from '@vibez/shared';
import { usePlaybackStore } from '@vibez/shared';
import { useEffect } from 'react';
import { api } from '../lib/api';
import type { QueueItem, RoomInfo } from '../types';
import { normalizeSong } from '../utils/songUtils';

interface UseRoomSyncProps {
  roomId: string | null;
  setQueue: (queue: QueueItem[]) => void;
  setRoomInfo: React.Dispatch<React.SetStateAction<RoomInfo | null>>;
  setStatusText: (text: string) => void;
  setRoomMode: (mode: string | null) => void;
  setError: (err: string | null) => void;
  updateMediaMetadata: (song: Song) => void;
  debugMode: boolean;
}

type SSEMessage =
  | { type: 'connected'; data: any }
  | { type: 'playback_update'; data: any }
  | { type: 'songs_update'; data: Song[] }
  | { type: 'settings_update'; data: any }
  | { type: 'users_update'; data: number };

export function useRoomSync({
  roomId,
  setQueue,
  setRoomInfo,
  setStatusText,
  setRoomMode,
  setError,
  updateMediaMetadata,
}: UseRoomSyncProps) {
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setIsPlaying = usePlaybackStore((state) => state.setIsPlaying);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (!roomId) return;

    const casterId = params.get('casterId') || params.get('casterUserId') || '';

    let isMounted = true;
    let unsubscribe: (() => void) | null = null;

    const connect = async () => {
      if (!roomId) return;

      // Fetch initial state
      Promise.all([
        api.get('/rooms/{id}/songs', { id: roomId }),
        api.get('/rooms/{id}/states', { id: roomId }),
      ])
        .then(([queueRes, playbackRes]) => {
          if (!isMounted) return;

          const [songsErr, songs] = queueRes;
          const [playbackErr, playbackState] = playbackRes;

          if (!songsErr && songs) {
            console.log(
              `[Cast] Fetched ${songs.length} songs for room ${roomId}`,
            );
            const normalizedSongs = songs.map((s) => normalizeSong(s));
            setQueue(normalizedSongs);
          } else if (songsErr) {
            console.error(
              `[Cast] Failed to fetch songs for room ${roomId}:`,
              songsErr,
            );
          }

          if (!playbackErr && playbackState && playbackState.currentSong) {
            const normalizedSong = normalizeSong(playbackState.currentSong);
            setPlaybackState({
              ...playbackState,
              currentSong: normalizedSong,
            });
            setIsPlaying(playbackState.isPlaying);
            setStatusText(`Now Playing: ${normalizedSong.title}`);
            updateMediaMetadata(normalizedSong);
          }
        })
        .catch((unexpectedErr) => {
          if (!isMounted) return;
          setError(`Fetch Error: ${unexpectedErr.message}`);
        });

      const [err, stop] = await api.sse(
        '/rooms/{id}/events',
        {
          $search: {
            castReceiver: '1',
            casterId: casterId || undefined,
          },
          id: roomId,
        },
        (result: [Error | null, unknown]) => {
          const [eventError, message] = result;
          if (eventError) {
            // connection error
            return;
          }
          if (!message || !isMounted) return;

          const typedMessage = message as SSEMessage;

          switch (typedMessage.type) {
            case 'connected':
              setStatusText(`Connected to ${roomId}`);
              break;
            case 'playback_update': {
              const data = typedMessage.data;
              const normalizedSong = data.currentSong
                ? normalizeSong(data.currentSong)
                : null;

              setPlaybackState({
                ...data,
                currentSong: normalizedSong,
              });

              if (normalizedSong) {
                updateMediaMetadata(normalizedSong);
                setStatusText(`Now Playing: ${normalizedSong.title}`);
              } else {
                setStatusText('Ready for Casting');
              }

              setIsPlaying(data.isPlaying);
              break;
            }
            case 'songs_update':
              if (Array.isArray(typedMessage.data)) {
                const normalizedQueue = typedMessage.data.map((s) =>
                  normalizeSong(s),
                );
                setQueue(normalizedQueue);
              }
              break;
            case 'settings_update':
              setRoomMode(typedMessage.data.mode);
              setRoomInfo((current) => ({
                name: typedMessage.data.name,
                participantCount: current?.participantCount ?? 0,
              }));
              break;
            case 'users_update':
              setRoomInfo((current) => ({
                name: current?.name || roomId,
                participantCount: typedMessage.data,
              }));
              break;
          }
        },
        {
          headers: {
            'X-Cast-Receiver': '1',
            ...(casterId ? { 'X-Cast-Caster-Id': casterId } : {}),
          },
        },
      );

      if (!isMounted) {
        if (!err && stop) {
          stop();
        }
        return;
      }

      if (!err && stop) {
        unsubscribe = stop;
      }
    };

    connect();

    return () => {
      isMounted = false;
      if (unsubscribe) unsubscribe();
    };
  }, [
    roomId,
    setQueue,
    setRoomInfo,
    setStatusText,
    setRoomMode,
    setError,
    updateMediaMetadata,
    setPlaybackState,
    setIsPlaying,
  ]);
}
