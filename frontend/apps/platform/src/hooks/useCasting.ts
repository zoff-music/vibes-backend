import {
  isTruthyFlag,
  usePlaybackStore,
  useQueueStore,
  useRoomStore,
} from '@vibez/shared';
import { useCallback, useEffect } from 'react';
import { useCastStore } from '../stores/castStore';

/**
 * Hook to integrate casting functionality with the existing playback system
 */
export const useCasting = (_roomId: string) => {
  const {
    isInitialized,
    isConnected,
    currentSession,
    availableDevices,
    lastError,
    initialize,
    castCurrentSong,
    syncPlaybackState,
    updateQueue,
    updateRoomInfo,
    clearError,
  } = useCastStore();

  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);
  const queueSongs = useQueueStore((state) => state.songs);
  const room = useRoomStore((state) => state.room);
  const usersCount = useRoomStore((state) => state.usersCount);

  const isLocalEmulatorEnabled = (() => {
    if (isTruthyFlag(import.meta.env.VITE_CAST_LOCAL_EMULATOR)) {
      return true;
    }
    return (
      typeof window !== 'undefined' &&
      (window.location.hostname === 'localhost' ||
        window.location.hostname === '127.0.0.1')
    );
  })();

  // Create stable callback references
  const stableCastCurrentSong = useCallback(castCurrentSong, []);
  const stableSyncPlaybackState = useCallback(syncPlaybackState, []);
  const stableUpdateQueue = useCallback(updateQueue, []);
  const stableUpdateRoomInfo = useCallback(updateRoomInfo, []);

  // Initialize casting when hook is first used
  useEffect(() => {
    if (!isInitialized) {
      console.log('[Cast] useCasting initializing');
      initialize();
    }
  }, [isInitialized, initialize]);

  useEffect(() => {
    if (!isLocalEmulatorEnabled) return;
    if (isConnected) return;
    if (availableDevices.length === 0) return;
    console.log('[Cast] local emulator available; waiting for user connect');
  }, [isConnected, isLocalEmulatorEnabled, availableDevices.length]);

  // Cast current song when casting becomes available or song changes
  // Only cast when user explicitly connects and there's a current song
  useEffect(() => {
    // Only auto-cast if we're already connected and have a session
    if (isConnected && currentSong && currentSession) {
      console.log('🎵 Auto-casting current song:', currentSong.title);
      stableCastCurrentSong(currentSong).catch((error) => {
        console.error('Failed to cast current song:', error);
      });
    }
  }, [isConnected, currentSong, currentSession, stableCastCurrentSong]);

  // Send Join Room message when connected
  useEffect(() => {
    if (isConnected && currentSession && _roomId) {
      console.log('[Cast] Sending joinRoom handshake for room:', _roomId);
      const { joinRoom } = useCastStore.getState();
      joinRoom(_roomId).catch((err) => {
        console.error('Failed to send joinRoom handshake:', err);
      });
    }
  }, [isConnected, currentSession, _roomId]);

  // Sync playback state with cast device
  useEffect(() => {
    const isLocalSession = currentSession?.deviceId === 'local-cast-emulator';
    if (
      isConnected &&
      currentSession &&
      currentSong &&
      (isLocalSession || currentSession.mediaSessionId || isConnected) // Sync if connected regardless of media session
    ) {
      const actualPositionMs = usePlaybackStore.getState().actualPositionMs;
      console.log('[Cast] syncing playback state', {
        title: currentSong.title,
        isPlaying,
        positionMs: actualPositionMs,
        mediaSessionId: currentSession.mediaSessionId,
      });
      stableSyncPlaybackState({
        isPlaying,
        positionMs: actualPositionMs,
        currentSong,
        updatedAt: new Date().toISOString(),
        serverTimeMs: Date.now(),
      }).catch((error) => {
        console.error('Failed to sync playback state:', error);
      });
    }
  }, [
    isConnected,
    isPlaying,
    currentSong,
    currentSession?.deviceId,
    currentSession?.mediaSessionId,
    stableSyncPlaybackState,
  ]);

  useEffect(() => {
    if (currentSession?.deviceId !== 'local-cast-emulator') return;
    if (!room) return;

    stableUpdateRoomInfo({
      name: room.name,
      participantCount: usersCount,
    }).catch((error) => {
      console.error('Failed to update local room info:', error);
    });
  }, [currentSession?.deviceId, room, usersCount, stableUpdateRoomInfo]);

  useEffect(() => {
    if (currentSession?.deviceId !== 'local-cast-emulator') return;

    stableUpdateQueue(queueSongs).catch((error) => {
      console.error('Failed to update local queue:', error);
    });
  }, [currentSession?.deviceId, queueSongs, stableUpdateQueue]);

  return {
    // State
    isInitialized,
    isConnected,
    currentSession,
    availableDevices,
    lastError,

    // Actions
    clearError,

    // Computed
    isCastingAvailable: availableDevices.length > 0,
    castDeviceName: currentSession?.deviceName || null,
  };
};
