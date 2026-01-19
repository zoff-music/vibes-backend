import React, { useEffect } from 'react';
import { useCastStore } from '../../stores/castStore';

interface CastButtonProps {
  onDeviceSelect?: () => void;
  className?: string;
}

export const CastButton: React.FC<CastButtonProps> = ({
  onDeviceSelect,
  className = '',
}) => {
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
    clearError,
  } = useCastStore();

  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  const handleCastClick = async () => {
    try {
      if (lastError) {
        clearError();
      }

      if (isConnected && currentSession) {
        // If connected, show device selector for disconnect/switch options
        if (onDeviceSelect) {
          onDeviceSelect();
        } else {
          // Fallback: disconnect from current device
          await disconnectFromDevice(currentSession.deviceId);
        }
      } else if (availableDevices.length > 0) {
        if (onDeviceSelect) {
          // Show device selector to choose device
          onDeviceSelect();
        } else {
          // Fallback: connect to first available device
          await connectToDevice(availableDevices[0].id);
        }
      } else {
        // Try to discover devices
        await discoverDevices();
      }
    } catch (error) {
      console.error('Cast button click error:', error);
    }
  };

  const getCastIcon = () => {
    const iconClasses = isConnected
      ? 'w-6 h-6 text-white'
      : 'w-6 h-6 text-gray-600 dark:text-gray-300 group-hover:text-primary dark:group-hover:text-primary-light transition-colors duration-200';

    if (isConnected) {
      return (
        <svg
          className={iconClasses}
          fill="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z" />
          <circle cx="6" cy="18" r="2" />
        </svg>
      );
    }

    return (
      <svg
        className={iconClasses}
        fill="currentColor"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z" />
      </svg>
    );
  };

  const getButtonText = () => {
    if (isConnected && currentSession) {
      return `Connected`;
    }
    if (availableDevices.length > 0) {
      return 'Cast';
    }
    return 'No devices';
  };

  const isDisabled =
    !isInitialized || (availableDevices.length === 0 && !isConnected);

  return (
    <div className="flex flex-col items-center">
      <button
        onClick={handleCastClick}
        disabled={isDisabled}
        className={`group flex items-center space-x-2 rounded-lg px-4 py-2 transition-all duration-200 ease-in-out ${
          isConnected
            ? 'bg-primary text-white shadow-lg hover:bg-primary-dark dark:bg-primary-light dark:hover:bg-primary'
            : 'bg-gray-200 text-gray-900 hover:bg-gray-300 dark:bg-gray-700 dark:text-white dark:hover:bg-gray-600'
        }
          ${
            isDisabled
              ? 'cursor-not-allowed opacity-50'
              : 'cursor-pointer hover:scale-105 active:scale-95'
          }border-2 border-transparent ${
            isConnected
              ? 'hover:border-primary-dark/20 dark:hover:border-primary/30'
              : 'hover:border-primary/20 dark:hover:border-primary-light/30'
          }focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800 ${className}
        `}
        aria-label={
          isConnected
            ? `Connected to ${currentSession?.deviceName}`
            : 'Cast to device'
        }
      >
        {getCastIcon()}
        <span className="font-medium text-sm transition-colors duration-200">
          {getButtonText()}
        </span>
      </button>

      {/* Error Display */}
      {lastError && (
        <div className="mt-2 rounded border border-red-300 bg-red-100 p-2 text-red-700 text-xs transition-colors duration-200 dark:border-red-700/30 dark:bg-red-900/20 dark:text-red-400">
          {lastError.description}
        </div>
      )}

      {/* Connection Status */}
      {isConnected && currentSession && (
        <div className="mt-1 text-center text-primary text-xs transition-colors duration-200 dark:text-primary-light">
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 animate-pulse rounded-full bg-primary dark:bg-primary-light"></div>
            <span>Casting to {currentSession.deviceName}</span>
          </div>
        </div>
      )}
    </div>
  );
};
