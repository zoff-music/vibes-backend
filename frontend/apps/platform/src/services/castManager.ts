import type {
  CastDevice,
  CastError,
  CastManager,
  CastSession,
  CastSessionState,
  MediaInfo,
} from '@vibez/models';
import { getToken, safeWrap, safeWrapAsync } from '@vibez/shared';

// Google Cast Application ID - Custom Vibez Receiver
// For development, we use the Styled Media Receiver which allows custom content
// In production, this should be replaced with a registered custom receiver app ID
const CAST_APPLICATION_ID = import.meta.env.VITE_CAST_APP_ID || '1FAF5D9F'; // Custom Vibez Receiver

// Development: Use local custom receiver
// Production: Use registered custom receiver
const DEVELOPMENT_MODE = true;
const CUSTOM_RECEIVER_URL =
  import.meta.env.VITE_CAST_RECEIVER_URL || '/casting/receiver/';

class GoogleCastManager implements CastManager {
  private devices: CastDevice[] = [];
  private currentSession: CastSession | null = null;
  private actualCastSession: any = null; // Store the actual Cast SDK session object
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
              this.waitForCastAPI().then(resolve).catch(reject);
            };
            script.onerror = () => {
              const error = new Error('Failed to load Google Cast SDK');
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
      console.log('Cast API version:', window.chrome.cast.VERSION);
    })();

    return this.initializationPromise;
  }

  private waitForCastAPI(): Promise<void> {
    return new Promise((resolve, reject) => {
      let attempts = 0;
      const maxAttempts = 100; // 10 seconds with 100ms intervals

      const checkInterval = setInterval(() => {
        attempts++;

        if (window.chrome?.cast?.isAvailable) {
          clearInterval(checkInterval);
          resolve();
        } else if (attempts >= maxAttempts) {
          clearInterval(checkInterval);
          const error = new Error(
            'Google Cast API not available after timeout',
          );
          this.notifyError({
            code: 'API_TIMEOUT',
            description:
              'Google Cast API failed to initialize within timeout period',
            details: error,
          });
          reject(error);
        }
      }, 100);
    });
  }

  private setupCastAPI(): void {
    const [_, err] = safeWrap(() => {
      console.log('Setting up Google Cast API...');

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

  private onSessionListener(session: any): void {
    console.log('🎯 Cast session established:', session);

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
    if (!isAlive && this.currentSession) {
      console.log('Media session ended');
      this.currentSession.mediaSessionId = undefined;
      this.currentSession.lastSyncAt = new Date();
      this.notifySessionStateChange(this.currentSession);
    }
  }

  private onSessionUpdateListener(isAlive: boolean): void {
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
    if (!this.isInitialized) {
      const [initErr] = await safeWrapAsync(this.initializeCastSDK());
      if (initErr) {
        console.error('Failed to discover devices:', initErr);
        this.notifyError({
          code: 'DISCOVERY_FAILED',
          description: 'Failed to discover casting devices',
          details: initErr,
        });
        return [];
      }
    }
    return [...this.devices];
  }

  getAvailableDevices(): CastDevice[] {
    return [...this.devices]; // Return a copy to prevent external modification
  }

  async connectToDevice(deviceId: string): Promise<CastSession> {
    if (!this.isInitialized) throw new Error('Cast SDK not initialized');

    const device = this.devices.find((d) => d.id === deviceId);
    if (!device) throw new Error(`Device ${deviceId} not found`);

    if (this.currentSession && this.currentSession.deviceId === deviceId) {
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

    const session = this.actualCastSession;
    if (!session) {
      console.error('No actual cast session stored');
      throw new Error('Cast session not found');
    }

    console.log('🎬 Using stored cast session for media loading:', session);

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
      // Create media info for our custom receiver HTML page
      const receiverMediaInfo = new window.chrome.cast.media.MediaInfo(
        CUSTOM_RECEIVER_URL,
        'text/html',
      );

      // Set up metadata that will be passed to the receiver
      const metadata = new window.chrome.cast.media.GenericMediaMetadata();
      metadata.title = 'Vibez Cast Receiver';
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
    const [err] = safeWrap(() => {
      if (this.isYouTubeUrl(mediaInfo.contentId)) {
        const videoId = this.extractYouTubeVideoId(mediaInfo.contentId);

        const message = {
          action: 'loadYouTube',
          videoId: videoId,
          metadata: {
            title: mediaInfo.metadata.title || 'Unknown Title',
            artist: mediaInfo.metadata.artist || 'Unknown Artist',
            images: mediaInfo.metadata.images || [],
          },
        };

        console.log('🎵 Sending YouTube content to receiver:', message);

        session.sendMessage(
          'urn:x-cast:vibez.media',
          message,
          () => console.log('✅ YouTube content sent to receiver'),
          (error: any) =>
            console.error('❌ Failed to send YouTube content:', error),
        );
      } else {
        // For non-YouTube content, send standard media info
        const message = {
          action: 'loadMedia',
          contentId: mediaInfo.contentId,
          contentType: mediaInfo.contentType,
          metadata: mediaInfo.metadata,
        };

        console.log('🎬 Sending standard media to receiver:', message);

        session.sendMessage(
          'urn:x-cast:vibez.media',
          message,
          () => console.log('✅ Media content sent to receiver'),
          (error: any) =>
            console.error('❌ Failed to send media content:', error),
        );
      }
    });

    if (err) console.error('Error sending media to receiver:', err);
  }

  private loadStandardMedia(
    mediaInfo: MediaInfo,
    session: any,
    resolve: () => void,
    reject: (error: Error) => void,
  ): void {
    const [err] = safeWrap(() => {
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
    const [_, err] = safeWrap(() => {
      if (!this.isConnected() || !this.actualCastSession) return;

      const queueMessage = {
        songs: queue.map((song) => ({
          id: song.id,
          title: song.title,
          artist: song.artist,
          thumbnailUrl: song.thumbnailUrl,
          duration: song.duration,
        })),
      };

      console.log('Sending queue update to receiver:', queueMessage);
      this.actualCastSession.sendMessage(
        'urn:x-cast:vibez.queue',
        queueMessage,
        () => console.log('Queue updated on receiver'),
        (error: any) => console.error('Failed to update queue:', error),
      );
    });

    if (err) {
      console.error('Failed to update queue:', err);
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
    const [_, err] = safeWrap(() => {
      if (!this.isConnected() || !this.actualCastSession) return;

      console.log('Sending room info to receiver:', roomInfo);
      this.actualCastSession.sendMessage(
        'urn:x-cast:vibez.room',
        roomInfo,
        () => console.log('Room info updated on receiver'),
        (error: any) => console.error('Failed to update room info:', error),
      );
    });

    if (err) {
      console.error('Failed to update room info:', err);
      this.notifyError({
        code: 'ROOM_UPDATE_FAILED',
        description: 'Failed to update room info on cast device',
        details: err,
      });
    }
  }

  async syncPlaybackState(state: any): Promise<void> {
    const [_, err] = await safeWrapAsync(
      (async () => {
        if (!this.isConnected() || !this.actualCastSession) return;
        const session = this.actualCastSession;
        if (!session.media?.[0]) return;

        const media = session.media[0];

        // Sync play/pause
        if (
          state.isPlaying &&
          media.playerState === window.chrome.cast.media.PlayerState.PAUSED
        ) {
          await new Promise<void>((res, rej) => media.play(null, res, rej));
        } else if (
          !state.isPlaying &&
          media.playerState === window.chrome.cast.media.PlayerState.PLAYING
        ) {
          await new Promise<void>((res, rej) => media.pause(null, res, rej));
        }

        // Sync position
        if (state.positionMs && typeof state.positionMs === 'number') {
          const currentTimeMs = (media.currentTime || 0) * 1000;
          if (Math.abs(currentTimeMs - state.positionMs) > 2000) {
            const seekRequest = new window.chrome.cast.media.SeekRequest();
            seekRequest.currentTime = state.positionMs / 1000;
            await new Promise<void>((res, rej) =>
              media.seek(seekRequest, res, rej),
            );
          }
        }

        if (this.currentSession) {
          this.currentSession.lastSyncAt = new Date();
          this.notifySessionStateChange(this.currentSession);
        }
      })(),
    );

    if (err) {
      console.error('Failed to sync playback state:', err);
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

  // Force refresh device discovery
  async forceDiscovery(): Promise<void> {
    console.log('🔍 Forcing device discovery...');
    const [err] = await safeWrapAsync(this.initializeCastSDK());
    if (err) {
      console.error('Failed to initialize SDK during force discovery:', err);
      return;
    }

    // The Cast SDK doesn't provide a direct way to force discovery
    // Device availability is handled automatically by the SDK
    console.log('Current devices:', this.devices);
  }
}

// Export singleton instance
export const castManager = new GoogleCastManager();
