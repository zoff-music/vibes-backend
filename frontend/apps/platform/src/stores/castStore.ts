import type {
  CastDevice,
  CastError,
  CastSession,
  PlaybackState,
  Song,
} from '@vibez/models';
import { safeWrap, safeWrapAsync } from '@vibez/shared';
import { create } from 'zustand';
import { castManager } from '../services/castManager';

const isHttpUrl = (value: string) =>
  value.startsWith('http://') || value.startsWith('https://');

const buildSoundCloudContentId = (sourceId: string) => {
  if (!sourceId) {
    return '';
  }

  if (isHttpUrl(sourceId)) {
    return sourceId;
  }

  return `https://api.soundcloud.com/tracks/${sourceId}`;
};

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
  castCurrentSong: (song: Song) => Promise<void>;
  syncPlaybackState: (state: PlaybackState) => Promise<void>;
  updateQueue: (queue: Song[]) => Promise<void>;
  updateRoomInfo: (roomInfo: {
    name: string;
    participantCount: number;
  }) => Promise<void>;
  joinRoom: (roomId: string) => Promise<void>;
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
    console.log('[Cast] store initialize:start');
    set({ lastError: null });

    // Set up event listeners
    castManager.onDeviceAvailable((device) => {
      console.log('[Cast] store device available', device);
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
      console.log('[Cast] store session state change', session);
      set({
        currentSession: session,
        isConnected: session.state === 'connected',
      });
    });

    castManager.onCastError((error) => {
      console.error('[Cast] store received error', error);
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

    console.log('[Cast] store initialize:devices', devices);
    set({
      isInitialized: true,
      availableDevices: devices || [],
    });
  },

  discoverDevices: async () => {
    console.log('[Cast] store discoverDevices:start');
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

    console.log('[Cast] store discoverDevices:done', devices);
    set({
      availableDevices: devices || [],
      isDiscovering: false,
    });
  },

  connectToDevice: async (deviceId: string) => {
    console.log('[Cast] store connectToDevice:start', { deviceId });
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

    console.log('[Cast] store connectToDevice:done', session);
    set({
      currentSession: session,
      isConnected: session.state === 'connected',
    });
  },

  disconnectFromDevice: async (deviceId: string) => {
    console.log('[Cast] store disconnectFromDevice:start', { deviceId });
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

    console.log('[Cast] store disconnectFromDevice:done', { deviceId });
    set({
      currentSession: null,
      isConnected: false,
    });
  },

  castCurrentSong: async (song: Song) => {
    if (!get().isConnected) {
      throw new Error('No active casting session');
    }

    console.log('[Cast] store castCurrentSong:start', {
      sourceType: song?.sourceType,
      title: song?.title,
      sourceId: song?.sourceId,
    });
    set({ lastError: null });

    // Build content URL based on source type
    let contentId = '';
    switch (song.sourceType) {
      case 'youtube':
        contentId = `https://www.youtube.com/watch?v=${song.sourceId}`;
        break;
      case 'spotify':
        contentId = `spotify:track:${song.sourceId}`;
        break;
      case 'soundcloud':
        contentId = buildSoundCloudContentId(song.sourceId);
        break;
      default:
        contentId = song.sourceId || '';
    }

    const mediaInfo = {
      contentId,
      contentType: song.sourceType === 'youtube' ? 'video/mp4' : 'audio/mp3',
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
    console.log('[Cast] store castCurrentSong:done');
  },

  syncPlaybackState: async (state: PlaybackState) => {
    if (!get().isConnected) return;

    console.log('[Cast] store syncPlaybackState:start', {
      isPlaying: state?.isPlaying,
      positionMs: state?.positionMs,
      title: state?.currentSong?.title,
    });
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

  updateQueue: async (queue: Song[]) => {
    if (!get().isConnected) return;

    console.log('[Cast] store updateQueue:start', { count: queue.length });
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

    console.log('[Cast] store updateRoomInfo:start', roomInfo);
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

  joinRoom: async (roomId: string) => {
    if (!get().isConnected) return;

    console.log('[Cast] store joinRoom:start', { roomId });
    set({ lastError: null });
    const [error, _] = await safeWrapAsync(castManager.joinRoom(roomId));

    if (error) {
      console.error('Failed to join room on cast device:', error);
      set({
        lastError: {
          code: 'JOIN_ROOM_FAILED',
          description: 'Failed to send join room message',
          details: error,
        },
      });
    }
  },

  clearError: () => {
    set({ lastError: null });
  },

  cleanup: () => {
    console.log('[Cast] store cleanup:start');
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
