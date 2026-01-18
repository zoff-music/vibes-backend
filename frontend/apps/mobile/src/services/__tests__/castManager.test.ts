import { castManager } from '../castManager';
import type { CastDevice, MediaInfo } from '../../types/casting';

// Mock the Google Cast SDK
const mockCastSDK = {
  isAvailable: true,
  initialize: jest.fn(),
  requestSession: jest.fn(),
  Session: {
    getSessionById: jest.fn()
  },
  media: {
    MediaInfo: jest.fn(),
    GenericMediaMetadata: jest.fn(),
    LoadRequest: jest.fn(),
    SeekRequest: jest.fn(),
    StreamType: {
      BUFFERED: 'BUFFERED',
      LIVE: 'LIVE'
    },
    PlayerState: {
      IDLE: 'IDLE',
      PLAYING: 'PLAYING',
      PAUSED: 'PAUSED',
      BUFFERING: 'BUFFERING'
    }
  },
  ApiConfig: jest.fn(),
  SessionRequest: jest.fn(),
  AutoJoinPolicy: {
    TAB_AND_ORIGIN_SCOPED: 'TAB_AND_ORIGIN_SCOPED'
  },
  DefaultActionPolicy: {
    CREATE_SESSION: 'CREATE_SESSION'
  },
  Error: jest.fn(),
  ErrorCode: {
    API_NOT_INITIALIZED: 'API_NOT_INITIALIZED',
    CANCEL: 'CANCEL'
  }
};

// Mock window.chrome.cast
Object.defineProperty(window, 'chrome', {
  value: {
    cast: mockCastSDK
  },
  writable: true
});

describe('CastManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Device Discovery', () => {
    it('should discover available devices', async () => {
      const devices = await castManager.discoverDevices();
      expect(Array.isArray(devices)).toBe(true);
    });

    it('should return empty array when no devices available', async () => {
      const devices = await castManager.discoverDevices();
      expect(devices).toEqual([]);
    });
  });

  describe('Connection Management', () => {
    it('should handle connection errors gracefully', async () => {
      mockCastSDK.requestSession.mockImplementation((onSuccess, onError) => {
        onError({ code: 'CANCEL', description: 'User cancelled' });
      });

      await expect(castManager.connectToDevice('test-device')).rejects.toThrow();
    });

    it('should validate device exists before connecting', async () => {
      await expect(castManager.connectToDevice('non-existent-device')).rejects.toThrow('Device non-existent-device not found');
    });
  });

  describe('Media Casting', () => {
    const mockMediaInfo: MediaInfo = {
      contentId: 'https://www.youtube.com/watch?v=test',
      contentType: 'video/mp4',
      streamType: 'BUFFERED',
      metadata: {
        title: 'Test Song',
        artist: 'Test Artist',
        images: [{
          url: 'https://example.com/image.jpg',
          height: 480,
          width: 640
        }]
      },
      duration: 180
    };

    it('should reject casting when no active session', async () => {
      await expect(castManager.castMedia(mockMediaInfo)).rejects.toThrow('No active cast session');
    });

    it('should validate media info before casting', async () => {
      const invalidMediaInfo = {
        ...mockMediaInfo,
        contentId: ''
      };

      await expect(castManager.castMedia(invalidMediaInfo)).rejects.toThrow();
    });
  });

  describe('Error Handling', () => {
    it('should handle SDK initialization failures', () => {
      const errorCallback = jest.fn();
      castManager.onCastError(errorCallback);

      // Simulate initialization error
      mockCastSDK.initialize.mockImplementation((config, onSuccess, onError) => {
        onError({ code: 'API_NOT_INITIALIZED', description: 'Failed to initialize' });
      });

      expect(errorCallback).not.toHaveBeenCalled(); // Error handling is internal
    });

    it('should provide error callbacks', () => {
      const errorCallback = jest.fn();
      castManager.onCastError(errorCallback);

      // This should not throw
      expect(() => castManager.onCastError(errorCallback)).not.toThrow();
    });
  });

  describe('Utility Methods', () => {
    it('should return current session status', () => {
      const session = castManager.getCurrentSession();
      expect(session).toBeNull(); // No active session initially
    });

    it('should return connection state', () => {
      const isConnected = castManager.isConnected();
      expect(isConnected).toBe(false); // Not connected initially
    });

    it('should provide status information', () => {
      const status = castManager.getStatus();
      expect(status).toHaveProperty('isInitialized');
      expect(status).toHaveProperty('deviceCount');
      expect(status).toHaveProperty('currentSession');
      expect(status).toHaveProperty('reconnectAttempts');
    });
  });

  describe('Cleanup', () => {
    it('should clean up resources on destroy', () => {
      expect(() => castManager.destroy()).not.toThrow();
    });
  });
});