import type {
  CastDevice,
  CastError,
  CastManager,
  CastSession,
  CastSessionState,
  MediaInfo,
} from '@vibez/models';
import { getToken } from '@vibez/shared';

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
    this.initializeCastSDK();
  }

  private async initializeCastSDK(): Promise<void> {
    // Return existing promise if initialization is already in progress
    if (this.initializationPromise) {
      return this.initializationPromise;
    }

    this.initializationPromise = new Promise((resolve, reject) => {
      // Check if Cast SDK is already loaded
      if (window.chrome?.cast?.isAvailable) {
        this.setupCastAPI();
        this.isInitialized = true;
        resolve();
        return;
      }

      // Load Google Cast SDK if not already loaded
      if (!window.chrome?.cast) {
        const script = document.createElement('script');
        script.src =
          'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1';
        script.onload = () => {
          this.waitForCastAPI()
            .then(() => {
              this.setupCastAPI();
              this.isInitialized = true;
              resolve();
            })
            .catch(reject);
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
        this.waitForCastAPI()
          .then(() => {
            this.setupCastAPI();
            this.isInitialized = true;
            resolve();
          })
          .catch(reject);
      }
    });

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
    try {
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
          this.isInitialized = true;
          this.reconnectAttempts = 0; // Reset reconnect attempts on successful init
          console.log('✅ Google Cast initialized successfully');
          console.log('Cast API version:', window.chrome.cast.VERSION);

          // Log current receiver availability
          console.log('Checking for available receivers...');
        },
        (error: any) => {
          console.error('❌ Google Cast initialization failed:', error);
          this.notifyError({
            code: 'INITIALIZATION_FAILED',
            description: 'Failed to initialize Google Cast',
            details: error,
          });

          // Attempt to reinitialize after delay if not too many attempts
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        },
      );
    } catch (error) {
      console.error('Error setting up Cast API:', error);
      this.notifyError({
        code: 'SETUP_FAILED',
        description: 'Failed to set up Google Cast API',
        details: error,
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
      this.initializeCastSDK().catch((error) => {
        console.error('Reconnect attempt failed:', error);
      });
    }, delay);
  }

  private onSessionListener(session: any): void {
    console.log('🎯 Cast session established:', session);

    try {
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
      this.actualCastSession = session; // Store the actual Cast SDK session object
      this.reconnectAttempts = 0; // Reset reconnect attempts on successful session
      this.notifySessionStateChange(castSession);

      // Set up session event listeners
      if (session.addUpdateListener) {
        session.addUpdateListener(this.onSessionUpdateListener.bind(this));
      }

      if (session.addMessageListener) {
        session.addMessageListener(
          'urn:x-cast:com.google.cast.media',
          this.onMediaMessage.bind(this),
        );
      }

      // Handle existing media sessions
      if (session.media && session.media.length > 0) {
        this.handleExistingMediaSession(session.media[0]);
      }
    } catch (error) {
      console.error('❌ Error handling session listener:', error);
      this.notifyError({
        code: 'SESSION_HANDLER_ERROR',
        description: 'Error processing cast session',
        details: error,
      });
    }
  }

  private handleExistingMediaSession(media: any): void {
    try {
      if (this.currentSession) {
        this.currentSession.mediaSessionId = media.sessionId;
        this.currentSession.lastSyncAt = new Date();
        this.notifySessionStateChange(this.currentSession);
      }

      // Set up media event listeners
      media.addUpdateListener(this.onMediaUpdateListener.bind(this));
    } catch (error) {
      console.error('Error handling existing media session:', error);
    }
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

  private onMediaMessage(namespace: string, message: string): void {
    console.log('Media message received:', namespace, message);
  }

  private onReceiverListener(availability: string): void {
    console.log('Cast receiver availability changed:', availability);

    try {
      // Check for both possible availability values
      const isAvailable =
        availability === 'available' ||
        availability === window.chrome?.cast?.ReceiverAvailability?.AVAILABLE;

      if (isAvailable) {
        console.log('Chromecast devices are available on the network');

        // When receivers are available, create a generic device entry
        // The actual device selection happens through the Cast SDK dialog
        const device: CastDevice = {
          id: 'chromecast-available',
          name: 'Cast to TV',
          type: 'chromecast',
          capabilities: ['video_out', 'audio_out'],
          isAvailable: true,
          lastSeen: new Date(),
        };

        // Update devices list
        this.devices = [device];
        this.notifyDeviceAvailable(device);
      } else {
        console.log('No Chromecast devices available');
        this.devices = [];
      }
    } catch (error) {
      console.error('Error handling receiver availability:', error);
      this.notifyError({
        code: 'RECEIVER_HANDLER_ERROR',
        description: 'Error processing receiver availability',
        details: error,
      });
    }
  }

  // Public API methods
  async discoverDevices(): Promise<CastDevice[]> {
    try {
      if (!this.isInitialized) {
        await this.initializeCastSDK();
      }
      return [...this.devices]; // Return a copy to prevent external modification
    } catch (error) {
      console.error('Failed to discover devices:', error);
      this.notifyError({
        code: 'DISCOVERY_FAILED',
        description: 'Failed to discover casting devices',
        details: error,
      });
      return [];
    }
  }

  getAvailableDevices(): CastDevice[] {
    return [...this.devices]; // Return a copy to prevent external modification
  }

  async connectToDevice(deviceId: string): Promise<CastSession> {
    return new Promise((resolve, reject) => {
      try {
        if (!this.isInitialized) {
          reject(new Error('Cast SDK not initialized'));
          return;
        }

        // Check if device exists
        const device = this.devices.find((d) => d.id === deviceId);
        if (!device) {
          reject(new Error(`Device ${deviceId} not found`));
          return;
        }

        // Check if already connected to this device
        if (this.currentSession && this.currentSession.deviceId === deviceId) {
          resolve(this.currentSession);
          return;
        }

        console.log('🔗 Requesting cast session for device:', deviceId);

        window.chrome.cast.requestSession(
          (session: any) => {
            try {
              console.log('✅ Session created successfully:', session);
              console.log('Session details:', {
                sessionId: session.sessionId,
                receiver: session.receiver,
                hasLoadMedia: typeof session.loadMedia === 'function',
                hasStop: typeof session.stop === 'function',
              });

              const castSession: CastSession = {
                id: session.sessionId || session.getSessionId?.(),
                deviceId: deviceId,
                deviceName: session.receiver?.friendlyName || 'Unknown Device',
                deviceType: 'chromecast',
                state: 'connected',
                startedAt: new Date(),
                lastSyncAt: new Date(),
              };

              console.log('📱 Storing cast session:', castSession);

              this.currentSession = castSession;
              this.actualCastSession = session; // Store the actual Cast SDK session object
              this.reconnectAttempts = 0; // Reset on successful connection

              console.log('✅ Session stored successfully:', {
                hasCurrentSession: !!this.currentSession,
                hasActualSession: !!this.actualCastSession,
                actualSessionType: typeof this.actualCastSession,
              });

              // Set up session listeners
              if (session.addUpdateListener) {
                session.addUpdateListener(
                  this.onSessionUpdateListener.bind(this),
                );
              }

              // Notify session state change
              this.notifySessionStateChange(castSession);

              resolve(castSession);
            } catch (error) {
              console.error('❌ Error processing session:', error);
              reject(new Error(`Failed to process session: ${error}`));
            }
          },
          (error: any) => {
            const errorMessage =
              error?.description || error?.message || 'Unknown error';
            console.error('Failed to connect to device:', error);

            this.notifyError({
              code: error?.code || 'CONNECTION_FAILED',
              description: `Failed to connect to device: ${errorMessage}`,
              details: error,
            });

            reject(new Error(`Failed to connect to device: ${errorMessage}`));
          },
        );
      } catch (error) {
        reject(new Error(`Connection error: ${error}`));
      }
    });
  }

  async disconnectFromDevice(deviceId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        if (!this.currentSession || this.currentSession.deviceId !== deviceId) {
          resolve(); // Already disconnected or different device
          return;
        }

        const session = this.actualCastSession;
        if (session) {
          session.stop(
            () => {
              console.log('Session stopped successfully');
              if (this.currentSession) {
                this.currentSession.state = 'disconnected';
                this.currentSession.lastSyncAt = new Date();
                this.notifySessionStateChange(this.currentSession);
                this.currentSession = null;
                this.actualCastSession = null; // Clear the stored session
              }
              resolve();
            },
            (error: any) => {
              console.error('Failed to stop session:', error);
              // Even if stop fails, clean up our state
              if (this.currentSession) {
                this.currentSession.state = 'error';
                this.currentSession.lastSyncAt = new Date();
                this.notifySessionStateChange(this.currentSession);
                this.currentSession = null;
                this.actualCastSession = null; // Clear the stored session
              }

              this.notifyError({
                code: 'DISCONNECT_FAILED',
                description: 'Failed to properly disconnect from device',
                details: error,
              });

              resolve(); // Resolve anyway since we cleaned up our state
            },
          );
        } else {
          // Session not found, clean up our state
          this.currentSession = null;
          this.actualCastSession = null;
          resolve();
        }
      } catch (error) {
        console.error('Error during disconnect:', error);
        // Clean up state even on error
        this.currentSession = null;
        this.actualCastSession = null;
        reject(new Error(`Disconnect error: ${error}`));
      }
    });
  }

  async castMedia(mediaInfo: MediaInfo): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        if (!this.currentSession) {
          reject(new Error('No active cast session'));
          return;
        }

        if (this.currentSession.state !== 'connected') {
          reject(
            new Error(`Cast session not ready: ${this.currentSession.state}`),
          );
          return;
        }

        // Use the stored actual Cast SDK session object
        const session = this.actualCastSession;

        if (!session) {
          console.error('No actual cast session stored');
          reject(new Error('Cast session not found'));
          return;
        }

        console.log('🎬 Using stored cast session for media loading:', session);

        // In development mode, always load our custom receiver
        if (DEVELOPMENT_MODE) {
          console.log('🎵 Loading custom Vibez receiver for all content');
          this.loadCustomReceiver(mediaInfo, session, resolve, reject);
          return;
        }

        // For production, handle different content types
        if (this.isYouTubeUrl(mediaInfo.contentId)) {
          console.log('🎵 YouTube content detected - loading custom receiver');
          this.loadCustomReceiver(mediaInfo, session, resolve, reject);
          return;
        }

        // For non-YouTube content, use standard casting
        this.loadStandardMedia(mediaInfo, session, resolve, reject);
      } catch (error) {
        console.error('Error casting media:', error);
        reject(new Error(`Cast media error: ${error}`));
      }
    });
  }

  private loadCustomReceiver(
    mediaInfo: MediaInfo,
    session: any,
    resolve: () => void,
    reject: (error: Error) => void,
  ): void {
    try {
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
    } catch (error) {
      console.error('Error loading custom receiver:', error);
      reject(new Error(`Custom receiver error: ${error}`));
    }
  }

  private sendMediaToReceiver(mediaInfo: MediaInfo, session: any): void {
    try {
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
    } catch (error) {
      console.error('Error sending media to receiver:', error);
    }
  }

  private loadStandardMedia(
    mediaInfo: MediaInfo,
    session: any,
    resolve: () => void,
    reject: (error: Error) => void,
  ): void {
    try {
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
    } catch (error) {
      console.error('Error loading standard media:', error);
      reject(new Error(`Standard media error: ${error}`));
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
    try {
      if (!this.currentSession || this.currentSession.state !== 'connected') {
        return; // Silently return if no active session
      }

      const session = this.actualCastSession;
      if (!session) {
        return; // No session to send messages to
      }

      // Send queue data to custom receiver
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

      session.sendMessage(
        'urn:x-cast:vibez.queue',
        queueMessage,
        () => console.log('Queue updated on receiver'),
        (error: any) => console.error('Failed to update queue:', error),
      );
    } catch (error) {
      console.error('Failed to update queue:', error);
      this.notifyError({
        code: 'QUEUE_UPDATE_FAILED',
        description: 'Failed to update queue on cast device',
        details: error,
      });
    }
  }

  async updateRoomInfo(roomInfo: {
    name: string;
    participantCount: number;
  }): Promise<void> {
    try {
      if (!this.currentSession || this.currentSession.state !== 'connected') {
        return; // Silently return if no active session
      }

      const session = this.actualCastSession;
      if (!session) {
        return; // No session to send messages to
      }

      console.log('Sending room info to receiver:', roomInfo);

      session.sendMessage(
        'urn:x-cast:vibez.room',
        roomInfo,
        () => console.log('Room info updated on receiver'),
        (error: any) => console.error('Failed to update room info:', error),
      );
    } catch (error) {
      console.error('Failed to update room info:', error);
      this.notifyError({
        code: 'ROOM_UPDATE_FAILED',
        description: 'Failed to update room info on cast device',
        details: error,
      });
    }
  }

  async syncPlaybackState(state: any): Promise<void> {
    try {
      if (!this.currentSession || this.currentSession.state !== 'connected') {
        return; // Silently return if no active session
      }

      const session = this.actualCastSession;
      if (!session || !session.media || session.media.length === 0) {
        return; // No media session to sync
      }

      const media = session.media[0];

      // Sync play/pause state
      if (
        state.isPlaying &&
        media.playerState === window.chrome.cast.media.PlayerState.PAUSED
      ) {
        await new Promise<void>((resolve, reject) => {
          media.play(
            null,
            () => {
              console.log('Playback resumed on cast device');
              resolve();
            },
            (error: any) => {
              console.error('Failed to resume playback:', error);
              reject(error);
            },
          );
        });
      } else if (
        !state.isPlaying &&
        media.playerState === window.chrome.cast.media.PlayerState.PLAYING
      ) {
        await new Promise<void>((resolve, reject) => {
          media.pause(
            null,
            () => {
              console.log('Playback paused on cast device');
              resolve();
            },
            (error: any) => {
              console.error('Failed to pause playback:', error);
              reject(error);
            },
          );
        });
      }

      // Sync position if there's a significant difference (more than 2 seconds)
      if (state.positionMs && typeof state.positionMs === 'number') {
        const currentTimeMs = (media.currentTime || 0) * 1000;
        const timeDifference = Math.abs(currentTimeMs - state.positionMs);

        if (timeDifference > 2000) {
          // 2 second threshold
          const seekRequest = new window.chrome.cast.media.SeekRequest();
          seekRequest.currentTime = state.positionMs / 1000;

          await new Promise<void>((resolve, reject) => {
            media.seek(
              seekRequest,
              () => {
                console.log(`Seek completed to ${seekRequest.currentTime}s`);
                resolve();
              },
              (error: any) => {
                console.error('Failed to seek:', error);
                reject(error);
              },
            );
          });
        }
      }

      // Update last sync time
      if (this.currentSession) {
        this.currentSession.lastSyncAt = new Date();
        this.notifySessionStateChange(this.currentSession);
      }
    } catch (error) {
      console.error('Failed to sync playback state:', error);
      this.notifyError({
        code: 'SYNC_FAILED',
        description: 'Failed to synchronize playback state with cast device',
        details: error,
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
    try {
      // Clear all callbacks
      this.deviceAvailableCallbacks = [];
      this.sessionStateCallbacks = [];
      this.errorCallbacks = [];

      // Disconnect from current session
      if (this.currentSession) {
        this.disconnectFromDevice(this.currentSession.deviceId).catch(
          (error) => {
            console.error('Error disconnecting during destroy:', error);
          },
        );
      }

      // Reset state
      this.devices = [];
      this.currentSession = null;
      this.actualCastSession = null; // Clear the stored session
      this.isInitialized = false;
      this.initializationPromise = null;
      this.reconnectAttempts = 0;

      console.log('Cast manager destroyed');
    } catch (error) {
      console.error('Error during cast manager destruction:', error);
    }
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
    console.log('Debug info:', this.getDebugInfo());

    if (!this.isInitialized) {
      console.log('Cast not initialized, initializing...');
      await this.initializeCastSDK();
    }

    // The Cast SDK doesn't provide a direct way to force discovery
    // Device availability is handled automatically by the SDK
    console.log('Current devices:', this.devices);
  }
}

// Export singleton instance
export const castManager = new GoogleCastManager();
export default castManager;
