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
  updateQueue(queue: unknown[]): Promise<void>;
  updateRoomInfo(roomInfo: {
    name: string;
    participantCount: number;
  }): Promise<void>;
  syncPlaybackState(state: unknown): Promise<void>;

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
        Session: {
          new (
            sessionId: string,
            appId: string,
            displayName: string,
            appImages: chrome.cast.Image[],
            receiver: chrome.cast.Receiver,
          ): chrome.cast.Session;
        };
        media: {
          Media: {
            new (
              sessionId: string,
              mediaSessionId: number,
            ): chrome.cast.media.Media;
          };
          MediaInfo: {
            new (
              contentId: string,
              contentType: string,
            ): chrome.cast.media.MediaInfo;
          };
          GenericMediaMetadata: {
            new (): chrome.cast.media.GenericMediaMetadata;
          };
          LoadRequest: {
            new (
              mediaInfo: chrome.cast.media.MediaInfo,
            ): chrome.cast.media.LoadRequest;
          };
          SeekRequest: {
            new (): chrome.cast.media.SeekRequest;
          };
          StreamType: {
            BUFFERED: string;
            LIVE: string;
            OTHER: string;
          };
          PlayerState: {
            IDLE: string;
            PLAYING: string;
            PAUSED: string;
            BUFFERING: string;
          };
          MetadataType: {
            GENERIC: number;
            TV_SHOW: number;
            MOVIE: number;
            MUSIC_TRACK: number;
            PHOTO: number;
          };
        };
        ApiConfig: {
          new (
            sessionRequest: chrome.cast.SessionRequest,
            sessionListener: (session: chrome.cast.Session) => void,
            receiverListener: (availability: string) => void,
            autoJoinPolicy?: string,
            defaultActionPolicy?: string,
          ): chrome.cast.ApiConfig;
        };
        SessionRequest: {
          new (
            appId: string,
            capabilities?: string[],
            timeout?: number,
          ): chrome.cast.SessionRequest;
        };
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
        Error: {
          new (
            code: string,
            description?: string,
            details?: unknown,
          ): chrome.cast.Error;
        };
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
              getCurrentSession(): chrome.cast.Session | null;
            };
          };
        };
        Image: {
          new (url: string): chrome.cast.Image;
        };
        Receiver: {
          new (
            label: string,
            friendlyName: string,
            capabilities?: string[],
            volume?: chrome.cast.Volume,
          ): chrome.cast.Receiver;
        };
        Volume: {
          new (level?: number, muted?: boolean): chrome.cast.Volume;
        };
      };
    };
  }

  namespace chrome.cast {
    interface ApiConfig {
      sessionRequest: SessionRequest;
      sessionListener: (session: Session) => void;
      receiverListener: (availability: string) => void;
      autoJoinPolicy: string;
      defaultActionPolicy: string;
    }

    interface Session {
      sessionId: string;
      appId: string;
      displayName: string;
      appImages: Image[];
      receiver: Receiver;
      media: media.Media[];
      status: string;
      statusText: string | null;
      transportId: string;
      addUpdateListener(listener: (isAlive: boolean) => void): void;
      removeUpdateListener(listener: (isAlive: boolean) => void): void;
      addMessageListener(
        namespace: string,
        listener: (namespace: string, message: string) => void,
      ): void;
      removeMessageListener(
        namespace: string,
        listener: (namespace: string, message: string) => void,
      ): void;
      sendMessage(
        namespace: string,
        message: unknown,
        onSuccess?: () => void,
        onError?: (error: Error) => void,
      ): void;
      loadMedia(
        loadRequest: media.LoadRequest,
        onSuccess?: (media: media.Media) => void,
        onError?: (error: Error) => void,
      ): void;
      stop(onSuccess?: () => void, onError?: (error: Error) => void): void;
    }

    interface Error {
      code: string;
      description: string | null;
      details: unknown;
    }

    interface Image {
      url: string;
      height: number | null;
      width: number | null;
    }

    interface Receiver {
      label: string;
      friendlyName: string;
      capabilities: string[];
      volume: Volume;
    }

    interface Volume {
      level: number | null;
      muted: boolean | null;
    }

    interface SessionRequest {
      appId: string;
      capabilities: string[];
      requestSessionTimeout: number;
      language: string | null;
    }

    namespace media {
      interface Media {
        sessionId: string;
        mediaSessionId: number;
        media: MediaInfo;
        playerState: string;
        currentTime: number;
        getEstimatedTime(): number;
        play(
          request?: unknown,
          onSuccess?: () => void,
          onError?: (error: Error) => void,
        ): void;
        pause(
          request?: unknown,
          onSuccess?: () => void,
          onError?: (error: Error) => void,
        ): void;
        seek(
          seekRequest: SeekRequest,
          onSuccess?: () => void,
          onError?: (error: Error) => void,
        ): void;
        stop(
          request?: unknown,
          onSuccess?: () => void,
          onError?: (error: Error) => void,
        ): void;
        addUpdateListener(listener: (isAlive: boolean) => void): void;
      }

      interface MediaInfo {
        contentId: string;
        contentType: string;
        metadata: GenericMediaMetadata;
        streamType: string;
        duration: number | null;
        customData: unknown;
      }

      interface GenericMediaMetadata {
        type: number;
        title: string;
        subtitle: string;
        images: Image[];
        releaseDate: string | null;
      }

      interface LoadRequest {
        media: MediaInfo;
        autoplay: boolean;
        currentTime: number | null;
        customData: unknown;
      }

      interface SeekRequest {
        currentTime: number | null;
        resumeState: string | null;
        customData: unknown;
      }
    }
  }
}
