import { usePlaybackStore } from '@vibez/shared';
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
    clearError,
  } = useCastStore();

  const currentSong = usePlaybackStore((state) => state.currentSong);
  const isPlaying = usePlaybackStore((state) => state.isPlaying);

  // Create stable callback references
  const stableCastCurrentSong = useCallback(castCurrentSong, []);
  const stableSyncPlaybackState = useCallback(syncPlaybackState, []);

  // Initialize casting when hook is first used
  useEffect(() => {
    if (!isInitialized) {
      console.log('[Cast] useCasting initializing');
      initialize();
    }
  }, [isInitialized, initialize]);

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

  // Sync playback state with cast device
  useEffect(() => {
    // Only sync if we have an active session with media
    if (isConnected && currentSong && currentSession?.mediaSessionId) {
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
      }).catch((error) => {
        console.error('Failed to sync playback state:', error);
      });
    }
  }, [
    isConnected,
    isPlaying,
    currentSong,
    currentSession?.mediaSessionId,
    stableSyncPlaybackState,
  ]);

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
