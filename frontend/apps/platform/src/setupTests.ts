import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock React Player and YouTube to avoid script loading issues in tests
vi.mock('react-player', () => ({
  default: () => null,
}));

vi.mock('react-youtube', () => ({
  default: () => null,
}));

// Mock castManager to avoid Google Cast SDK loading in tests
vi.mock('./services/castManager', () => ({
  castManager: {
    discoverDevices: vi.fn().mockResolvedValue([]),
    getAvailableDevices: vi.fn().mockReturnValue([]),
    connectToDevice: vi.fn().mockResolvedValue(null),
    disconnectFromDevice: vi.fn().mockResolvedValue(undefined),
    castMedia: vi.fn().mockResolvedValue(undefined),
    updateQueue: vi.fn().mockResolvedValue(undefined),
    updateRoomInfo: vi.fn().mockResolvedValue(undefined),
    syncPlaybackState: vi.fn().mockResolvedValue(undefined),
    onDeviceAvailable: vi.fn(),
    onSessionStateChange: vi.fn(),
    onCastError: vi.fn(),
    destroy: vi.fn(),
  },
  default: {
    discoverDevices: vi.fn().mockResolvedValue([]),
    getAvailableDevices: vi.fn().mockReturnValue([]),
    connectToDevice: vi.fn().mockResolvedValue(null),
    disconnectFromDevice: vi.fn().mockResolvedValue(undefined),
    castMedia: vi.fn().mockResolvedValue(undefined),
    updateQueue: vi.fn().mockResolvedValue(undefined),
    updateRoomInfo: vi.fn().mockResolvedValue(undefined),
    syncPlaybackState: vi.fn().mockResolvedValue(undefined),
    onDeviceAvailable: vi.fn(),
    onSessionStateChange: vi.fn(),
    onCastError: vi.fn(),
    destroy: vi.fn(),
  },
}));
