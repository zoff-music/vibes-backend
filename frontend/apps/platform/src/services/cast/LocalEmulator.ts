import type { CastDevice, CastSession } from '@vibez/models';
import { safeWrap, useRoomStore } from '@vibez/shared';

import { CUSTOM_RECEIVER_URL, LOCAL_EMULATOR_DEVICE_ID } from './constants';
import type { LocalCastMessage } from './types';

/**
 * Handles local cast receiver window for development/testing.
 * Opens a popup window that displays the cast receiver UI locally.
 */
export class LocalEmulator {
  private localReceiverWindow: Window | null = null;
  private localReceiverOrigin: string | null = null;
  private localReceiverFrame: HTMLIFrameElement | null = null;
  private localMessageQueue: LocalCastMessage[] = [];
  private localReceiverReady = false;
  private localReadyTimeout: ReturnType<typeof setTimeout> | null = null;

  private notifyDeviceAvailable: (device: CastDevice) => void;
  private getDevices: () => CastDevice[];
  private setDevices: (devices: CastDevice[]) => void;

  constructor(deps: {
    notifyDeviceAvailable: (device: CastDevice) => void;
    getDevices: () => CastDevice[];
    setDevices: (devices: CastDevice[]) => void;
  }) {
    this.notifyDeviceAvailable = deps.notifyDeviceAvailable;
    this.getDevices = deps.getDevices;
    this.setDevices = deps.setDevices;

    if (typeof window !== 'undefined') {
      window.addEventListener(
        'message',
        this.handleLocalReceiverMessage.bind(this),
      );
    }
  }

  handleLocalReceiverMessage(event: MessageEvent): void {
    const data = event.data as LocalCastMessage | null;
    if (!data || typeof data !== 'object' || !('action' in data)) return;
    if (data.action !== 'receiverReady') return;

    if (event.origin !== window.location.origin) {
      const [originErr, originUrl] = safeWrap(() => new URL(event.origin));
      const receiverOrigin = this.localReceiverOrigin || window.location.origin;
      const [receiverErr, receiverUrl] = safeWrap(
        () => new URL(receiverOrigin),
      );

      if (originErr || receiverErr || !originUrl || !receiverUrl) return;

      const isLocalOrigin =
        (originUrl.hostname === 'localhost' ||
          originUrl.hostname === '127.0.0.1') &&
        (receiverUrl.hostname === 'localhost' ||
          receiverUrl.hostname === '127.0.0.1');
      if (!isLocalOrigin) return;
    }

    this.localReceiverReady = true;
    if (this.localReadyTimeout) {
      clearTimeout(this.localReadyTimeout);
      this.localReadyTimeout = null;
    }
    this.flushLocalMessageQueue();
  }

  ensureLocalEmulatorDevice(): void {
    const devices = this.getDevices();
    const exists = devices.some(
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

    this.setDevices([device, ...devices]);
    this.notifyDeviceAvailable(device);
  }

  getRoomIdFromContext(): string {
    const roomState = useRoomStore.getState();
    const roomIdFromState = roomState.room?.id || '';
    const roomIdFromSearch =
      new URLSearchParams(window.location.search).get('roomId') || '';
    const roomIdFromPath = (() => {
      const match = window.location.pathname.match(/\/rooms\/([^/]+)/);
      return match?.[1] || '';
    })();
    const roomIdFromHash = (() => {
      const match = window.location.hash.match(/\/rooms\/([^/]+)/);
      return match?.[1] || '';
    })();

    return (
      roomIdFromState || roomIdFromSearch || roomIdFromPath || roomIdFromHash
    );
  }

  getCasterIdFromContext(): string {
    const roomState = useRoomStore.getState();
    if (roomState.userId) return roomState.userId;

    const params = new URLSearchParams(window.location.search);
    return params.get('casterId') || params.get('sessionId') || '';
  }

  getLocalReceiverUrl(): string {
    const receiverPath = (() => {
      if (
        CUSTOM_RECEIVER_URL.startsWith('http://') ||
        CUSTOM_RECEIVER_URL.startsWith('https://')
      ) {
        const [parseErr, parsedUrl] = safeWrap(
          () => new URL(CUSTOM_RECEIVER_URL),
        );
        if (parseErr || !parsedUrl) {
          console.error(
            'Invalid custom receiver URL; falling back to path:',
            CUSTOM_RECEIVER_URL,
          );
          return '/casting/receiver/';
        }

        const pathname = parsedUrl.pathname || '/';
        return `${pathname}${parsedUrl.search || ''}`;
      }

      if (!CUSTOM_RECEIVER_URL.startsWith('/')) {
        return `/${CUSTOM_RECEIVER_URL}`;
      }

      return CUSTOM_RECEIVER_URL;
    })();

    return this.getReceiverUrlWithParams(window.location.origin, receiverPath);
  }

  getReceiverUrlWithParams(
    originOverride?: string,
    pathOverride?: string,
  ): string {
    const baseOrigin = originOverride || window.location.origin;
    const basePath = pathOverride || CUSTOM_RECEIVER_URL;
    const baseUrl =
      basePath.startsWith('http://') || basePath.startsWith('https://')
        ? basePath
        : `${baseOrigin}${basePath.startsWith('/') ? basePath : `/${basePath}`}`;

    const params = new URLSearchParams();
    params.set('castReceiver', '1');

    const roomId = this.getRoomIdFromContext();
    if (roomId) {
      params.set('roomId', roomId);
    }

    const casterId = this.getCasterIdFromContext();
    if (casterId) {
      params.set('casterId', casterId);
      params.set('sessionId', casterId);
    }

    // Only pass debug flag to receiver when VITE_DEBUG is enabled in sender app
    if (import.meta.env.VITE_DEBUG === 'true') {
      params.set('debug', 'true');
    }

    const separator = baseUrl.includes('?') ? '&' : '?';
    return `${baseUrl}${separator}${params.toString()}`;
  }

  openLocalReceiver(): Window | null {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return null;
    }

    this.localReceiverReady = false;
    if (this.localReadyTimeout) {
      clearTimeout(this.localReadyTimeout);
      this.localReadyTimeout = null;
    }

    const receiverUrl = this.getLocalReceiverUrl();
    const [urlErr, parsedUrl] = safeWrap(() => new URL(receiverUrl));
    if (urlErr || !parsedUrl) {
      console.error('Invalid local receiver URL:', receiverUrl);
      return null;
    }

    this.localReceiverOrigin = parsedUrl.origin;

    const existingWindow =
      this.localReceiverWindow && !this.localReceiverWindow.closed
        ? this.localReceiverWindow
        : null;
    const popup =
      existingWindow ||
      (() => {
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

    if (existingWindow && popup) {
      const [navErr] = safeWrap(() => {
        popup.location.href = receiverUrl;
      });
      if (navErr) {
        console.error('Failed to update local receiver URL:', navErr);
      }
    }

    if (!popup) {
      console.error('Local cast receiver window was blocked or failed to open');
      return null;
    }

    this.localReceiverFrame = null;
    this.localReceiverWindow = popup;
    this.localReadyTimeout = setTimeout(() => {
      if (!this.localReceiverReady) {
        this.localReceiverReady = true;
        this.flushLocalMessageQueue();
      }
      this.localReadyTimeout = null;
    }, 1500);
    this.flushLocalMessageQueue();
    return popup;
  }

  sendLocalMessage(message: LocalCastMessage): void {
    if (!this.localReceiverWindow || this.localReceiverWindow.closed) {
      console.warn('[Local Cast] receiver window not available');
      return;
    }

    if (!this.localReceiverReady) {
      this.localMessageQueue.push(message);
      return;
    }

    const targetOrigin = this.localReceiverOrigin || '*';
    this.localReceiverWindow.postMessage(message, targetOrigin);
  }

  flushLocalMessageQueue(): void {
    if (!this.localReceiverReady) return;
    if (!this.localReceiverWindow || this.localReceiverWindow.closed) return;

    const targetOrigin = this.localReceiverOrigin || '*';
    while (this.localMessageQueue.length > 0) {
      const message = this.localMessageQueue.shift();
      if (message) {
        this.localReceiverWindow.postMessage(message, targetOrigin);
      }
    }
  }

  prepareLocalReceiverWindow(): boolean {
    const receiverWindow = this.openLocalReceiver();
    return receiverWindow !== null;
  }

  createLocalSession(): CastSession {
    return {
      id: `local-${Date.now()}`,
      deviceId: LOCAL_EMULATOR_DEVICE_ID,
      deviceName: 'Local Cast (Emulator)',
      deviceType: 'chromecast',
      state: 'connected',
      startedAt: new Date(),
      lastSyncAt: new Date(),
      mediaSessionId: 'local-session',
    };
  }

  disconnectLocal(): void {
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
  }

  isLocalEmulator(deviceId: string): boolean {
    return deviceId === LOCAL_EMULATOR_DEVICE_ID;
  }

  destroy(): void {
    this.disconnectLocal();
    if (typeof window !== 'undefined') {
      window.removeEventListener('message', this.handleLocalReceiverMessage);
    }
  }
}
