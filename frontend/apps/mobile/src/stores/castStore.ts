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
  
  // Actions
  initialize: () => Promise<void>;
  discoverDevices: () => Promise<void>;
  connectToDevice: (deviceId: string) => Promise<void>;
  disconnectFromDevice: (deviceId: string) => Promise<void>;
  castCurrentSong: (song: any) => Promise<void>;
  syncPlaybackState: (state: any) => Promise<void>;
  clearError: () => void;
}

export const useCastStore = create<CastState>((set, get) => ({
  // Initial state
  isInitialized: false,
  availableDevices: [],
  currentSession: null,
  isConnected: false,
  lastError: null,

  // Actions
  initialize: async () => {
    try {
      // Set up event listeners
      castManager.onDeviceAvailable((device) => {
        set((state) => ({
          availableDevices: [...state.availableDevices.filter(d => d.id !== device.id), device]
        }));
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
      const devices = await castManager.discoverDevices();
      set({ availableDevices: devices });
    } catch (error) {
      console.error('Failed to discover devices:', error);
      set({
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
        isConnected: true
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

      const mediaInfo = {
        contentId: `https://www.youtube.com/watch?v=${song.sourceId}`,
        contentType: 'video/mp4',
        streamType: 'BUFFERED' as const,
        metadata: {
          title: song.title,
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
    }
  },

  syncPlaybackState: async (state: any) => {
    try {
      if (!get().isConnected) return;
      
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

  clearError: () => {
    set({ lastError: null });
  }
}));