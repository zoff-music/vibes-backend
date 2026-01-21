import type { CastDevice, CastError, CastSession } from '@vibez/models';
import { safeWrap, safeWrapAsync } from '@vibez/shared';
import { create } from 'zustand';
import { castManager } from '../services/castManager';

interface CastState {
  // State
  isInitialized: boolean;
  availableDevices: CastDevice[];
  currentSession: CastSession | null;
  isConnected: boolean;
  lastError: CastError | null;
  isDiscovering: boolean;

  // Actions
  initialize: () => Promise<void>;
  discoverDevices: () => Promise<void>;
  connectToDevice: (deviceId: string) => Promise<void>;
  disconnectFromDevice: (deviceId: string) => Promise<void>;
  castCurrentSong: (song: any) => Promise<void>;
  syncPlaybackState: (state: any) => Promise<void>;
  updateQueue: (queue: any[]) => Promise<void>;
  updateRoomInfo: (roomInfo: {
    name: string;
    participantCount: number;
  }) => Promise<void>;
  clearError: () => void;
  cleanup: () => void;
}

export const useCastStore = create<CastState>((set, get) => ({
  // Initial state
  isInitialized: false,
  availableDevices: [],
  currentSession: null,
  isConnected: false,
  lastError: null,
  isDiscovering: false,

  // Actions
  initialize: async () => {
    set({ lastError: null });

    // Set up event listeners
    castManager.onDeviceAvailable((device) => {
      set((state) => {
        // Remove existing device with same ID and add updated one
        const filteredDevices = state.availableDevices.filter(
          (d) => d.id !== device.id,
        );
        return {
          availableDevices: [...filteredDevices, device],
        };
      });
    });

    castManager.onSessionStateChange((session) => {
      set({
        currentSession: session,
        isConnected: session.state === 'connected',
      });
    });

    castManager.onCastError((error) => {
      set({ lastError: error });
    });

    // Discover initial devices
    const [error, devices] = await safeWrapAsync(castManager.discoverDevices());

    if (error) {
      console.error('Failed to initialize casting:', error);
      set({
        lastError: {
          code: 'INITIALIZATION_FAILED',
          description: 'Failed to initialize casting system',
          details: error,
        },
      });
      return;
    }

    set({
      isInitialized: true,
      availableDevices: devices || [],
    });
  },

  discoverDevices: async () => {
    set({ isDiscovering: true, lastError: null });
    const [error, devices] = await safeWrapAsync(castManager.discoverDevices());

    if (error) {
      console.error('Failed to discover devices:', error);
      set({
        isDiscovering: false,
        lastError: {
          code: 'DISCOVERY_FAILED',
          description: 'Failed to discover casting devices',
          details: error,
        },
      });
      return;
    }

    set({
      availableDevices: devices || [],
      isDiscovering: false,
    });
  },

  connectToDevice: async (deviceId: string) => {
    set({ lastError: null });
    const [error, session] = await safeWrapAsync(
      castManager.connectToDevice(deviceId),
    );

    if (error || !session) {
      console.error('Failed to connect to device:', error);
      set({
        lastError: {
          code: 'CONNECTION_FAILED',
          description: 'Failed to connect to casting device',
          details: error,
        },
      });
      return;
    }

    set({
      currentSession: session,
      isConnected: session.state === 'connected',
    });
  },

  disconnectFromDevice: async (deviceId: string) => {
    set({ lastError: null });
    const [error, _] = await safeWrapAsync(
      castManager.disconnectFromDevice(deviceId),
    );

    if (error) {
      console.error('Failed to disconnect from device:', error);
      set({
        lastError: {
          code: 'DISCONNECTION_FAILED',
          description: 'Failed to disconnect from casting device',
          details: error,
        },
      });
      return;
    }

    set({
      currentSession: null,
      isConnected: false,
    });
  },

  castCurrentSong: async (song: any) => {
    if (!get().isConnected) {
      throw new Error('No active casting session');
    }

    set({ lastError: null });

    const mediaInfo = {
      contentId: `https://www.youtube.com/watch?v=${song.sourceId}`,
      contentType: 'video/mp4',
      streamType: 'BUFFERED' as const,
      metadata: {
        title: song.title || 'Unknown Title',
        artist: song.artist || 'Unknown Artist',
        images: song.thumbnailUrl
          ? [
              {
                url: song.thumbnailUrl,
                height: 480,
                width: 640,
              },
            ]
          : [],
      },
      duration: song.duration,
    };

    const [error, _] = await safeWrapAsync(castManager.castMedia(mediaInfo));

    if (error) {
      console.error('Failed to cast song:', error);
      set({
        lastError: {
          code: 'CAST_MEDIA_FAILED',
          description: 'Failed to cast media to device',
          details: error,
        },
      });
      throw error;
    }
  },

  syncPlaybackState: async (state: any) => {
    if (!get().isConnected) return;

    set({ lastError: null });
    const [error, _] = await safeWrapAsync(
      castManager.syncPlaybackState(state),
    );

    if (error) {
      console.error('Failed to sync playback state:', error);
      set({
        lastError: {
          code: 'SYNC_FAILED',
          description: 'Failed to synchronize playback state',
          details: error,
        },
      });
    }
  },

  updateQueue: async (queue: any[]) => {
    if (!get().isConnected) return;

    set({ lastError: null });
    const [error, _] = await safeWrapAsync(castManager.updateQueue(queue));

    if (error) {
      console.error('Failed to update queue:', error);
      set({
        lastError: {
          code: 'QUEUE_UPDATE_FAILED',
          description: 'Failed to update queue on cast device',
          details: error,
        },
      });
    }
  },

  updateRoomInfo: async (roomInfo: {
    name: string;
    participantCount: number;
  }) => {
    if (!get().isConnected) return;

    set({ lastError: null });
    const [error, _] = await safeWrapAsync(
      castManager.updateRoomInfo(roomInfo),
    );

    if (error) {
      console.error('Failed to update room info:', error);
      set({
        lastError: {
          code: 'ROOM_UPDATE_FAILED',
          description: 'Failed to update room info on cast device',
          details: error,
        },
      });
    }
  },

  clearError: () => {
    set({ lastError: null });
  },

  cleanup: () => {
    const [error, _] = safeWrap(() => castManager.destroy());

    if (error) {
      console.error('Error during cleanup:', error);
    }

    set({
      isInitialized: false,
      availableDevices: [],
      currentSession: null,
      isConnected: false,
      lastError: null,
      isDiscovering: false,
    });
  },
}));
