import type {
  CastDevice,
  CastError,
  CastManager,
  CastSession,
  CastSessionState,
  MediaInfo,
} from '@vibez/models';
import { getToken, safeWrap, safeWrapAsync, useRoomStore } from '@vibez/shared';

// const CAST_APP_ID = '333649E5'; // Receiver App ID - Custom Vibez Receiver
// Google Cast Application ID - Custom Vibez Receiver
// For development, we use the Styled Media Receiver which allows custom content
// In production, this should be replaced with a registered custom receiver app ID
const CAST_APPLICATION_ID = import.meta?.env?.VITE_CAST_APP_ID || '1FAF5D9F'; // Custom Vibez Receiver

// Development: Use local custom receiver
// Production: Use registered custom receiver
const DEVELOPMENT_MODE = import.meta.env.VITE_DEVELOPMENT_MODE === 'true';
const CUSTOM_RECEIVER_URL =
  import.meta?.env?.VITE_CAST_RECEIVER_URL || '/casting/receiver/';
const LOCAL_EMULATOR_ENABLED = (() => {
  const envValue = import.meta?.env?.VITE_CAST_LOCAL_EMULATOR;
  if (envValue === 'true' || envValue === '1') return true;
  if (typeof window === 'undefined') return false;
  return (
    window.location.hostname === 'localhost' ||
    window.location.hostname === '127.0.0.1'
  );
})();
const LOCAL_EMULATOR_DEVICE_ID = 'local-cast-emulator';

type LocalCastMessage =
  | {
      action: 'updatePlayback';
      currentSong: {
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      };
      isPlaying: boolean;
      positionMs: number;
      queue: Array<{
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      }>;
      roomInfo: {
        name: string;
        participantCount: number;
      };
      timestamp: number;
    }
  | {
      action: 'updateQueue';
      queue: Array<{
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      }>;
      timestamp: number;
    }
  | {
      action: 'updateRoomInfo';
      roomInfo: {
        name: string;
        participantCount: number;
      };
      timestamp: number;
    }
  | {
      action: 'syncPlayback';
      isPlaying: boolean;
      positionMs: number;
      currentSong?: {
        id: string;
        title: string;
        artist: string;
        sourceType: string;
        sourceId: string;
        thumbnailUrl?: string;
        duration?: number;
      };
      timestamp: number;
    };

class GoogleCastManager implements CastManager {
  private devices: CastDevice[] = [];
  private currentSession: CastSession | null = null;
  private actualCastSession: any = null; // Store the actual Cast SDK session object
  private localReceiverWindow: Window | null = null;
  private localReceiverOrigin: string | null = null;
  private localReceiverFrame: HTMLIFrameElement | null = null;
  private localMessageQueue: LocalCastMessage[] = [];
  private localReceiverReady = false;
  private isInitialized = false;
  private initializationPromise: Promise<void> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 3;
  private reconnectDelay = 1000; // Start with 1 second

  private deviceAvailableCallbacks: Array<(device: CastDevice) => void> = [];
  private sessionStateCallbacks: Array<(session: CastSession) => void> = [];
  private errorCallbacks: Array<(error: CastError) => void> = [];

  constructor() {
    if (typeof window !== 'undefined') {
      this.initializeCastSDK();
    }
  }

  private async initializeCastSDK(): Promise<void> {
    if (typeof window === 'undefined') return;
    console.log('[Cast] initializeCastSDK:start', {
      sdkAvailable: !!window.chrome?.cast?.isAvailable,
      sdkLoaded: !!window.chrome?.cast,
    });
    if (this.initializationPromise) return this.initializationPromise;

    this.initializationPromise = (async () => {
      if (this.isInitialized) return;

      const [loadErr] = await safeWrapAsync(
        new Promise<void>((resolve, reject) => {
          if (window.chrome?.cast?.isAvailable) {
            resolve();
            return;
          }

          if (!window.chrome?.cast) {
            const script = document.createElement('script');
            script.src =
              'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1';
            script.onload = () => {
              console.log('[Cast] sender SDK script loaded');
              this.waitForCastAPI().then(resolve).catch(reject);
            };
            script.onerror = () => {
              const error = new Error('Failed to load Google Cast SDK');
              console.error('[Cast] sender SDK script failed to load');
              this.notifyError({
                code: 'SDK_LOAD_FAILED',
                description: 'Failed to load Google Cast SDK',
                details: error,
              });
              reject(error);
            };
            document.head.appendChild(script);
          } else {
            this.waitForCastAPI().then(resolve).catch(reject);
          }
        }),
      );

      if (loadErr) {
        this.initializationPromise = null;
        throw loadErr;
      }

      const [setupErr] = safeWrap(() => this.setupCastAPI());
      if (setupErr) {
        this.initializationPromise = null;
        throw setupErr;
      }

      this.isInitialized = true;
      this.reconnectAttempts = 0;
      console.log('✅ Google Cast initialized successfully');
      console.log('[Cast] Cast API version:', window.chrome.cast.VERSION);
    })();

    return this.initializationPromise;
  }

  private waitForCastAPI(): Promise<void> {
    return new Promise((resolve, reject) => {
      let attempts = 0;
      const maxAttempts = 100; // 10 seconds with 100ms intervals

      console.log('[Cast] waiting for Cast API availability...');
      const checkInterval = setInterval(() => {
        attempts++;

        if (window.chrome?.cast?.isAvailable) {
          clearInterval(checkInterval);
          console.log('[Cast] Cast API available', { attempts });
          resolve();
        } else if (attempts >= maxAttempts) {
          clearInterval(checkInterval);
          const error: any = new Error(
            'Google Cast API not available after timeout',
          );
          error.code = 'API_TIMEOUT_INTERNAL';
          // Don't notify error for timeout, just log warning
          console.warn(
            '[Cast] API timeout - Cast likely not supported or extension missing',
          );
          reject(error);
        }
      }, 100);
    });
  }

  private setupCastAPI(): void {
    const [_, err] = safeWrap(() => {
      console.log('Setting up Google Cast API...');
      console.log('[Cast] setup config', {
        appId: CAST_APPLICATION_ID,
        receiverUrl: CUSTOM_RECEIVER_URL,
        developmentMode: DEVELOPMENT_MODE,
      });

      const sessionRequest = new window.chrome.cast.SessionRequest(
        CAST_APPLICATION_ID,
      );
      console.log('Created session request for app ID:', CAST_APPLICATION_ID);

      const apiConfig = new window.chrome.cast.ApiConfig(
        sessionRequest,
        this.onSessionListener.bind(this),
        this.onReceiverListener.bind(this),
        window.chrome.cast.AutoJoinPolicy.TAB_AND_ORIGIN_SCOPED,
        window.chrome.cast.DefaultActionPolicy.CREATE_SESSION,
      );
      console.log('Created API config');

      window.chrome.cast.initialize(
        apiConfig,
        () => {
          // This success callback is for the `initialize` call itself, not the full SDK init.
          // The full SDK init success is handled in initializeCastSDK's promise resolution.
          console.log('Google Cast API `initialize` call successful.');
        },
        (error: any) => {
          console.error('❌ Google Cast initialization failed:', error);
          this.notifyError({
            code: 'INITIALIZATION_FAILED',
            description: 'Failed to initialize Google Cast',
            details: error,
          });

          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        },
      );
    });

    if (err) {
      console.error('Error setting up Cast API:', err);
      this.notifyError({
        code: 'SETUP_FAILED',
        description: 'Failed to set up Google Cast API',
        details: err,
      });
    }
  }

  private scheduleReconnect(): void {
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * 2 ** (this.reconnectAttempts - 1); // Exponential backoff

    console.log(
      `Scheduling Cast SDK reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`,
    );

    setTimeout(() => {
      this.initializationPromise = null; // Reset initialization promise
      const [err] = safeWrap(() => {
        this.initializeCastSDK();
      });
      if (err) console.error('Reconnect attempt failed:', err);
    }, delay);
  }

  private onMediaMessage(namespace: string, message: string): void {
    console.log('Media message received:', namespace, message);
  }

  private onCustomMessage(_namespace: string, message: string): void {
    const [err] = safeWrap(() => {
      const data = JSON.parse(message);
      if (data.action === 'LOG') {
        const { level, args } = data;
        const prefix = '%c[RECEIVER]';
        const style =
          'background: #222; color: #bada55; font-weight: bold; padding: 2px 4px; border-radius: 2px;';

        const logArgs = args.map((arg: string) => {
          try {
            // Try to parse back objects if possible, otherwise leave as string
            return JSON.parse(arg);
          } catch {
            return arg;
          }
        });

        switch (level) {
          case 'error':
            console.error(prefix, style, ...logArgs);
            break;
          case 'warn':
            console.warn(prefix, style, ...logArgs);
            break;
          case 'debug':
            console.debug(prefix, style, ...logArgs);
            break;
          default:
            console.log(prefix, style, ...logArgs);
        }
      }
    });

    if (err) {
      console.error('Failed to process custom cast message', err);
    }
  }

  private onSessionListener(session: any): void {
    console.log('🎯 Cast session established:', session);
    console.log('[Cast] session details', {
      sessionId: session?.sessionId || session?.getSessionId?.(),
      receiverName: session?.receiver?.friendlyName,
      mediaCount: session?.media?.length || 0,
    });

    const [_, err] = safeWrap(() => {
      const castSession: CastSession = {
        id: session.sessionId || session.getSessionId?.(),
        deviceId: session.receiver?.friendlyName || 'unknown-device',
        deviceName: session.receiver?.friendlyName || 'Unknown Device',
        deviceType: 'chromecast',
        state: 'connected',
        startedAt: new Date(),
        lastSyncAt: new Date(),
        mediaSessionId: session.media?.[0]?.sessionId,
      };

      console.log('📱 Created cast session object:', castSession);

      this.currentSession = castSession;
      this.actualCastSession = session;
      this.reconnectAttempts = 0;
      this.notifySessionStateChange(castSession);

      if (session.addUpdateListener) {
        session.addUpdateListener(this.onSessionUpdateListener.bind(this));
      }

      if (session.addMessageListener) {
        session.addMessageListener(
          'urn:x-cast:com.google.cast.media',
          this.onMediaMessage.bind(this),
        );
        session.addMessageListener(
          'urn:x-cast:com.vibez.cast',
          this.onCustomMessage.bind(this),
        );
      }

      if (session.media?.[0]) {
        this.handleExistingMediaSession(session.media[0]);
      }
    });

    if (err) {
      console.error('❌ Error handling session listener:', err);
      this.notifyError({
        code: 'SESSION_HANDLER_ERROR',
        description: 'Error processing cast session',
        details: err,
      });
    }
  }

  private handleExistingMediaSession(media: any): void {
    const [_, err] = safeWrap(() => {
      console.log('[Cast] existing media session detected', {
        sessionId: media?.sessionId,
        mediaStatus: media?.playerState,
      });
      if (this.currentSession) {
        this.currentSession.mediaSessionId = media.sessionId;
        this.currentSession.lastSyncAt = new Date();
        this.notifySessionStateChange(this.currentSession);
      }
      media.addUpdateListener(this.onMediaUpdateListener.bind(this));
    });

    if (err) console.error('Error handling existing media session:', err);
  }

  private onMediaUpdateListener(isAlive: boolean): void {
    console.log('[Cast] media update listener', { isAlive });
    if (!isAlive && this.currentSession) {
      console.log('Media session ended');
      this.currentSession.mediaSessionId = undefined;
      this.currentSession.lastSyncAt = new Date();
      this.notifySessionStateChange(this.currentSession);
    }
  }

  private onSessionUpdateListener(isAlive: boolean): void {
    console.log('[Cast] session update listener', { isAlive });
    if (!isAlive && this.currentSession) {
      console.log('Cast session ended');
      this.currentSession.state = 'disconnected';
      this.currentSession.lastSyncAt = new Date();
      this.notifySessionStateChange(this.currentSession);
      this.currentSession = null;
      this.actualCastSession = null; // Clear the stored session
    }
  }

  private onReceiverListener(availability: string): void {
    console.log('Cast receiver availability changed:', availability);

    const [_, err] = safeWrap(() => {
      const isAvailable =
        availability === 'available' ||
        availability === window.chrome?.cast?.ReceiverAvailability?.AVAILABLE;

      if (!isAvailable) {
        console.log('No Chromecast devices available');
        this.devices = [];
        if (LOCAL_EMULATOR_ENABLED) {
          this.ensureLocalEmulatorDevice();
        } else {
          console.log('[Cast] cleared device list');
        }
        return;
      }

      console.log('Chromecast devices are available on the network');
      const device: CastDevice = {
        id: 'chromecast-available',
        name: 'Cast to TV',
        type: 'chromecast',
        capabilities: ['video_out', 'audio_out'],
        isAvailable: true,
        lastSeen: new Date(),
      };

      this.devices = [device];
      if (LOCAL_EMULATOR_ENABLED) {
        this.ensureLocalEmulatorDevice();
      }
      console.log('[Cast] device available', device);
      this.notifyDeviceAvailable(device);
    });

    if (err) {
      console.error('Error handling receiver availability:', err);
      this.notifyError({
        code: 'RECEIVER_HANDLER_ERROR',
        description: 'Error processing receiver availability',
        details: err,
      });
    }
  }

  // Public API methods
  async discoverDevices(): Promise<CastDevice[]> {
    if (LOCAL_EMULATOR_ENABLED) {
      this.ensureLocalEmulatorDevice();
    }
    if (!this.isInitialized) {
      const [initErr] = await safeWrapAsync(this.initializeCastSDK());
      if (initErr) {
        if ((initErr as any).code === 'API_TIMEOUT_INTERNAL') {
          console.warn(
            'Cast initialization timed out - likely unsupported browser. Skipping discovery error.',
          );
          return LOCAL_EMULATOR_ENABLED ? [...this.devices] : [];
        }

        console.error('Failed to discover devices:', initErr);
        this.notifyError({
          code: 'DISCOVERY_FAILED',
          description: 'Failed to discover casting devices',
          details: initErr,
        });
        return LOCAL_EMULATOR_ENABLED ? [...this.devices] : [];
      }
    }
    return [...this.devices];
  }

  getAvailableDevices(): CastDevice[] {
    if (LOCAL_EMULATOR_ENABLED) {
      this.ensureLocalEmulatorDevice();
    }
    return [...this.devices]; // Return a copy to prevent external modification
  }

  prepareLocalReceiverWindow(): boolean {
    if (!LOCAL_EMULATOR_ENABLED) {
      return false;
    }

    const receiverWindow = this.openLocalReceiver();
    if (!receiverWindow) {
      const error = new Error('Local cast receiver window could not be opened');
      this.notifyError({
        code: 'LOCAL_RECEIVER_BLOCKED',
        description: 'Failed to open local cast receiver window',
        details: error,
      });
      return false;
    }

    return true;
  }

  async connectToDevice(deviceId: string): Promise<CastSession> {
    if (LOCAL_EMULATOR_ENABLED && deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      const receiverWindow =
        this.localReceiverWindow && !this.localReceiverWindow.closed
          ? this.localReceiverWindow
          : null;
      if (!receiverWindow) {
        const error = new Error(
          'Local cast receiver window could not be opened',
        );
        this.notifyError({
          code: 'LOCAL_RECEIVER_BLOCKED',
          description: 'Failed to open local cast receiver window',
          details: error,
        });
        throw error;
      }

      const castSession: CastSession = {
        id: `local-${Date.now()}`,
        deviceId: LOCAL_EMULATOR_DEVICE_ID,
        deviceName: 'Local Cast (Emulator)',
        deviceType: 'chromecast',
        state: 'connected',
        startedAt: new Date(),
        lastSyncAt: new Date(),
        mediaSessionId: 'local-session',
      };

      this.currentSession = castSession;
      this.actualCastSession = null;
      this.notifySessionStateChange(castSession);
      return castSession;
    }

    if (!this.isInitialized) throw new Error('Cast SDK not initialized');

    const device = this.devices.find((d) => d.id === deviceId);
    if (!device) throw new Error(`Device ${deviceId} not found`);

    if (this.currentSession && this.currentSession.deviceId === deviceId) {
      console.log('[Cast] already connected to device', deviceId);
      return this.currentSession;
    }

    console.log('🔗 Requesting cast session for device:', deviceId);

    return new Promise((resolve, reject) => {
      window.chrome.cast.requestSession(
        (session: any) => {
          const [err, res] = safeWrap(() => {
            console.log('✅ Session created successfully:', session);
            const castSession: CastSession = {
              id: session.sessionId || session.getSessionId?.(),
              deviceId: deviceId,
              deviceName: session.receiver?.friendlyName || 'Unknown Device',
              deviceType: 'chromecast',
              state: 'connected',
              startedAt: new Date(),
              lastSyncAt: new Date(),
            };

            this.currentSession = castSession;
            this.actualCastSession = session;
            this.reconnectAttempts = 0;
            console.log('[Cast] stored session', {
              sessionId: castSession.id,
              deviceName: castSession.deviceName,
            });

            if (session.addUpdateListener) {
              session.addUpdateListener(
                this.onSessionUpdateListener.bind(this),
              );
            }

            this.notifySessionStateChange(castSession);
            return castSession;
          });

          if (err) reject(err);
          else resolve(res!);
        },
        (error: any) => {
          const errMsg =
            error?.description || error?.message || 'Unknown error';
          console.error('Failed to connect to device:', error);
          this.notifyError({
            code: error?.code || 'CONNECTION_FAILED',
            description: `Failed to connect to device: ${errMsg}`,
            details: error,
          });
          reject(new Error(`Failed to connect to device: ${errMsg}`));
        },
      );
    });
  }

  async disconnectFromDevice(deviceId: string): Promise<void> {
    if (!this.currentSession || this.currentSession.deviceId !== deviceId) {
      return;
    }

    if (deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      if (this.localReceiverWindow && !this.localReceiverWindow.closed) {
        this.localReceiverWindow.close();
      }
      this.localReceiverWindow = null;
      this.localReceiverOrigin = null;
      if (this.localReceiverFrame) {
        this.localReceiverFrame.remove();
      }
      this.localReceiverFrame = null;
      this.localReceiverReady = false;
      this.localMessageQueue = [];
      this.currentSession.state = 'disconnected';
      this.currentSession.lastSyncAt = new Date();
      this.notifySessionStateChange(this.currentSession);
      this.currentSession = null;
      this.actualCastSession = null;
      return;
    }

    const session = this.actualCastSession;
    if (!session) {
      this.currentSession = null;
      this.actualCastSession = null;
      return;
    }

    return new Promise((resolve) => {
      session.stop(
        () => {
          console.log('Session stopped successfully');
          if (this.currentSession) {
            this.currentSession.state = 'disconnected';
            this.currentSession.lastSyncAt = new Date();
            this.notifySessionStateChange(this.currentSession);
          }
          this.currentSession = null;
          this.actualCastSession = null;
          resolve();
        },
        (error: any) => {
          console.error('Failed to stop session:', error);
          if (this.currentSession) {
            this.currentSession.state = 'error';
            this.currentSession.lastSyncAt = new Date();
            this.notifySessionStateChange(this.currentSession);
          }
          this.currentSession = null;
          this.actualCastSession = null;

          this.notifyError({
            code: 'DISCONNECT_FAILED',
            description: 'Failed to properly disconnect from device',
            details: error,
          });
          resolve();
        },
      );
    });
  }

  async castMedia(mediaInfo: MediaInfo): Promise<void> {
    if (!this.currentSession) throw new Error('No active cast session');
    if (this.currentSession.state !== 'connected') {
      throw new Error(`Cast session not ready: ${this.currentSession.state}`);
    }

    if (this.currentSession.deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      this.sendLocalPlaybackState(mediaInfo);
      return;
    }

    const session = this.actualCastSession;
    if (!session) {
      console.error('No actual cast session stored');
      throw new Error('Cast session not found');
    }

    console.log('🎬 Using stored cast session for media loading:', session);
    console.log('[Cast] castMedia payload', {
      contentId: mediaInfo.contentId,
      contentType: mediaInfo.contentType,
      title: mediaInfo.metadata.title,
      sourceType: mediaInfo.metadata.artist ? 'audio' : 'unknown',
    });

    return new Promise((resolve, reject) => {
      if (DEVELOPMENT_MODE || this.isYouTubeUrl(mediaInfo.contentId)) {
        console.log('🎵 Loading custom Vibez receiver');
        this.loadCustomReceiver(mediaInfo, session, resolve, reject);
        return;
      }

      this.loadStandardMedia(mediaInfo, session, resolve, reject);
    });
  }

  private loadCustomReceiver(
    mediaInfo: MediaInfo,
    session: any,
    resolve: () => void,
    reject: (error: Error) => void,
  ): void {
    const [err] = safeWrap(() => {
      console.log('[Cast] loadCustomReceiver:start', {
        receiverUrl: CUSTOM_RECEIVER_URL,
        sessionId: session?.sessionId || session?.getSessionId?.(),
      });
      // Create media info for our custom receiver HTML page
      const receiverMediaInfo = new window.chrome.cast.media.MediaInfo(
        CUSTOM_RECEIVER_URL,
        'text/html',
      );

      // Set up metadata that will be passed to the receiver
      const metadata = new window.chrome.cast.media.GenericMediaMetadata();
      metadata.title = 'ノリ nori Cast Receiver';
      metadata.subtitle = 'Loading...';

      receiverMediaInfo.metadata = metadata;
      receiverMediaInfo.streamType =
        window.chrome.cast.media.StreamType.BUFFERED;

      // Store the original media info to send to receiver once loaded
      receiverMediaInfo.customData = {
        originalMedia: {
          contentId: mediaInfo.contentId,
          contentType: mediaInfo.contentType,
          metadata: mediaInfo.metadata,
          duration: mediaInfo.duration,
          isYouTube: this.isYouTubeUrl(mediaInfo.contentId),
          videoId: this.isYouTubeUrl(mediaInfo.contentId)
            ? this.extractYouTubeVideoId(mediaInfo.contentId)
            : null,
        },
        tokens: {
          spotify: { token: getToken('spotify') },
          soundcloud: { token: getToken('soundcloud') },
        },
        debug:
          new URLSearchParams(window.location.search).get('debug') === 'true',
      };

      const request = new window.chrome.cast.media.LoadRequest(
        receiverMediaInfo,
      );

      console.log('🎬 Loading custom receiver with data:', {
        receiverUrl: CUSTOM_RECEIVER_URL,
        originalContent: mediaInfo.contentId,
        isYouTube: this.isYouTubeUrl(mediaInfo.contentId),
        customData: receiverMediaInfo.customData,
      });

      session.loadMedia(
        request,
        (media: any) => {
          console.log('✅ Custom receiver loaded successfully:', media);
          console.log('[Cast] custom receiver media session', {
            mediaSessionId: media?.sessionId,
          });

          // Wait a moment for receiver to initialize, then send the actual content
          setTimeout(() => {
            this.sendMediaToReceiver(mediaInfo, session);
          }, 2000);

          if (this.currentSession) {
            this.currentSession.mediaSessionId = media.sessionId;
            this.currentSession.state = 'connected';
            this.currentSession.lastSyncAt = new Date();
            this.notifySessionStateChange(this.currentSession);
          }
          resolve();
        },
        (error: any) => {
          const errorMessage =
            error?.description || error?.message || 'Unknown error';
          console.error('❌ Failed to load custom receiver:', error);

          this.notifyError({
            code: error?.code || 'RECEIVER_LOAD_FAILED',
            description: `Failed to load custom receiver: ${errorMessage}`,
            details: error,
          });

          reject(new Error(`Failed to load custom receiver: ${errorMessage}`));
        },
      );
    });

    if (err) {
      console.error('Error loading custom receiver:', err);
      reject(err);
    }
  }

  private sendMediaToReceiver(mediaInfo: MediaInfo, session: any): void {
    // Instead of sending individual media, send the current playback state
    // This includes the current song and queue information
    const [err] = safeWrap(() => {
      const message = {
        action: 'updatePlayback',
        currentSong: {
          id: mediaInfo.metadata.title || 'unknown',
          title: mediaInfo.metadata.title || 'Unknown Title',
          artist: mediaInfo.metadata.artist || 'Unknown Artist',
          sourceType: this.isYouTubeUrl(mediaInfo.contentId)
            ? 'youtube'
            : 'other',
          sourceId: this.isYouTubeUrl(mediaInfo.contentId)
            ? this.extractYouTubeVideoId(mediaInfo.contentId)
            : mediaInfo.contentId,
          duration: mediaInfo.duration || 0,
          thumbnailUrl: mediaInfo.metadata.images?.[0]?.url || '',
        },
        isPlaying: true,
        positionMs: 0,
        queue: [], // Will be updated separately via updateQueue
        roomInfo: {
          name: 'Cast Session',
          participantCount: 1,
        },
      };

      console.log('🎵 Sending playback state to receiver:', {
        action: message.action,
        title: message.currentSong.title,
        sourceType: message.currentSong.sourceType,
        sourceId: message.currentSong.sourceId,
      });

      session.sendMessage(
        'urn:x-cast:com.vibez.cast',
        message,
        () => console.log('✅ Playback state sent to receiver'),
        (error: any) =>
          console.error('❌ Failed to send playback state:', error),
      );
    });

    if (err) console.error('Error sending media to receiver:', err);
  }

  private sendLocalPlaybackState(mediaInfo: MediaInfo): void {
    const [err] = safeWrap(() => {
      const message: LocalCastMessage = {
        action: 'updatePlayback',
        currentSong: {
          id: mediaInfo.metadata.title || 'unknown',
          title: mediaInfo.metadata.title || 'Unknown Title',
          artist: mediaInfo.metadata.artist || 'Unknown Artist',
          sourceType: this.isYouTubeUrl(mediaInfo.contentId)
            ? 'youtube'
            : 'other',
          sourceId: this.isYouTubeUrl(mediaInfo.contentId)
            ? this.extractYouTubeVideoId(mediaInfo.contentId) ||
              mediaInfo.contentId
            : mediaInfo.contentId,
          duration: mediaInfo.duration || 0,
          thumbnailUrl: mediaInfo.metadata.images?.[0]?.url || '',
        },
        isPlaying: true,
        positionMs: 0,
        queue: [],
        roomInfo: {
          name: 'Local Cast',
          participantCount: 1,
        },
        timestamp: Date.now(),
      };

      console.log('[Local Cast] sending playback state', {
        title: message.currentSong.title,
        sourceType: message.currentSong.sourceType,
      });
      this.sendLocalMessage(message);
    });

    if (err) {
      console.error('Error sending local playback state:', err);
    }
  }

  private loadStandardMedia(
    mediaInfo: MediaInfo,
    session: any,
    resolve: () => void,
    reject: (error: Error) => void,
  ): void {
    const [err] = safeWrap(() => {
      console.log('[Cast] loadStandardMedia:start', {
        contentId: mediaInfo.contentId,
        contentType: mediaInfo.contentType,
      });
      // For non-YouTube content, proceed with normal casting
      const castMediaInfo = new window.chrome.cast.media.MediaInfo(
        mediaInfo.contentId,
        mediaInfo.contentType,
      );

      // Set up metadata
      const metadata = new window.chrome.cast.media.GenericMediaMetadata();
      metadata.title = mediaInfo.metadata.title;
      metadata.subtitle = mediaInfo.metadata.artist || '';

      if (mediaInfo.metadata.images && mediaInfo.metadata.images.length > 0) {
        metadata.images = mediaInfo.metadata.images.map((img) => ({
          url: img.url,
          height: img.height || 480,
          width: img.width || 640,
        }));
      }

      castMediaInfo.metadata = metadata;
      castMediaInfo.streamType =
        mediaInfo.streamType === 'LIVE'
          ? window.chrome.cast.media.StreamType.LIVE
          : window.chrome.cast.media.StreamType.BUFFERED;

      if (mediaInfo.duration) {
        castMediaInfo.duration = mediaInfo.duration;
      }

      const request = new window.chrome.cast.media.LoadRequest(castMediaInfo);

      console.log('🎬 Loading standard media:', {
        contentId: mediaInfo.contentId,
        contentType: mediaInfo.contentType,
        title: mediaInfo.metadata.title,
      });

      session.loadMedia(
        request,
        (media: any) => {
          console.log('✅ Standard media loaded successfully:', media);
          if (this.currentSession) {
            this.currentSession.mediaSessionId = media.sessionId;
            this.currentSession.state = 'connected';
            this.currentSession.lastSyncAt = new Date();
            this.notifySessionStateChange(this.currentSession);
          }
          resolve();
        },
        (error: any) => {
          const errorMessage =
            error?.description || error?.message || 'Unknown error';
          console.error('❌ Failed to load standard media:', error);

          this.notifyError({
            code: error?.code || 'MEDIA_LOAD_FAILED',
            description: `Failed to cast media: ${errorMessage}`,
            details: error,
          });

          reject(new Error(`Failed to cast media: ${errorMessage}`));
        },
      );
    });

    if (err) {
      console.error('Error loading standard media:', err);
      reject(err);
    }
  }

  private extractYouTubeVideoId(url: string): string | null {
    const regex =
      /(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([^&\n?#]+)/;
    const match = url.match(regex);
    return match ? match[1] : null;
  }

  private isYouTubeUrl(url: string): boolean {
    return url.includes('youtube.com') || url.includes('youtu.be');
  }

  async updateQueue(queue: any[]): Promise<void> {
    if (!this.currentSession || this.currentSession.state !== 'connected') {
      return;
    }

    if (this.currentSession.deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      const message: LocalCastMessage = {
        action: 'updateQueue',
        queue: queue.map((song) => ({
          id: song.id,
          title: song.title,
          artist: song.artist,
          sourceType: song.sourceType,
          sourceId: song.sourceId,
          thumbnailUrl: song.thumbnailUrl,
          duration: song.duration,
        })),
        timestamp: Date.now(),
      };
      this.sendLocalMessage(message);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const message = {
      action: 'updateQueue',
      queue: queue.map((song) => ({
        id: song.id,
        title: song.title,
        artist: song.artist,
        sourceType: song.sourceType,
        sourceId: song.sourceId,
        thumbnailUrl: song.thumbnailUrl,
        duration: song.duration,
      })),
      timestamp: Date.now(),
    };

    const [err] = safeWrap(() => {
      console.log('[Cast] updateQueue send', {
        count: message.queue.length,
        timestamp: message.timestamp,
      });
      session.sendMessage(
        'urn:x-cast:com.vibez.cast',
        message,
        () => console.log('✅ Queue update sent to receiver'),
        (error: any) => console.error('❌ Failed to update queue:', error),
      );
    });

    if (err) {
      console.error('Error updating queue:', err);
      this.notifyError({
        code: 'QUEUE_UPDATE_FAILED',
        description: 'Failed to update queue on cast device',
        details: err,
      });
    }
  }

  async updateRoomInfo(roomInfo: {
    name: string;
    participantCount: number;
  }): Promise<void> {
    if (!this.currentSession || this.currentSession.state !== 'connected') {
      return;
    }

    if (this.currentSession.deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      const message: LocalCastMessage = {
        action: 'updateRoomInfo',
        roomInfo: roomInfo,
        timestamp: Date.now(),
      };
      this.sendLocalMessage(message);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const message = {
      action: 'updateRoomInfo',
      roomInfo: roomInfo,
      timestamp: Date.now(),
    };

    const [err] = safeWrap(() => {
      console.log('[Cast] updateRoomInfo send', message);
      session.sendMessage(
        'urn:x-cast:com.vibez.cast',
        message,
        () => console.log('✅ Room info sent to receiver'),
        (error: any) => console.error('❌ Failed to update room info:', error),
      );
    });

    if (err) {
      console.error('Error updating room info:', err);
      this.notifyError({
        code: 'ROOM_UPDATE_FAILED',
        description: 'Failed to update room info on cast device',
        details: err,
      });
    }
  }

  async syncPlaybackState(state: any): Promise<void> {
    if (!this.currentSession || this.currentSession.state !== 'connected') {
      return;
    }

    if (this.currentSession.deviceId === LOCAL_EMULATOR_DEVICE_ID) {
      const message: LocalCastMessage = {
        action: 'syncPlayback',
        isPlaying: state.isPlaying,
        positionMs: state.positionMs,
        currentSong: state.currentSong,
        timestamp: Date.now(),
      };
      this.sendLocalMessage(message);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const message = {
      action: 'syncPlayback',
      isPlaying: state.isPlaying,
      positionMs: state.positionMs,
      currentSong: state.currentSong,
      timestamp: Date.now(),
    };

    const [err] = safeWrap(() => {
      console.log('[Cast] syncPlayback send', {
        isPlaying: message.isPlaying,
        positionMs: message.positionMs,
        title: message.currentSong?.title,
        timestamp: message.timestamp,
      });
      session.sendMessage(
        'urn:x-cast:com.vibez.cast',
        message,
        () => console.log('✅ Playback sync sent to receiver'),
        (error: any) => console.error('❌ Failed to sync playback:', error),
      );
    });

    if (err) {
      console.error('Error syncing playback state:', err);
      this.notifyError({
        code: 'SYNC_FAILED',
        description: 'Failed to synchronize playback state with cast device',
        details: err,
      });
    }
  }

  // Event handling
  onDeviceAvailable(callback: (device: CastDevice) => void): void {
    this.deviceAvailableCallbacks.push(callback);
  }

  onSessionStateChange(callback: (session: CastSession) => void): void {
    this.sessionStateCallbacks.push(callback);
  }

  onCastError(callback: (error: CastError) => void): void {
    this.errorCallbacks.push(callback);
  }

  // Private notification methods
  private notifyDeviceAvailable(device: CastDevice): void {
    this.deviceAvailableCallbacks.forEach((callback) => {
      callback(device);
    });
  }

  private notifySessionStateChange(session: CastSession): void {
    this.sessionStateCallbacks.forEach((callback) => {
      callback(session);
    });
  }

  private notifyError(error: CastError): void {
    this.errorCallbacks.forEach((callback) => {
      callback(error);
    });
  }

  // Utility methods
  getCurrentSession(): CastSession | null {
    return this.currentSession ? { ...this.currentSession } : null; // Return copy
  }

  isConnected(): boolean {
    return this.currentSession?.state === 'connected';
  }

  getConnectionState(): CastSessionState | null {
    return this.currentSession?.state || null;
  }

  // Clean up resources
  destroy(): void {
    const [_, err] = safeWrap(() => {
      this.deviceAvailableCallbacks = [];
      this.sessionStateCallbacks = [];
      this.errorCallbacks = [];

      if (this.currentSession) {
        safeWrapAsync(this.disconnectFromDevice(this.currentSession.deviceId));
      }

      this.devices = [];
      this.currentSession = null;
      this.actualCastSession = null;
      if (this.localReceiverWindow && !this.localReceiverWindow.closed) {
        this.localReceiverWindow.close();
      }
      this.localReceiverWindow = null;
      this.localReceiverOrigin = null;
      if (this.localReceiverFrame) {
        this.localReceiverFrame.remove();
      }
      this.localReceiverFrame = null;
      this.localReceiverReady = false;
      this.localMessageQueue = [];
      this.isInitialized = false;
      this.initializationPromise = null;
      this.reconnectAttempts = 0;

      console.log('Cast manager destroyed');
    });

    if (err) console.error('Error during cast manager destruction:', err);
  }

  // Get detailed status for debugging
  getStatus(): {
    isInitialized: boolean;
    deviceCount: number;
    currentSession: CastSession | null;
    reconnectAttempts: number;
  } {
    return {
      isInitialized: this.isInitialized,
      deviceCount: this.devices.length,
      currentSession: this.getCurrentSession(),
      reconnectAttempts: this.reconnectAttempts,
    };
  }

  // Debug method to help troubleshoot casting issues
  getDebugInfo(): {
    sdkLoaded: boolean;
    sdkAvailable: boolean;
    apiVersion: string | undefined;
    receiverAvailability: string | undefined;
    devices: CastDevice[];
    currentSession: CastSession | null;
    isInitialized: boolean;
  } {
    return {
      sdkLoaded: !!window.chrome?.cast,
      sdkAvailable: !!window.chrome?.cast?.isAvailable,
      apiVersion: window.chrome?.cast?.VERSION,
      receiverAvailability: 'unknown', // This would need to be tracked
      devices: this.devices,
      currentSession: this.currentSession,
      isInitialized: this.isInitialized,
    };
  }

  private ensureLocalEmulatorDevice(): void {
    const exists = this.devices.some(
      (device) => device.id === LOCAL_EMULATOR_DEVICE_ID,
    );
    if (exists) return;

    const device: CastDevice = {
      id: LOCAL_EMULATOR_DEVICE_ID,
      name: 'Local Cast (Emulator)',
      type: 'chromecast',
      capabilities: ['video_out', 'audio_out'],
      isAvailable: true,
      lastSeen: new Date(),
    };

    this.devices = [device, ...this.devices];
    this.notifyDeviceAvailable(device);
  }

  private getLocalReceiverUrl(): string {
    return this.getReceiverUrlWithParams();
  }

  private getReceiverUrlWithParams(): string {
    const baseUrl =
      CUSTOM_RECEIVER_URL.startsWith('http://') ||
      CUSTOM_RECEIVER_URL.startsWith('https://')
        ? CUSTOM_RECEIVER_URL
        : `${window.location.origin}${
            CUSTOM_RECEIVER_URL.startsWith('/')
              ? CUSTOM_RECEIVER_URL
              : `/${CUSTOM_RECEIVER_URL}`
          }`;

    const params = new URLSearchParams();
    params.set('castReceiver', '1');

    const roomState = useRoomStore.getState();
    if (roomState.userId) {
      params.set('casterId', roomState.userId);
    }
    if (roomState.room?.id) {
      params.set('roomId', roomState.room.id);
    }

    // Check for debug flag in current window URL and pass it to receiver
    const currentParams = new URLSearchParams(window.location.search);
    if (currentParams.get('debug') === 'true') {
      params.set('debug', 'true');
    }

    const separator = baseUrl.includes('?') ? '&' : '?';
    return `${baseUrl}${separator}${params.toString()}`;
  }

  private openLocalReceiver(): Window | null {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return null;
    }

    const receiverUrl = this.getLocalReceiverUrl();
    const [urlErr, parsedUrl] = safeWrap(() => new URL(receiverUrl));
    if (urlErr || !parsedUrl) {
      console.error('Invalid local receiver URL:', receiverUrl);
      return null;
    }

    this.localReceiverOrigin = parsedUrl.origin;

    const popup =
      this.localReceiverWindow && !this.localReceiverWindow.closed
        ? this.localReceiverWindow
        : (() => {
            const width = 480;
            const height = 270;
            const left = Math.max(0, window.screen.width / 2 - width / 2);
            const top = Math.max(0, window.screen.height / 2 - height / 2);
            const features = [
              `width=${width}`,
              `height=${height}`,
              `left=${left}`,
              `top=${top}`,
              'resizable=yes',
              'scrollbars=no',
            ].join(',');
            return window.open(receiverUrl, 'vibez-cast-receiver', features);
          })();

    if (!popup) {
      console.error('Local cast receiver window was blocked or failed to open');
      return null;
    }

    this.localReceiverFrame = null;
    this.localReceiverWindow = popup;
    this.localReceiverReady = true;
    this.flushLocalMessageQueue();
    return popup;
  }

  private sendLocalMessage(message: LocalCastMessage): void {
    if (!this.localReceiverWindow || this.localReceiverWindow.closed) {
      console.warn('[Local Cast] receiver window not available');
      return;
    }

    this.localMessageQueue.push(message);
    this.flushLocalMessageQueue();
  }

  private flushLocalMessageQueue(): void {
    if (!this.localReceiverWindow || !this.localReceiverReady) return;

    const origin = this.localReceiverOrigin || window.location.origin;
    while (this.localMessageQueue.length > 0) {
      const nextMessage = this.localMessageQueue.shift();
      if (!nextMessage) break;
      const [err] = safeWrap(() => {
        this.localReceiverWindow?.postMessage(nextMessage, origin);
      });
      if (err) {
        console.error('Failed to post local cast message:', err);
        break;
      }
    }
  }

  // Force refresh device discovery
  async forceDiscovery(): Promise<void> {
    console.log('🔍 Forcing device discovery...');
    const [err] = await safeWrapAsync(this.initializeCastSDK());
    if (err) {
      console.error('Failed to initialize SDK during force discovery:', err);
      return;
    }

    if (LOCAL_EMULATOR_ENABLED) {
      this.ensureLocalEmulatorDevice();
    }

    // The Cast SDK doesn't provide a direct way to force discovery
    // Device availability is handled automatically by the SDK
    console.log('Current devices:', this.devices);
  }
}

// Export singleton instance
export const castManager = new GoogleCastManager();
