import type { CastDevice } from '@vibez/models';
import { safeWrapAsync, usePlaybackStore } from '@vibez/shared';
import React, { useState } from 'react';
import { useCastStore } from '../../stores/castStore';

interface DeviceSelectorProps {
  isOpen: boolean;
  onClose: () => void;
}

export const DeviceSelector: React.FC<DeviceSelectorProps> = ({
  isOpen,
  onClose,
}) => {
  const {
    availableDevices,
    currentSession,
    isConnected,
    connectToDevice,
    disconnectFromDevice,
    discoverDevices,
    castCurrentSong,
  } = useCastStore();

  const currentSong = usePlaybackStore((state) => state.currentSong);

  const [isConnecting, setIsConnecting] = useState<string | null>(null);
  const [isCasting, setIsCasting] = useState(false);

  const handleDeviceSelect = async (device: CastDevice) => {
    setIsConnecting(device.id);

    if (isConnected && currentSession) {
      await safeWrapAsync(disconnectFromDevice(currentSession.deviceId));
    }

    const [err] = await safeWrapAsync(connectToDevice(device.id));
    if (err) console.error('Failed to connect to device:', err);

    setIsConnecting(null);
  };

  const handleDisconnect = async () => {
    if (!currentSession) return;

    const [err] = await safeWrapAsync(
      disconnectFromDevice(currentSession.deviceId),
    );
    if (err) {
      console.error('Failed to disconnect:', err);
      return;
    }
    onClose();
  };

  const handleCastCurrentSong = async (media?: any) => {
    const songToCast = media || currentSong;
    if (!songToCast || !isConnected) return;

    setIsCasting(true);
    const [err] = await safeWrapAsync(castCurrentSong(songToCast));

    if (err) {
      console.error('Failed to cast:', err);
      if (err.message.includes('YouTube')) {
        console.log(
          '💡 YouTube casting requires a custom receiver - this is a known limitation',
        );
      }
    } else {
      console.log('✅ Successfully cast');
    }

    setIsCasting(false);
  };

  const handleRefresh = async () => {
    console.log('🔄 Refreshing devices...');
    const { castManager } = await import('../../services/castManager');

    console.log('Cast Debug Info:', castManager.getDebugInfo());
    await safeWrapAsync(castManager.forceDiscovery());
    await safeWrapAsync(discoverDevices());
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm transition-colors duration-200 dark:bg-black/70">
      <div className="mx-4 w-80 max-w-sm rounded-lg border-2 border-gray-200 bg-white p-6 shadow-xl transition-colors duration-200 dark:border-gray-700 dark:bg-gray-800">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-semibold text-gray-900 text-lg dark:text-white">
            Cast to Device
          </h3>
          <button
            onClick={onClose}
            className="cursor-pointer text-gray-400 transition-colors duration-200 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            <svg
              className="h-6 w-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Current connection */}
        {isConnected && currentSession && (
          <div className="mb-4 rounded-lg border border-primary/20 bg-primary/10 p-3 transition-colors duration-200 dark:border-primary/30 dark:bg-primary/20">
            <div className="mb-3 flex items-center justify-between">
              <div>
                <div className="font-medium text-primary dark:text-primary-light">
                  Connected
                </div>
                <div className="text-primary/80 text-sm dark:text-primary-light/80">
                  {currentSession.deviceName}
                </div>
              </div>
              <button
                onClick={handleDisconnect}
                className="cursor-pointer rounded bg-primary px-3 py-1 text-sm text-white transition-colors duration-200 hover:bg-primary/90 dark:bg-primary-light dark:hover:bg-primary"
              >
                Disconnect
              </button>
            </div>

            {/* Cast Current Song Button */}
            {currentSong && (
              <div className="border-primary/20 border-t pt-3 dark:border-primary/30">
                <button
                  onClick={handleCastCurrentSong}
                  disabled={isCasting}
                  className="flex w-full cursor-pointer items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-white transition-colors duration-200 hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-primary-light dark:hover:bg-primary"
                >
                  {isCasting ? (
                    <>
                      <svg
                        className="h-4 w-4 animate-spin"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          className="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          strokeWidth="4"
                        ></circle>
                        <path
                          className="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        ></path>
                      </svg>
                      <span>Casting...</span>
                    </>
                  ) : (
                    <>
                      <svg
                        className="h-4 w-4"
                        fill="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path d="M8 5v14l11-7z" />
                      </svg>
                      <span>Cast Current Song</span>
                    </>
                  )}
                </button>
                <div className="mt-1 text-center text-primary/70 text-xs dark:text-primary-light/70">
                  {currentSong.title}
                </div>

                {/* YouTube Limitation Notice */}
                {currentSong.sourceType === 'youtube' && (
                  <div className="mt-2 rounded border border-blue-300 bg-blue-100 p-3 text-blue-800 text-xs transition-colors duration-200 dark:border-blue-700/30 dark:bg-blue-900/20 dark:text-blue-400">
                    <div className="flex items-start gap-2">
                      <svg
                        className="mt-0.5 h-4 w-4 shrink-0"
                        fill="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z" />
                      </svg>
                      <div>
                        <div className="font-medium">Custom Receiver Ready</div>
                        <div className="mt-1">
                          YouTube content requires our custom receiver. Click
                          "Cast Current Song" to see the demo.
                        </div>
                        <div className="mt-2">
                          <a
                            href="/casting/receiver/"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 underline hover:no-underline dark:text-blue-400"
                          >
                            View Custom Receiver →
                          </a>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Test Cast Button for Demo */}
                <button
                  onClick={() => {
                    const testMedia = {
                      contentId:
                        'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4',
                      contentType: 'video/mp4',
                      streamType: 'BUFFERED' as const,
                      metadata: {
                        title: 'Test Video - Big Buck Bunny',
                        artist: 'Blender Foundation',
                        images: [
                          {
                            url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/images/BigBuckBunny.jpg',
                            height: 480,
                            width: 640,
                          },
                        ],
                      },
                      duration: 596,
                    };
                    handleCastCurrentSong(testMedia);
                  }}
                  className="mt-2 flex w-full items-center justify-center gap-2 rounded-lg bg-gray-600 px-4 py-2 text-white text-xs transition-colors duration-200 hover:bg-gray-700"
                >
                  <svg
                    className="h-3 w-3"
                    fill="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path d="M8 5v14l11-7z" />
                  </svg>
                  <span>Test Cast (Demo Video)</span>
                </button>
              </div>
            )}
          </div>
        )}

        {/* Available devices */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h4 className="font-medium text-gray-900 dark:text-white">
              Available Devices
            </h4>
            <button
              onClick={handleRefresh}
              className="cursor-pointer text-primary text-sm transition-colors duration-200 hover:text-primary/80 dark:text-primary-light dark:hover:text-primary"
            >
              Refresh
            </button>
          </div>

          {availableDevices.length === 0 ? (
            <div className="py-8 text-center text-gray-500 dark:text-gray-400">
              <svg
                className="mx-auto mb-2 h-12 w-12 text-gray-300 dark:text-gray-600"
                fill="currentColor"
                viewBox="0 0 24 24"
              >
                <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z" />
              </svg>
              <div className="font-medium text-gray-500 dark:text-gray-400">
                No Chromecast devices found
              </div>
              <div className="mt-2 space-y-1 text-gray-400 text-xs dark:text-gray-500">
                <div>Make sure your Chromecast is:</div>
                <div>• On the same Wi-Fi network</div>
                <div>• Powered on and ready</div>
                <div>• Not being used by another app</div>
              </div>
              <div className="mt-3 text-gray-400 text-xs dark:text-gray-500">
                Check browser console for debug info
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {availableDevices.map((device) => (
                <button
                  key={device.id}
                  onClick={() => handleDeviceSelect(device)}
                  disabled={
                    isConnecting === device.id ||
                    (isConnected && currentSession?.deviceId === device.id)
                  }
                  className={`w-full rounded-lg border p-3 text-left transition-colors duration-200 ${
                    isConnected && currentSession?.deviceId === device.id
                      ? 'cursor-default border-primary/20 bg-primary/10 dark:border-primary/30 dark:bg-primary/20'
                      : 'cursor-pointer border-gray-200 bg-white hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:hover:bg-gray-600'
                  }
                    ${isConnecting === device.id ? 'opacity-50' : ''}focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800`}
                >
                  <div className="flex items-center space-x-3">
                    <div className="shrink-0">
                      {device.type === 'chromecast' ? (
                        <svg
                          className="h-6 w-6 text-gray-600 dark:text-gray-300"
                          fill="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z" />
                        </svg>
                      ) : (
                        <svg
                          className="h-6 w-6 text-gray-600 dark:text-gray-300"
                          fill="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
                        </svg>
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="font-medium text-gray-900 dark:text-white">
                        {device.name}
                      </div>
                      <div className="text-gray-500 text-sm capitalize dark:text-gray-400">
                        {device.type}
                      </div>
                    </div>
                    {isConnecting === device.id && (
                      <div className="shrink-0">
                        <svg
                          className="h-4 w-4 animate-spin text-primary dark:text-primary-light"
                          fill="none"
                          viewBox="0 0 24 24"
                        >
                          <circle
                            className="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            strokeWidth="4"
                          ></circle>
                          <path
                            className="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                          ></path>
                        </svg>
                      </div>
                    )}
                    {isConnected && currentSession?.deviceId === device.id && (
                      <div className="shrink-0">
                        <svg
                          className="h-4 w-4 text-primary dark:text-primary-light"
                          fill="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                        </svg>
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-6 text-center text-gray-500 text-xs transition-colors duration-200 dark:text-gray-400">
          Make sure your device supports Google Cast and is connected to the
          same Wi-Fi network.
        </div>
      </div>
    </div>
  );
};
