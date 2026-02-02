import type { Song } from '@vibez/shared';
import { usePlaybackStore } from '@vibez/shared';
import { useCallback } from 'react';
import type { LocalCastMessage, QueueItem, RoomInfo } from '../types';
import { normalizeSong } from '../utils/songUtils';

interface UseCastMessageHandlerProps {
  setRoomId: (id: string) => void;
  setCasterId: (id: string | null) => void;
  setRoomInfo: (info: RoomInfo | null) => void;
  setQueue: (queue: QueueItem[]) => void;
  setStatusText: (text: string) => void;
  updateMediaMetadata: (song: Song) => void;
  roomMode: string | null;
}

export const useCastMessageHandler = ({
  setRoomId,
  setCasterId,
  setRoomInfo,
  setQueue,
  setStatusText,
  updateMediaMetadata,
  roomMode,
}: UseCastMessageHandlerProps) => {
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setIsPlaying = usePlaybackStore((state) => state.setIsPlaying);

  const handleCastMessage = useCallback(
    (message: LocalCastMessage) => {
      if (!message || !message.action) return;

      const action = message.action;
      const currentState = usePlaybackStore.getState();
      const currentSongId = currentState.currentSong?.id;
      const currentActualPosition = currentState.actualPositionMs;

      switch (action) {
        case 'joinRoom': {
          if (message.roomId) {
            setRoomId(message.roomId);
          }
          const casterId = message.casterId || message.sessionId || undefined;
          if (casterId !== undefined) {
            setCasterId(casterId || null);
          }
          break;
        }

        case 'updatePlayback':
        case 'syncPlayback': {
          if (message.currentSong) {
            const normalizedSong = normalizeSong(message.currentSong);
            const isSameSong = currentSongId === normalizedSong.id;

            // Prevent reset to 0 if we are already playing this song and have a position
            const shouldPreservePosition =
              isSameSong &&
              (!message.positionMs || message.positionMs === 0) &&
              currentActualPosition > 1000;
            const positionMs = shouldPreservePosition
              ? currentActualPosition
              : message.positionMs || 0;

            setPlaybackState(
              {
                currentSong: normalizedSong,
                isPlaying: message.isPlaying || false,
                positionMs: positionMs,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              },
              roomMode || undefined,
            );
            setIsPlaying(message.isPlaying || false);

            if (action === 'updatePlayback') {
              setStatusText(`Now Playing: ${message.currentSong.title}`);
            }
            updateMediaMetadata(normalizedSong);
          }
          if (
            action === 'updatePlayback' &&
            'roomInfo' in message &&
            message.roomInfo
          ) {
            setRoomInfo(message.roomInfo);
          }
          break;
        }

        case 'updateQueue':
          if (message.queue) {
            setQueue(message.queue.map((s) => normalizeSong(s)));
          }
          break;

        case 'updateRoomInfo':
          if (message.roomInfo) {
            setRoomInfo(message.roomInfo);
          }
          break;

        default:
          break;
      }
    },
    [
      roomMode,
      setIsPlaying,
      setPlaybackState,
      updateMediaMetadata,
      setRoomId,
      setCasterId,
      setRoomInfo,
      setQueue,
      setStatusText,
    ],
  );

  return handleCastMessage;
};
