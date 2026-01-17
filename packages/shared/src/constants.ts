// API versions
export const API_VERSION = 'v1';
export const API_BASE_PATH = `/api/${API_VERSION}`;

// Playback sync
export const PLAYBACK_SYNC_INTERVAL_MS = 5000;
export const PLAYBACK_DRIFT_THRESHOLD_MS = 2000;

// Room settings limits
export const MAX_ROOM_NAME_LENGTH = 100;
export const MIN_SKIP_VOTE_THRESHOLD = 0.1;
export const MAX_SKIP_VOTE_THRESHOLD = 1.0;
export const MIN_CONTINUOUS_ADDS = 1;
export const MAX_CONTINUOUS_ADDS = 10;

// SSE
export const SSE_RECONNECT_DELAY_MS = 1000;
export const SSE_MAX_RECONNECT_DELAY_MS = 30000;
export const SSE_HEARTBEAT_INTERVAL_MS = 30000;
