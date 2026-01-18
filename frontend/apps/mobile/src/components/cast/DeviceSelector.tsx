import React, { useState } from 'react';
import { useCastStore } from '../../stores/castStore';
import { usePlaybackStore } from '../../stores/playbackStore';
import type { CastDevice } from '../../types/casting';

interface DeviceSelectorProps {
  isOpen: boolean;
  onClose: () => void;
}

export const DeviceSelector: React.FC<DeviceSelectorProps> = ({ isOpen, onClose }) => {
  const {
    availableDevices,
    currentSession,
    isConnected,
    connectToDevice,
    disconnectFromDevice,
    discoverDevices,
    castCurrentSong
  } = useCastStore();

  const currentSong = usePlaybackStore((state) => state.currentSong);

  const [isConnecting, setIsConnecting] = useState<string | null>(null);
  const [isCasting, setIsCasting] = useState(false);

  const handleDeviceSelect = async (device: CastDevice) => {
    setIsConnecting(device.id);
    try {
      if (isConnected && currentSession) {
        await disconnectFromDevice(currentSession.deviceId);
      }
      await connectToDevice(device.id);
      // Don't close modal immediately - let user manually cast
    } catch (error) {
      console.error('Failed to connect to device:', error);
    } finally {
      setIsConnecting(null);
    }
  };

  const handleDisconnect = async () => {
    if (currentSession) {
      try {
        await disconnectFromDevice(currentSession.deviceId);
        onClose();
      } catch (error) {
        console.error('Failed to disconnect:', error);
      }
    }
  };

  const handleCastCurrentSong = async () => {
    if (!currentSong || !isConnected) return;
    
    setIsCasting(true);
    try {
      await castCurrentSong(currentSong);
      console.log('✅ Successfully cast current song');
    } catch (error) {
      console.error('Failed to cast current song:', error);
      
      // Show user-friendly error for YouTube content
      if (error instanceof Error && error.message.includes('YouTube')) {
        // Could show a toast or modal here explaining the limitation
        console.log('💡 YouTube casting requires a custom receiver - this is a known limitation');
      }
    } finally {
      setIsCasting(false);
    }
  };

  const handleRefresh = async () => {
    console.log('🔄 Refreshing devices...');
    
    // Import cast manager for debugging
    const { castManager } = await import('../../services/castManager');
    
    // Log debug info
    console.log('Cast Debug Info:', castManager.getDebugInfo());
    
    // Force discovery
    await castManager.forceDiscovery();
    
    // Refresh devices in store
    await discoverDevices();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50 transition-colors duration-200 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-80 max-w-sm mx-4 shadow-xl border-2 border-gray-200 dark:border-gray-700 transition-colors duration-200">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Cast to Device</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 transition-colors duration-200"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Current connection */}
        {isConnected && currentSession && (
          <div className="mb-4 p-3 bg-primary/10 border border-primary/20 rounded-lg dark:bg-primary/20 dark:border-primary/30 transition-colors duration-200">
            <div className="flex items-center justify-between mb-3">
              <div>
                <div className="font-medium text-primary dark:text-primary-light">Connected</div>
                <div className="text-sm text-primary/80 dark:text-primary-light/80">{currentSession.deviceName}</div>
              </div>
              <button
                onClick={handleDisconnect}
                className="px-3 py-1 bg-primary text-white text-sm rounded hover:bg-primary/90 dark:bg-primary-light dark:hover:bg-primary transition-colors duration-200"
              >
                Disconnect
              </button>
            </div>
            
            {/* Cast Current Song Button */}
            {currentSong && (
              <div className="border-t border-primary/20 pt-3 dark:border-primary/30">
                <button
                  onClick={handleCastCurrentSong}
                  disabled={isCasting}
                  className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-200 dark:bg-primary-light dark:hover:bg-primary"
                >
                  {isCasting ? (
                    <>
                      <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      <span>Casting...</span>
                    </>
                  ) : (
                    <>
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z"/>
                      </svg>
                      <span>Cast Current Song</span>
                    </>
                  )}
                </button>
                <div className="text-xs text-primary/70 dark:text-primary-light/70 mt-1 text-center">
                  {currentSong.title}
                </div>
                
                {/* YouTube Limitation Notice */}
                {currentSong.sourceType === 'youtube' && (
                  <div className="mt-2 p-3 bg-blue-100 border border-blue-300 rounded text-blue-800 text-xs dark:bg-blue-900/20 dark:border-blue-700/30 dark:text-blue-400 transition-colors duration-200">
                    <div className="flex items-start gap-2">
                      <svg className="w-4 h-4 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
                      </svg>
                      <div>
                        <div className="font-medium">Custom Receiver Ready</div>
                        <div className="mt-1">YouTube content requires our custom receiver. Click "Cast Current Song" to see the demo.</div>
                        <div className="mt-2">
                          <a 
                            href="http://localhost:5173/cast-receiver.html" 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="text-blue-600 dark:text-blue-400 underline hover:no-underline"
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
                  onClick={async () => {
                    if (!isConnected) return;
                    
                    try {
                      // Test with a sample video that should work
                      const testMedia = {
                        contentId: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4',
                        contentType: 'video/mp4',
                        streamType: 'BUFFERED' as const,
                        metadata: {
                          title: 'Test Video - Big Buck Bunny',
                          artist: 'Blender Foundation',
                          images: [{
                            url: 'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/images/BigBuckBunny.jpg',
                            height: 480,
                            width: 640
                          }]
                        },
                        duration: 596
                      };
                      
                      await castCurrentSong(testMedia as any);
                      console.log('✅ Test video cast successfully');
                    } catch (error) {
                      console.error('Failed to cast test video:', error);
                    }
                  }}
                  className="w-full mt-2 flex items-center justify-center gap-2 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors duration-200 text-xs"
                >
                  <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8 5v14l11-7z"/>
                  </svg>
                  <span>Test Cast (Demo Video)</span>
                </button>
              </div>
            )}
          </div>
        )}

        {/* Available devices */}
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <h4 className="font-medium text-gray-900 dark:text-white">Available Devices</h4>
            <button
              onClick={handleRefresh}
              className="text-primary hover:text-primary/80 dark:text-primary-light dark:hover:text-primary text-sm transition-colors duration-200"
            >
              Refresh
            </button>
          </div>

          {availableDevices.length === 0 ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              <svg className="w-12 h-12 mx-auto mb-2 text-gray-300 dark:text-gray-600" fill="currentColor" viewBox="0 0 24 24">
                <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
              </svg>
              <div className="text-gray-500 dark:text-gray-400 font-medium">No Chromecast devices found</div>
              <div className="text-xs mt-2 text-gray-400 dark:text-gray-500 space-y-1">
                <div>Make sure your Chromecast is:</div>
                <div>• On the same Wi-Fi network</div>
                <div>• Powered on and ready</div>
                <div>• Not being used by another app</div>
              </div>
              <div className="text-xs mt-3 text-gray-400 dark:text-gray-500">
                Check browser console for debug info
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {availableDevices.map((device) => (
                <button
                  key={device.id}
                  onClick={() => handleDeviceSelect(device)}
                  disabled={isConnecting === device.id || (isConnected && currentSession?.deviceId === device.id)}
                  className={`
                    w-full p-3 text-left rounded-lg border transition-colors duration-200
                    ${isConnected && currentSession?.deviceId === device.id
                      ? 'bg-primary/10 border-primary/20 cursor-default dark:bg-primary/20 dark:border-primary/30'
                      : 'bg-white border-gray-200 hover:bg-gray-50 cursor-pointer dark:bg-gray-700 dark:border-gray-600 dark:hover:bg-gray-600'
                    }
                    ${isConnecting === device.id ? 'opacity-50' : ''}
                    focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800
                  `}
                >
                  <div className="flex items-center space-x-3">
                    <div className="shrink-0">
                      {device.type === 'chromecast' ? (
                        <svg className="w-6 h-6 text-gray-600 dark:text-gray-300" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
                        </svg>
                      ) : (
                        <svg className="w-6 h-6 text-gray-600 dark:text-gray-300" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                        </svg>
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="font-medium text-gray-900 dark:text-white">{device.name}</div>
                      <div className="text-sm text-gray-500 dark:text-gray-400 capitalize">{device.type}</div>
                    </div>
                    {isConnecting === device.id && (
                      <div className="shrink-0">
                        <svg className="w-4 h-4 animate-spin text-primary dark:text-primary-light" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      </div>
                    )}
                    {isConnected && currentSession?.deviceId === device.id && (
                      <div className="shrink-0">
                        <svg className="w-4 h-4 text-primary dark:text-primary-light" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                        </svg>
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-6 text-xs text-gray-500 dark:text-gray-400 text-center transition-colors duration-200">
          Make sure your device supports Google Cast and is connected to the same Wi-Fi network.
        </div>
      </div>
    </div>
  );
};