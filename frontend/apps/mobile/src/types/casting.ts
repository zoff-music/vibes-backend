// Google Cast types and interfaces for frontend-only casting implementation

export type CastDeviceType = 'chromecast' | 'airplay' | 'dlna';

export type CastSessionState = 
  | 'connecting' 
  | 'connected' 
  | 'syncing' 
  | 'error' 
  | 'disconnected';

export interface CastDevice {
  id: string;
  name: string;
  type: CastDeviceType;
  capabilities: string[];
  isAvailable: boolean;
  lastSeen: Date;
}

export interface CastSession {
  id: string;
  deviceId: string;
  deviceName: string;
  deviceType: CastDeviceType;
  state: CastSessionState;
  startedAt: Date;
  lastSyncAt?: Date;
  mediaSessionId?: string;
}

export interface MediaInfo {
  contentId: string;
  contentType: string;
  streamType: 'BUFFERED' | 'LIVE';
  metadata: {
    title: string;
    artist?: string;
    albumArtist?: string;
    albumName?: string;
    images?: Array<{
      url: string;
      height?: number;
      width?: number;
    }>;
  };
  duration?: number;
}

export interface CastError {
  code: string;
  description: string;
  details?: any;
}

// Google Cast SDK interfaces
export interface CastManager {
  // Device Discovery
  discoverDevices(): Promise<CastDevice[]>;
  getAvailableDevices(): CastDevice[];
  
  // Connection Management
  connectToDevice(deviceId: string): Promise<CastSession>;
  disconnectFromDevice(deviceId: string): Promise<void>;
  
  // Playback Control
  castMedia(mediaInfo: MediaInfo): Promise<void>;
  updateQueue(queue: any[]): Promise<void>;
  syncPlaybackState(state: any): Promise<void>;
  
  // Event Handling
  onDeviceAvailable(callback: (device: CastDevice) => void): void;
  onSessionStateChange(callback: (session: CastSession) => void): void;
  onCastError(callback: (error: CastError) => void): void;
}

// Google Cast Web SDK types
declare global {
  interface Window {
    chrome: {
      cast: {
        isAvailable: boolean;
        initialize: (
          apiConfig: chrome.cast.ApiConfig,
          onInitSuccess: () => void,
          onInitError: (error: chrome.cast.Error) => void
        ) => void;
        requestSession: (
          onSuccess: (session: chrome.cast.Session) => void,
          onError: (error: chrome.cast.Error) => void
        ) => void;
        Session: any;
        media: {
          Media: any;
          MediaInfo: any;
          GenericMediaMetadata: any;
        };
        ApiConfig: any;
        SessionRequest: any;
        AutoJoinPolicy: {
          TAB_AND_ORIGIN_SCOPED: string;
          ORIGIN_SCOPED: string;
          PAGE_SCOPED: string;
        };
        Capability: {
          VIDEO_OUT: string;
          AUDIO_OUT: string;
        };
        DefaultActionPolicy: {
          CREATE_SESSION: string;
          CAST_THIS_TAB: string;
        };
        Error: any;
        ErrorCode: {
          API_NOT_INITIALIZED: string;
          CANCEL: string;
          CHANNEL_ERROR: string;
          EXTENSION_NOT_COMPATIBLE: string;
          EXTENSION_MISSING: string;
          INVALID_PARAMETER: string;
          LOAD_MEDIA_FAILED: string;
          RECEIVER_UNAVAILABLE: string;
          SESSION_ERROR: string;
          TIMEOUT: string;
        };
      };
    };
  }
}