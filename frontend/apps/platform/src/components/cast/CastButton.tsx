import { safeWrapAsync } from '@vibez/shared';
import React, { useEffect } from 'react';
import { useCastStore } from '../../stores/castStore';
import { CastIcon } from '../icons/CastIcon';

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
    if (lastError) clearError();

    if (isConnected && currentSession) {
      if (onDeviceSelect) {
        onDeviceSelect();
      } else {
        const [err] = await safeWrapAsync(
          disconnectFromDevice(currentSession.deviceId),
        );
        if (err) console.error('Failed to disconnect:', err);
      }
      return;
    }

    if (availableDevices.length > 0) {
      if (onDeviceSelect) {
        onDeviceSelect();
      } else {
        const [err] = await safeWrapAsync(
          connectToDevice(availableDevices[0].id),
        );
        if (err) console.error('Failed to connect:', err);
      }
      return;
    }

    const [err] = await safeWrapAsync(discoverDevices());
    if (err) console.error('Cast discovery error:', err);
  };

  const getCastIcon = () => {
    const iconClasses = isConnected
      ? 'w-6 h-6 text-white'
      : 'w-6 h-6 text-gray-600 dark:text-gray-300 group-hover:text-primary dark:group-hover:text-primary-light transition-colors duration-200';

    if (isConnected) {
      return <CastIcon className={iconClasses} isActive={true} />;
    }

    return <CastIcon className={iconClasses} />;
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
