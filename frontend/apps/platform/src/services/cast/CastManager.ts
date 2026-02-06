import { createCastingToken } from '@vibez/api';
import type {
  CastDevice,
  CastError,
  CastSession,
  CastSessionState,
  CastManager as ICastManager,
  MediaInfo,
} from '@vibez/models';
import { safeWrap, safeWrapAsync, useRoomStore } from '@vibez/shared';

import {
  CAST_APPLICATION_ID,
  CUSTOM_RECEIVER_URL,
  DEVELOPMENT_MODE,
  LOCAL_EMULATOR_ENABLED,
} from './constants';
import { CastEventBus } from './EventBus';
import { LocalEmulator } from './LocalEmulator';
import type { LocalCastMessage } from './types';

const CUSTOM_NAMESPACE = 'urn:x-cast:com.vibez.cast';
const MEDIA_NAMESPACE = 'urn:x-cast:com.google.cast.media';

/**
 * Google Cast SDK Manager - orchestrates all casting functionality.
 * Implements the CastManager interface from @vibez/models.
 */
class GoogleCastManager implements ICastManager {
  private devices: CastDevice[] = [];
  private currentSession: CastSession | null = null;
  private actualCastSession: chrome.cast.Session | null = null;

  private isInitialized = false;
  private initializationPromise: Promise<void> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 3;
  private reconnectDelay = 1000;

  private eventBus: CastEventBus;
  private localEmulator: LocalEmulator;

  constructor() {
    this.eventBus = new CastEventBus();
    this.localEmulator = new LocalEmulator({
      notifyDeviceAvailable: this.notifyDeviceAvailable.bind(this),
      getDevices: () => this.devices,
      setDevices: (devices) => {
        this.devices = devices;
      },
    });

    if (typeof window !== 'undefined') {
      this.initializeCastSDK();
    }
  }

  // ============================================================
  // SDK Initialization
  // ============================================================

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
      const maxAttempts = 100;

      console.log('[Cast] waiting for Cast API availability...');
      const checkInterval = setInterval(() => {
        attempts++;

        if (window.chrome?.cast?.isAvailable) {
          clearInterval(checkInterval);
          console.log('[Cast] Cast API available', { attempts });
          resolve();
        } else if (attempts >= maxAttempts) {
          clearInterval(checkInterval);
          const error = {
            message: 'Google Cast API not available after timeout',
            code: 'API_TIMEOUT_INTERNAL',
          };
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
          console.log('Google Cast API `initialize` call successful.');
        },
        (error: chrome.cast.Error) => {
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
    const delay = this.reconnectDelay * 2 ** (this.reconnectAttempts - 1);

    console.log(
      `Scheduling Cast SDK reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`,
    );

    setTimeout(() => {
      this.initializationPromise = null;
      const [err] = safeWrap(() => {
        this.initializeCastSDK();
      });
      if (err) console.error('Reconnect attempt failed:', err);
    }, delay);
  }

  // ============================================================
  // Session Listeners
  // ============================================================

  private onSessionListener(session: chrome.cast.Session): void {
    console.log('🎯 Cast session established:', session);
    console.log('[Cast] session details', {
      sessionId: session?.sessionId,
      receiverName: session?.receiver?.friendlyName,
      mediaCount: session?.media?.length || 0,
    });

    const [_, err] = safeWrap(() => {
      const castSession: CastSession = {
        id: session.sessionId,
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
          MEDIA_NAMESPACE,
          this.onMediaMessage.bind(this),
        );
        session.addMessageListener(
          CUSTOM_NAMESPACE,
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

  private handleExistingMediaSession(media: chrome.cast.media.Media): void {
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
      this.actualCastSession = null;
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
          this.localEmulator.ensureLocalEmulatorDevice();
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
        this.localEmulator.ensureLocalEmulatorDevice();
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

  private onMediaMessage(namespace: string, message: string): void {
    console.log('Media message received:', namespace, message);
  }

  private onCustomMessage(_namespace: string, message: string): void {
    const [err] = safeWrap(() => {
      const data = JSON.parse(message);
      if (data.action === 'receiverReady') {
        const roomId = this.localEmulator.getRoomIdFromContext();
        if (!roomId) {
          console.warn('[Cast] receiverReady but roomId missing');
          return;
        }

        const [joinErr] = safeWrap(() => {
          void this.joinRoom(roomId);
        });
        if (joinErr) {
          console.error(
            'Failed to send joinRoom after receiverReady:',
            joinErr,
          );
        }
        return;
      }

      if (data.action === 'LOG') {
        const { level, args } = data;
        const prefix = '%c[RECEIVER]';
        const style =
          'background: #222; color: #bada55; font-weight: bold; padding: 2px 4px; border-radius: 2px;';

        const logArgs = args.map((arg: string) => {
          try {
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

  // ============================================================
  // Public API - Device Discovery
  // ============================================================

  async discoverDevices(): Promise<CastDevice[]> {
    if (LOCAL_EMULATOR_ENABLED) {
      this.localEmulator.ensureLocalEmulatorDevice();
    }
    if (!this.isInitialized) {
      const [initErr] = await safeWrapAsync(this.initializeCastSDK());
      if (initErr) {
        if (
          typeof initErr === 'object' &&
          initErr !== null &&
          'code' in initErr &&
          (initErr as { code: string }).code === 'API_TIMEOUT_INTERNAL'
        ) {
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
      this.localEmulator.ensureLocalEmulatorDevice();
    }
    return [...this.devices];
  }

  prepareLocalReceiverWindow(): boolean {
    if (!LOCAL_EMULATOR_ENABLED) {
      return false;
    }

    const success = this.localEmulator.prepareLocalReceiverWindow();
    if (!success) {
      this.notifyError({
        code: 'LOCAL_RECEIVER_BLOCKED',
        description: 'Failed to open local cast receiver window',
        details: new Error('Local cast receiver window could not be opened'),
      });
    }
    return success;
  }

  // ============================================================
  // Public API - Session Management
  // ============================================================

  async connectToDevice(deviceId: string): Promise<CastSession> {
    if (
      LOCAL_EMULATOR_ENABLED &&
      this.localEmulator.isLocalEmulator(deviceId)
    ) {
      const receiverWindow = this.localEmulator.openLocalReceiver();
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

      const castSession = this.localEmulator.createLocalSession();
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
        (session: chrome.cast.Session) => {
          const [err, res] = safeWrap(() => {
            console.log('✅ Session created successfully:', session);
            const castSession: CastSession = {
              id: session.sessionId,
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
        (error: chrome.cast.Error) => {
          const errMsg = error?.description || 'Unknown error';
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

    if (this.localEmulator.isLocalEmulator(deviceId)) {
      this.localEmulator.disconnectLocal();
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
        (error: chrome.cast.Error) => {
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

  // ============================================================
  // Public API - Media Control
  // ============================================================

  async castMedia(mediaInfo: MediaInfo): Promise<void> {
    if (!this.currentSession) throw new Error('No active cast session');
    if (this.currentSession.state !== 'connected') {
      throw new Error(`Cast session not ready: ${this.currentSession.state}`);
    }

    if (this.localEmulator.isLocalEmulator(this.currentSession.deviceId)) {
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
      const roomState = useRoomStore.getState();
      const roomId = roomState.room?.id || '';

      if (DEVELOPMENT_MODE || this.isYouTubeUrl(mediaInfo.contentId)) {
        console.log('🎵 Preparing custom Zoff receiver session');

        const url = new URL(CUSTOM_RECEIVER_URL, window.location.origin);
        if (roomId) url.searchParams.set('roomId', roomId);
        if (
          new URLSearchParams(window.location.search).get('debug') === 'true'
        ) {
          url.searchParams.set('debug', 'true');
        }

        const receiverUrl = url.toString();

        console.log('[Cast] Custom context ready', { receiverUrl });

        setTimeout(() => {
          this.sendMediaToReceiver(mediaInfo, session);
        }, 1000);

        if (this.currentSession) {
          this.currentSession.state = 'connected';
          this.currentSession.lastSyncAt = new Date();
          this.notifySessionStateChange(this.currentSession);
        }

        resolve();
      } else {
        this.loadStandardMedia(mediaInfo, session, resolve, reject);
      }
    });
  }

  private sendMediaToReceiver(mediaInfo: MediaInfo, session: any): void {
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
        queue: [],
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
        CUSTOM_NAMESPACE,
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
      this.localEmulator.sendLocalMessage(message);
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

      const castMediaInfo = new window.chrome.cast.media.MediaInfo(
        mediaInfo.contentId,
        mediaInfo.contentType,
      );

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

  async updateQueue(queue: any[]): Promise<void> {
    if (!this.currentSession || this.currentSession.state !== 'connected') {
      return;
    }

    const queueMessage = {
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

    if (this.localEmulator.isLocalEmulator(this.currentSession.deviceId)) {
      this.localEmulator.sendLocalMessage(queueMessage as LocalCastMessage);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const [err] = safeWrap(() => {
      console.log('[Cast] updateQueue send', {
        count: queueMessage.queue.length,
        timestamp: queueMessage.timestamp,
      });
      session.sendMessage(
        CUSTOM_NAMESPACE,
        queueMessage,
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

    const message = {
      action: 'updateRoomInfo',
      roomInfo: roomInfo,
      timestamp: Date.now(),
    };

    if (this.localEmulator.isLocalEmulator(this.currentSession.deviceId)) {
      this.localEmulator.sendLocalMessage(message as LocalCastMessage);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const [err] = safeWrap(() => {
      console.log('[Cast] updateRoomInfo send', message);
      session.sendMessage(
        CUSTOM_NAMESPACE,
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

    const message = {
      action: 'syncPlayback',
      isPlaying: state.isPlaying,
      positionMs: state.positionMs,
      currentSong: state.currentSong,
      timestamp: Date.now(),
    };

    if (this.localEmulator.isLocalEmulator(this.currentSession.deviceId)) {
      this.localEmulator.sendLocalMessage(message as LocalCastMessage);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    const [err] = safeWrap(() => {
      console.log('[Cast] syncPlayback send', {
        isPlaying: message.isPlaying,
        positionMs: message.positionMs,
        title: message.currentSong?.title,
        timestamp: message.timestamp,
      });
      session.sendMessage(
        CUSTOM_NAMESPACE,
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

  async joinRoom(roomId: string): Promise<void> {
    if (!this.currentSession) return;

    const [tokenErr, tokenResp] = await createCastingToken(roomId);
    if (tokenErr || !tokenResp?.token) {
      console.error('[Cast] failed to mint cast token', tokenErr);
      throw tokenErr || new Error('Failed to mint cast token');
    }

    const casterId = this.localEmulator.getCasterIdFromContext() || undefined;
    const message = {
      action: 'joinRoom',
      roomId,
      castToken: tokenResp.token,
      casterId,
      sessionId: casterId,
      timestamp: Date.now(),
    };

    if (this.localEmulator.isLocalEmulator(this.currentSession.deviceId)) {
      this.localEmulator.sendLocalMessage(message as LocalCastMessage);
      return;
    }

    const session = this.actualCastSession;
    if (!session) return;

    return new Promise((resolve, reject) => {
      console.log('[Cast] sending joinRoom message', message);

      const [err] = safeWrap(() => {
        session.sendMessage(
          CUSTOM_NAMESPACE,
          message,
          () => {
            console.log('✅ joinRoom message sent');
            resolve();
          },
          (error: chrome.cast.Error) => {
            console.error('Failed to send joinRoom message:', error);
            reject(error);
          },
        );
      });

      if (err) reject(err);
    });
  }

  // ============================================================
  // Event Handling
  // ============================================================

  onDeviceAvailable(callback: (device: CastDevice) => void): void {
    this.eventBus.onDeviceAvailable(callback);
  }

  onSessionStateChange(callback: (session: CastSession) => void): void {
    this.eventBus.onSessionStateChange(callback);
  }

  onCastError(callback: (error: CastError) => void): void {
    this.eventBus.onCastError(callback);
  }

  private notifyDeviceAvailable(device: CastDevice): void {
    this.eventBus.notifyDeviceAvailable(device);
  }

  private notifySessionStateChange(session: CastSession): void {
    this.eventBus.notifySessionStateChange(session);
  }

  private notifyError(error: CastError): void {
    this.eventBus.notifyError(error);
  }

  // ============================================================
  // Utility Methods
  // ============================================================

  getCurrentSession(): CastSession | null {
    return this.currentSession;
  }

  isConnected(): boolean {
    return this.currentSession?.state === 'connected';
  }

  getConnectionState(): CastSessionState | null {
    return this.currentSession?.state || null;
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

  getStatus(): {
    isInitialized: boolean;
    deviceCount: number;
    currentSession: CastSession | null;
    reconnectAttempts: number;
  } {
    return {
      isInitialized: this.isInitialized,
      deviceCount: this.devices.length,
      currentSession: this.currentSession,
      reconnectAttempts: this.reconnectAttempts,
    };
  }

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
      receiverAvailability: undefined,
      devices: this.devices,
      currentSession: this.currentSession,
      isInitialized: this.isInitialized,
    };
  }

  async forceDiscovery(): Promise<void> {
    console.log('🔍 Forcing device discovery...');
    const [err] = await safeWrapAsync(this.initializeCastSDK());
    if (err) {
      console.error('Failed to initialize SDK during force discovery:', err);
      return;
    }

    if (LOCAL_EMULATOR_ENABLED) {
      this.localEmulator.ensureLocalEmulatorDevice();
    }

    console.log('Current devices:', this.devices);
  }

  destroy(): void {
    this.localEmulator.destroy();
    this.eventBus.removeAllListeners();

    if (this.currentSession && this.actualCastSession) {
      const [err] = safeWrap(() => {
        this.actualCastSession?.stop(
          () => console.log('Session stopped during destroy'),
          (err: chrome.cast.Error) =>
            console.error('Error stopping session:', err),
        );
      });
      if (err) console.error('Error during session cleanup:', err);
    }

    this.currentSession = null;
    this.actualCastSession = null;
    this.devices = [];
    this.isInitialized = false;
    this.initializationPromise = null;
  }
}

// Export singleton instance
export const castManager = new GoogleCastManager();
