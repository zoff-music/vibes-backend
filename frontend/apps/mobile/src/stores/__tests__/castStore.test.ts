import { renderHook, act } from '@testing-library/react';
import { useCastStore } from '../castStore';

// Mock the cast manager
jest.mock('../../services/castManager', () => ({
  castManager: {
    discoverDevices: jest.fn().mockResolvedValue([]),
    connectToDevice: jest.fn().mockResolvedValue({
      id: 'test-session',
      deviceId: 'test-device',
      deviceName: 'Test Device',
      deviceType: 'chromecast',
      state: 'connected',
      startedAt: new Date()
    }),
    disconnectFromDevice: jest.fn().mockResolvedValue(undefined),
    castMedia: jest.fn().mockResolvedValue(undefined),
    syncPlaybackState: jest.fn().mockResolvedValue(undefined),
    onDeviceAvailable: jest.fn(),
    onSessionStateChange: jest.fn(),
    onCastError: jest.fn(),
    destroy: jest.fn()
  }
}));

describe('useCastStore', () => {
  beforeEach(() => {
    // Reset store state
    useCastStore.setState({
      isInitialized: false,
      availableDevices: [],
      currentSession: null,
      isConnected: false,
      lastError: null,
      isDiscovering: false
    });
  });

  it('should initialize with default state', () => {
    const { result } = renderHook(() => useCastStore());
    
    expect(result.current.isInitialized).toBe(false);
    expect(result.current.availableDevices).toEqual([]);
    expect(result.current.currentSession).toBeNull();
    expect(result.current.isConnected).toBe(false);
    expect(result.current.lastError).toBeNull();
    expect(result.current.isDiscovering).toBe(false);
  });

  it('should handle initialization', async () => {
    const { result } = renderHook(() => useCastStore());
    
    await act(async () => {
      await result.current.initialize();
    });
    
    expect(result.current.isInitialized).toBe(true);
  });

  it('should handle device discovery', async () => {
    const { result } = renderHook(() => useCastStore());
    
    await act(async () => {
      await result.current.discoverDevices();
    });
    
    // Should not throw and should update discovering state
    expect(result.current.isDiscovering).toBe(false);
  });

  it('should handle device connection', async () => {
    const { result } = renderHook(() => useCastStore());
    
    await act(async () => {
      await result.current.connectToDevice('test-device');
    });
    
    expect(result.current.isConnected).toBe(true);
    expect(result.current.currentSession).toBeTruthy();
  });

  it('should handle device disconnection', async () => {
    const { result } = renderHook(() => useCastStore());
    
    // First connect
    await act(async () => {
      await result.current.connectToDevice('test-device');
    });
    
    // Then disconnect
    await act(async () => {
      await result.current.disconnectFromDevice('test-device');
    });
    
    expect(result.current.isConnected).toBe(false);
    expect(result.current.currentSession).toBeNull();
  });

  it('should handle media casting', async () => {
    const { result } = renderHook(() => useCastStore());
    
    const mockSong = {
      sourceId: 'test-video-id',
      title: 'Test Song',
      artist: 'Test Artist',
      thumbnailUrl: 'https://example.com/thumb.jpg',
      duration: 180
    };
    
    // First connect to enable casting
    await act(async () => {
      await result.current.connectToDevice('test-device');
    });
    
    // Then cast media
    await act(async () => {
      await result.current.castCurrentSong(mockSong);
    });
    
    // Should not throw
    expect(result.current.lastError).toBeNull();
  });

  it('should handle playback state sync', async () => {
    const { result } = renderHook(() => useCastStore());
    
    const mockState = {
      isPlaying: true,
      positionMs: 30000,
      currentSong: { id: 'test' }
    };
    
    // First connect
    await act(async () => {
      await result.current.connectToDevice('test-device');
    });
    
    // Then sync state
    await act(async () => {
      await result.current.syncPlaybackState(mockState);
    });
    
    // Should not throw
    expect(result.current.lastError).toBeNull();
  });

  it('should handle error clearing', () => {
    const { result } = renderHook(() => useCastStore());
    
    // Set an error
    act(() => {
      useCastStore.setState({
        lastError: {
          code: 'TEST_ERROR',
          description: 'Test error'
        }
      });
    });
    
    expect(result.current.lastError).toBeTruthy();
    
    // Clear the error
    act(() => {
      result.current.clearError();
    });
    
    expect(result.current.lastError).toBeNull();
  });

  it('should handle cleanup', () => {
    const { result } = renderHook(() => useCastStore());
    
    act(() => {
      result.current.cleanup();
    });
    
    expect(result.current.isInitialized).toBe(false);
    expect(result.current.availableDevices).toEqual([]);
    expect(result.current.currentSession).toBeNull();
    expect(result.current.isConnected).toBe(false);
  });
});