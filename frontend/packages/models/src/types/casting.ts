import * as yup from 'yup';
import {
  castDeviceSchema,
  castDeviceTypeSchema,
  castErrorSchema,
  castSessionSchema,
  castSessionStateSchema,
  mediaInfoSchema,
} from '../schemas/casting';

// Inferred Types
export type CastDeviceType = yup.InferType<typeof castDeviceTypeSchema>;
export type CastSessionState = yup.InferType<typeof castSessionStateSchema>;
export type CastDevice = yup.InferType<typeof castDeviceSchema>;
export type CastSession = yup.InferType<typeof castSessionSchema>;
export type MediaInfo = yup.InferType<typeof mediaInfoSchema>;
export type CastError = Omit<
  yup.InferType<typeof castErrorSchema>,
  'details'
> & { details?: unknown };

// Interfaces (Non-schema types)
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
  updateRoomInfo(roomInfo: {
    name: string;
    participantCount: number;
  }): Promise<void>;
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
        VERSION?: string;
        initialize: (
          apiConfig: chrome.cast.ApiConfig,
          onInitSuccess: () => void,
          onInitError: (error: chrome.cast.Error) => void,
        ) => void;
        requestSession: (
          onSuccess: (session: chrome.cast.Session) => void,
          onError: (error: chrome.cast.Error) => void,
        ) => void;
        Session: any;
        media: {
          Media: any;
          MediaInfo: any;
          GenericMediaMetadata: any;
          LoadRequest: any;
          SeekRequest: any;
          StreamType: {
            BUFFERED: string;
            LIVE: string;
          };
          PlayerState: {
            IDLE: string;
            PLAYING: string;
            PAUSED: string;
            BUFFERING: string;
          };
        };
        ApiConfig: any;
        SessionRequest: any;
        AutoJoinPolicy: {
          TAB_AND_ORIGIN_SCOPED: string;
          ORIGIN_SCOPED: string;
          PAGE_SCOPED: string;
        };
        ReceiverAvailability: {
          AVAILABLE: string;
          UNAVAILABLE: string;
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
        framework?: {
          CastContext: {
            getInstance(): {
              getCurrentSession(): any;
            };
          };
        };
      };
    };
  }

  namespace chrome {
    namespace cast {
      interface ApiConfig {}
      interface Session {}
      interface Error {}
    }
  }
}
