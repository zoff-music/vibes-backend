import type { 
  CastManager, 
  CastDevice, 
  CastSession, 
  MediaInfo, 
  CastError,
  CastSessionState 
} from '../types/casting';

// Google Cast Application ID - replace with your actual receiver app ID
const CAST_APPLICATION_ID = 'CC1AD845'; // Default Media Receiver

class GoogleCastManager implements CastManager {
  private devices: CastDevice[] = [];
  private currentSession: CastSession | null = null;
  private isInitialized = false;
  
  private deviceAvailableCallbacks: Array<(device: CastDevice) => void> = [];
  private sessionStateCallbacks: Array<(session: CastSession) => void> = [];
  private errorCallbacks: Array<(error: CastError) => void> = [];

  constructor() {
    this.initializeCastSDK();
  }

  private async initializeCastSDK(): Promise<void> {
    return new Promise((resolve, reject) => {
      // Load Google Cast SDK if not already loaded
      if (!window.chrome?.cast) {
        const script = document.createElement('script');
        script.src = 'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1';
        script.onload = () => {
          this.waitForCastAPI().then(() => {
            this.setupCastAPI();
            resolve();
          }).catch(reject);
        };
        script.onerror = () => reject(new Error('Failed to load Google Cast SDK'));
        document.head.appendChild(script);
      } else {
        this.waitForCastAPI().then(() => {
          this.setupCastAPI();
          resolve();
        }).catch(reject);
      }
    });
  }

  private waitForCastAPI(): Promise<void> {
    return new Promise((resolve, reject) => {
      const checkInterval = setInterval(() => {
        if (window.chrome?.cast?.isAvailable) {
          clearInterval(checkInterval);
          resolve();
        }
      }, 100);

      // Timeout after 10 seconds
      setTimeout(() => {
        clearInterval(checkInterval);
        reject(new Error('Google Cast API not available'));
      }, 10000);
    });
  }

  private setupCastAPI(): void {
    const sessionRequest = new window.chrome.cast.SessionRequest(CAST_APPLICATION_ID);
    const apiConfig = new window.chrome.cast.ApiConfig(
      sessionRequest,
      this.onSessionListener.bind(this),
      this.onReceiverListener.bind(this),
      window.chrome.cast.AutoJoinPolicy.TAB_AND_ORIGIN_SCOPED,
      window.chrome.cast.DefaultActionPolicy.CREATE_SESSION
    );

    window.chrome.cast.initialize(
      apiConfig,
      () => {
        this.isInitialized = true;
        console.log('Google Cast initialized successfully');
      },
      (error: any) => {
        console.error('Google Cast initialization failed:', error);
        this.notifyError({
          code: 'INITIALIZATION_FAILED',
          description: 'Failed to initialize Google Cast',
          details: error
        });
      }
    );
  }

  private onSessionListener(session: any): void {
    console.log('Cast session established:', session);
    
    const castSession: CastSession = {
      id: session.sessionId,
      deviceId: session.receiver.friendlyName,
      deviceName: session.receiver.friendlyName,
      deviceType: 'chromecast',
      state: 'connected',
      startedAt: new Date(),
      mediaSessionId: session.media?.[0]?.sessionId
    };

    this.currentSession = castSession;
    this.notifySessionStateChange(castSession);

    // Set up session event listeners
    session.addUpdateListener(this.onSessionUpdateListener.bind(this));
    session.addMessageListener('urn:x-cast:com.google.cast.media', this.onMediaMessage.bind(this));
  }

  private onSessionUpdateListener(isAlive: boolean): void {
    if (!isAlive && this.currentSession) {
      this.currentSession.state = 'disconnected';
      this.notifySessionStateChange(this.currentSession);
      this.currentSession = null;
    }
  }

  private onMediaMessage(namespace: string, message: string): void {
    console.log('Media message received:', namespace, message);
  }

  private onReceiverListener(availability: string): void {
    console.log('Cast receiver availability:', availability);
    
    if (availability === window.chrome.cast.ReceiverAvailability.AVAILABLE) {
      // Simulate device discovery for demo purposes
      const device: CastDevice = {
        id: 'chromecast-1',
        name: 'Living Room TV',
        type: 'chromecast',
        capabilities: ['video_out', 'audio_out'],
        isAvailable: true,
        lastSeen: new Date()
      };
      
      this.devices = [device];
      this.notifyDeviceAvailable(device);
    } else {
      this.devices = [];
    }
  }

  // Public API methods
  async discoverDevices(): Promise<CastDevice[]> {
    if (!this.isInitialized) {
      await this.initializeCastSDK();
    }
    return this.devices;
  }

  getAvailableDevices(): CastDevice[] {
    return this.devices;
  }

  async connectToDevice(deviceId: string): Promise<CastSession> {
    return new Promise((resolve, reject) => {
      if (!this.isInitialized) {
        reject(new Error('Cast SDK not initialized'));
        return;
      }

      window.chrome.cast.requestSession(
        (session: any) => {
          const castSession: CastSession = {
            id: session.sessionId,
            deviceId: deviceId,
            deviceName: session.receiver.friendlyName,
            deviceType: 'chromecast',
            state: 'connected',
            startedAt: new Date()
          };
          
          this.currentSession = castSession;
          resolve(castSession);
        },
        (error: any) => {
          reject(new Error(`Failed to connect to device: ${error.description}`));
        }
      );
    });
  }

  async disconnectFromDevice(deviceId: string): Promise<void> {
    if (this.currentSession && this.currentSession.deviceId === deviceId) {
      const session = window.chrome.cast.Session.getSessionById(this.currentSession.id);
      if (session) {
        session.stop(
          () => {
            console.log('Session stopped successfully');
            this.currentSession = null;
          },
          (error: any) => {
            console.error('Failed to stop session:', error);
          }
        );
      }
    }
  }

  async castMedia(mediaInfo: MediaInfo): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.currentSession) {
        reject(new Error('No active cast session'));
        return;
      }

      const session = window.chrome.cast.Session.getSessionById(this.currentSession.id);
      if (!session) {
        reject(new Error('Cast session not found'));
        return;
      }

      const castMediaInfo = new window.chrome.cast.media.MediaInfo(
        mediaInfo.contentId,
        mediaInfo.contentType
      );

      const metadata = new window.chrome.cast.media.GenericMediaMetadata();
      metadata.title = mediaInfo.metadata.title;
      metadata.subtitle = mediaInfo.metadata.artist;
      
      if (mediaInfo.metadata.images && mediaInfo.metadata.images.length > 0) {
        metadata.images = mediaInfo.metadata.images.map(img => ({
          url: img.url,
          height: img.height,
          width: img.width
        }));
      }

      castMediaInfo.metadata = metadata;
      castMediaInfo.streamType = window.chrome.cast.media.StreamType.BUFFERED;

      const request = new window.chrome.cast.media.LoadRequest(castMediaInfo);
      
      session.loadMedia(
        request,
        (media: any) => {
          console.log('Media loaded successfully:', media);
          if (this.currentSession) {
            this.currentSession.mediaSessionId = media.sessionId;
            this.currentSession.state = 'connected';
            this.notifySessionStateChange(this.currentSession);
          }
          resolve();
        },
        (error: any) => {
          console.error('Failed to load media:', error);
          reject(new Error(`Failed to cast media: ${error.description}`));
        }
      );
    });
  }

  async updateQueue(queue: any[]): Promise<void> {
    // For now, just cast the current song
    // In a full implementation, this would update the entire queue on the receiver
    console.log('Queue update requested:', queue);
  }

  async syncPlaybackState(state: any): Promise<void> {
    if (!this.currentSession) return;

    const session = window.chrome.cast.Session.getSessionById(this.currentSession.id);
    if (!session || !session.media || session.media.length === 0) return;

    const media = session.media[0];
    
    if (state.isPlaying) {
      media.play(null, 
        () => console.log('Playback resumed'),
        (error: any) => console.error('Failed to resume playback:', error)
      );
    } else {
      media.pause(null,
        () => console.log('Playback paused'),
        (error: any) => console.error('Failed to pause playback:', error)
      );
    }

    // Seek to position if needed
    if (state.positionMs && Math.abs(media.currentTime * 1000 - state.positionMs) > 1000) {
      const seekRequest = new window.chrome.cast.media.SeekRequest();
      seekRequest.currentTime = state.positionMs / 1000;
      
      media.seek(seekRequest,
        () => console.log('Seek completed'),
        (error: any) => console.error('Failed to seek:', error)
      );
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
    this.deviceAvailableCallbacks.forEach(callback => callback(device));
  }

  private notifySessionStateChange(session: CastSession): void {
    this.sessionStateCallbacks.forEach(callback => callback(session));
  }

  private notifyError(error: CastError): void {
    this.errorCallbacks.forEach(callback => callback(error));
  }

  // Utility methods
  getCurrentSession(): CastSession | null {
    return this.currentSession;
  }

  isConnected(): boolean {
    return this.currentSession?.state === 'connected';
  }
}

// Export singleton instance
export const castManager = new GoogleCastManager();
export default castManager;