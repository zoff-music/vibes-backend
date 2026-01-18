import React, { useEffect } from 'react';
import { useCastStore } from '../../stores/castStore';

interface CastButtonProps {
  className?: string;
}

export const CastButton: React.FC<CastButtonProps> = ({ className = '' }) => {
  const {
    isInitialized,
    availableDevices,
    isConnected,
    currentSession,
    initialize,
    discoverDevices,
    connectToDevice,
    disconnectFromDevice,
    lastError,
    clearError
  } = useCastStore();

  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  const handleCastClick = async () => {
    if (lastError) {
      clearError();
    }

    if (isConnected && currentSession) {
      // Disconnect from current device
      await disconnectFromDevice(currentSession.deviceId);
    } else if (availableDevices.length > 0) {
      // Connect to first available device
      await connectToDevice(availableDevices[0].id);
    } else {
      // Try to discover devices
      await discoverDevices();
    }
  };

  const getCastIcon = () => {
    if (isConnected) {
      return (
        <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
          <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
          <circle cx="6" cy="18" r="2"/>
        </svg>
      );
    }

    return (
      <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
        <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
      </svg>
    );
  };

  const getButtonText = () => {
    if (isConnected && currentSession) {
      return `Connected to ${currentSession.deviceName}`;
    }
    if (availableDevices.length > 0) {
      return 'Cast';
    }
    return 'No devices';
  };

  const isDisabled = !isInitialized || (availableDevices.length === 0 && !isConnected);

  return (
    <div className="flex flex-col items-center">
      <button
        onClick={handleCastClick}
        disabled={isDisabled}
        className={`
          flex items-center space-x-2 px-4 py-2 rounded-lg transition-colors
          ${isConnected 
            ? 'bg-blue-600 text-white hover:bg-blue-700' 
            : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
          }
          ${isDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
          ${className}
        `}
      >
        {getCastIcon()}
        <span className="text-sm font-medium">{getButtonText()}</span>
      </button>

      {lastError && (
        <div className="mt-2 p-2 bg-red-100 border border-red-300 rounded text-red-700 text-xs">
          {lastError.description}
        </div>
      )}

      {isConnected && currentSession && (
        <div className="mt-1 text-xs text-gray-500">
          Casting to {currentSession.deviceName}
        </div>
      )}
    </div>
  );
};