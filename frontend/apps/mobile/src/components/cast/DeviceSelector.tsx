import React, { useState } from 'react';
import { useCastStore } from '../../stores/castStore';
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
    discoverDevices
  } = useCastStore();

  const [isConnecting, setIsConnecting] = useState<string | null>(null);

  const handleDeviceSelect = async (device: CastDevice) => {
    setIsConnecting(device.id);
    try {
      if (isConnected && currentSession) {
        await disconnectFromDevice(currentSession.deviceId);
      }
      await connectToDevice(device.id);
      onClose();
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

  const handleRefresh = async () => {
    await discoverDevices();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-80 max-w-sm mx-4">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold">Cast to Device</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Current connection */}
        {isConnected && currentSession && (
          <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium text-blue-900">Connected</div>
                <div className="text-sm text-blue-700">{currentSession.deviceName}</div>
              </div>
              <button
                onClick={handleDisconnect}
                className="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                Disconnect
              </button>
            </div>
          </div>
        )}

        {/* Available devices */}
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <h4 className="font-medium text-gray-900">Available Devices</h4>
            <button
              onClick={handleRefresh}
              className="text-blue-600 hover:text-blue-700 text-sm"
            >
              Refresh
            </button>
          </div>

          {availableDevices.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <svg className="w-12 h-12 mx-auto mb-2 text-gray-300" fill="currentColor" viewBox="0 0 24 24">
                <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
              </svg>
              <div>No devices found</div>
              <div className="text-xs mt-1">Make sure your casting device is on the same network</div>
            </div>
          ) : (
            <div className="space-y-2">
              {availableDevices.map((device) => (
                <button
                  key={device.id}
                  onClick={() => handleDeviceSelect(device)}
                  disabled={isConnecting === device.id || (isConnected && currentSession?.deviceId === device.id)}
                  className={`
                    w-full p-3 text-left rounded-lg border transition-colors
                    ${isConnected && currentSession?.deviceId === device.id
                      ? 'bg-blue-50 border-blue-200 cursor-default'
                      : 'bg-white border-gray-200 hover:bg-gray-50 cursor-pointer'
                    }
                    ${isConnecting === device.id ? 'opacity-50' : ''}
                  `}
                >
                  <div className="flex items-center space-x-3">
                    <div className="flex-shrink-0">
                      {device.type === 'chromecast' ? (
                        <svg className="w-6 h-6 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
                        </svg>
                      ) : (
                        <svg className="w-6 h-6 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
                          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                        </svg>
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="font-medium text-gray-900">{device.name}</div>
                      <div className="text-sm text-gray-500 capitalize">{device.type}</div>
                    </div>
                    {isConnecting === device.id && (
                      <div className="flex-shrink-0">
                        <svg className="w-4 h-4 animate-spin text-blue-600" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      </div>
                    )}
                    {isConnected && currentSession?.deviceId === device.id && (
                      <div className="flex-shrink-0">
                        <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
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

        <div className="mt-6 text-xs text-gray-500 text-center">
          Make sure your device supports Google Cast and is connected to the same Wi-Fi network.
        </div>
      </div>
    </div>
  );
};