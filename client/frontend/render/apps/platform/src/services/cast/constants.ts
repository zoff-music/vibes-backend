// Cast SDK configuration constants

// Google Cast Application ID - Custom Zoff Receiver
// For development, we use the Styled Media Receiver which allows custom content
// In production, this should be replaced with a registered custom receiver app ID
export const CAST_APPLICATION_ID =
  import.meta.env.VITE_CAST_APP_ID || '1FAF5D9F';

// Development: Use local custom receiver
// Production: Use registered custom receiver
export const DEVELOPMENT_MODE =
  import.meta.env.VITE_DEVELOPMENT_MODE === 'true';

export const CUSTOM_RECEIVER_URL =
  import.meta.env.VITE_CAST_RECEIVER_URL || '/casting/receiver/';

export const LOCAL_EMULATOR_ENABLED = (() => {
  const envValue = import.meta.env.VITE_CAST_LOCAL_EMULATOR;
  if (envValue === 'true' || envValue === '1') return true;
  if (typeof window === 'undefined') return false;
  return (
    window.location.hostname === 'localhost' ||
    window.location.hostname === '127.0.0.1'
  );
})();

export const LOCAL_EMULATOR_DEVICE_ID = 'local-cast-emulator';
