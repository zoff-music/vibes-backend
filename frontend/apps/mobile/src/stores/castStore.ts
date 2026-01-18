import { create } from 'zustand';
import type { CastDevice, CastSession, CastError } from '../types/casting';
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
  updateRoomInfo: (roomInfo: { name: string; participantCount: number }) => Promise<void>;
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
    try {
      set({ lastError: null });
      
      // Set up event listeners
      castManager.onDeviceAvailable((device) => {
        set((state) => {
          // Remove existing device with same ID and add updated one
          const filteredDevices = state.availableDevices.filter(d => d.id !== device.id);
          return {
            availableDevices: [...filteredDevices, device]
          };
        });
      });

      castManager.onSessionStateChange((session) => {
        set({
          currentSession: session,
          isConnected: session.state === 'connected'
        });
      });

      castManager.onCastError((error) => {
        set({ lastError: error });
      });

      // Discover initial devices
      const devices = await castManager.discoverDevices();
      set({
        isInitialized: true,
        availableDevices: devices
      });
    } catch (error) {
      console.error('Failed to initialize casting:', error);
      set({
        lastError: {
          code: 'INITIALIZATION_FAILED',
          description: 'Failed to initialize casting system',
          details: error
        }
      });
    }
  },

  discoverDevices: async () => {
    try {
      set({ isDiscovering: true, lastError: null });
      const devices = await castManager.discoverDevices();
      set({ 
        availableDevices: devices,
        isDiscovering: false 
      });
    } catch (error) {
      console.error('Failed to discover devices:', error);
      set({
        isDiscovering: false,
        lastError: {
          code: 'DISCOVERY_FAILED',
          description: 'Failed to discover casting devices',
          details: error
        }
      });
    }
  },

  connectToDevice: async (deviceId: string) => {
    try {
      set({ lastError: null });
      const session = await castManager.connectToDevice(deviceId);
      set({
        currentSession: session,
        isConnected: session.state === 'connected'
      });
    } catch (error) {
      console.error('Failed to connect to device:', error);
      set({
        lastError: {
          code: 'CONNECTION_FAILED',
          description: 'Failed to connect to casting device',
          details: error
        }
      });
    }
  },

  disconnectFromDevice: async (deviceId: string) => {
    try {
      set({ lastError: null });
      await castManager.disconnectFromDevice(deviceId);
      set({
        currentSession: null,
        isConnected: false
      });
    } catch (error) {
      console.error('Failed to disconnect from device:', error);
      set({
        lastError: {
          code: 'DISCONNECTION_FAILED',
          description: 'Failed to disconnect from casting device',
          details: error
        }
      });
    }
  },

  castCurrentSong: async (song: any) => {
    try {
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
          images: song.thumbnailUrl ? [{
            url: song.thumbnailUrl,
            height: 480,
            width: 640
          }] : []
        },
        duration: song.duration
      };

      await castManager.castMedia(mediaInfo);
    } catch (error) {
      console.error('Failed to cast song:', error);
      set({
        lastError: {
          code: 'CAST_MEDIA_FAILED',
          description: 'Failed to cast media to device',
          details: error
        }
      });
      throw error; // Re-throw so calling code can handle it
    }
  },

  syncPlaybackState: async (state: any) => {
    try {
      if (!get().isConnected) return;
      
      set({ lastError: null });
      await castManager.syncPlaybackState(state);
    } catch (error) {
      console.error('Failed to sync playback state:', error);
      set({
        lastError: {
          code: 'SYNC_FAILED',
          description: 'Failed to synchronize playback state',
          details: error
        }
      });
    }
  },

  updateQueue: async (queue: any[]) => {
    try {
      if (!get().isConnected) return;
      
      set({ lastError: null });
      await castManager.updateQueue(queue);
    } catch (error) {
      console.error('Failed to update queue:', error);
      set({
        lastError: {
          code: 'QUEUE_UPDATE_FAILED',
          description: 'Failed to update queue on cast device',
          details: error
        }
      });
    }
  },

  updateRoomInfo: async (roomInfo: { name: string; participantCount: number }) => {
    try {
      if (!get().isConnected) return;
      
      set({ lastError: null });
      await castManager.updateRoomInfo(roomInfo);
    } catch (error) {
      console.error('Failed to update room info:', error);
      set({
        lastError: {
          code: 'ROOM_UPDATE_FAILED',
          description: 'Failed to update room info on cast device',
          details: error
        }
      });
    }
  },

  clearError: () => {
    set({ lastError: null });
  },

  cleanup: () => {
    try {
      castManager.destroy();
      set({
        isInitialized: false,
        availableDevices: [],
        currentSession: null,
        isConnected: false,
        lastError: null,
        isDiscovering: false
      });
    } catch (error) {
      console.error('Error during cleanup:', error);
    }
  }
}));