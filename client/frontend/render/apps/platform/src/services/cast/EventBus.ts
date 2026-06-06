import type { CastDevice, CastError, CastSession } from '@vibez/models';

type DeviceCallback = (device: CastDevice) => void;
type SessionCallback = (session: CastSession) => void;
type ErrorCallback = (error: CastError) => void;

/**
 * Simple event bus for cast-related callbacks.
 * Manages subscriptions for device availability, session state changes, and errors.
 */
export class CastEventBus {
  private deviceAvailableCallbacks: DeviceCallback[] = [];
  private sessionStateCallbacks: SessionCallback[] = [];
  private errorCallbacks: ErrorCallback[] = [];

  onDeviceAvailable(callback: DeviceCallback): void {
    this.deviceAvailableCallbacks.push(callback);
  }

  onSessionStateChange(callback: SessionCallback): void {
    this.sessionStateCallbacks.push(callback);
  }

  onCastError(callback: ErrorCallback): void {
    this.errorCallbacks.push(callback);
  }

  notifyDeviceAvailable(device: CastDevice): void {
    for (const callback of this.deviceAvailableCallbacks) {
      callback(device);
    }
  }

  notifySessionStateChange(session: CastSession): void {
    for (const callback of this.sessionStateCallbacks) {
      callback(session);
    }
  }

  notifyError(error: CastError): void {
    console.error('[Cast] Error:', error);
    for (const callback of this.errorCallbacks) {
      callback(error);
    }
  }

  removeAllListeners(): void {
    this.deviceAvailableCallbacks = [];
    this.sessionStateCallbacks = [];
    this.errorCallbacks = [];
  }
}
