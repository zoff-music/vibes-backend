import { API_BASE_URL } from '@vibez/api';
import type { Song } from '@vibez/shared';
import { safeWrap, usePlaybackStore } from '@vibez/shared';
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import { useCastMessageHandler } from '../hooks/useCastMessageHandler';
import { useCastReceiver } from '../hooks/useCastReceiver';
import { useMediaMetadata } from '../hooks/useMediaMetadata';
import { useRoomSync } from '../hooks/useRoomSync';
import type { LocalCastMessage, QueueItem, RoomInfo } from '../types';

interface CastContextType {
  roomInfo: RoomInfo | null;
  queue: QueueItem[];
  statusText: string;
  roomMode: string | null;
  currentSong: Song | null;
  actualPositionMs: number;
  updateActualPosition: () => void;
  debugMode: boolean;
  roomId: string | null;
  casterId: string | null;
  castToken: string | null;
  error: string | null;
  apiUrl: string;
}

const CastContext = createContext<CastContextType | undefined>(undefined);

export function CastProvider({ children }: { children: React.ReactNode }) {
  // --- State ---
  const [roomInfo, setRoomInfo] = useState<RoomInfo | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [statusText, setStatusText] = useState('Ready for Casting');
  const [roomMode, setRoomMode] = useState<string | null>(null);
  const [debugMode, setDebugModeState] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('debug') === 'true';
  });

  // Room ID State
  const [roomId, setRoomId] = useState<string | null>(() => {
    return new URLSearchParams(window.location.search).get('roomId');
  });
  const [casterId, setCasterId] = useState<string | null>(() => {
    const params = new URLSearchParams(window.location.search);
    return (
      params.get('casterId') ||
      params.get('casterUserId') ||
      params.get('sessionId')
    );
  });
  const [castToken, setCastToken] = useState<string | null>(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('castToken');
  });

  const debugModeRef = useRef(debugMode);
  const setDebugMode = useCallback((value: boolean) => {
    setDebugModeState(value);
    debugModeRef.current = value;
  }, []);

  // --- Store ---
  const updateActualPosition = usePlaybackStore(
    (state) => state.updateActualPosition,
  );
  const actualPositionMs = usePlaybackStore((state) => state.actualPositionMs);
  const currentSong = usePlaybackStore((state) => state.currentSong);

  // --- Hooks ---
  const updateMediaMetadata = useMediaMetadata();

  const handleCastMessage = useCastMessageHandler({
    setRoomId,
    setCasterId,
    setCastToken,
    setRoomInfo,
    setQueue,
    setStatusText,
    updateMediaMetadata,
    roomMode,
  });

  useCastReceiver({
    debugMode,
    setDebugMode,
    handleCastMessage,
    updateMediaMetadata,
    setStatusText,
  });

  useRoomSync({
    roomId,
    casterId,
    castToken,
    setQueue,
    setRoomInfo,
    setStatusText,
    setRoomMode,
    setError,
    updateMediaMetadata,
    debugMode,
  });

  // --- Effects ---

  // Update actual position interval
  useEffect(() => {
    const interval = setInterval(() => {
      updateActualPosition();
    }, 500);

    return () => clearInterval(interval);
  }, [updateActualPosition]);

  // Local window message listener (emulator/dev)
  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) {
        const [parseErr, originUrl] = safeWrap(() => new URL(event.origin));
        if (parseErr || !originUrl) return;
        const receiverUrl = new URL(window.location.origin);
        const isLocalOrigin =
          (originUrl.hostname === 'localhost' ||
            originUrl.hostname === '127.0.0.1') &&
          (receiverUrl.hostname === 'localhost' ||
            receiverUrl.hostname === '127.0.0.1');
        if (!isLocalOrigin) return;
      }
      const data = event.data as LocalCastMessage | null;
      if (!data || typeof data !== 'object') return;
      if (!('action' in data)) return;

      handleCastMessage(data);
    };

    window.addEventListener('message', onMessage);
    return () => {
      window.removeEventListener('message', onMessage);
    };
  }, [handleCastMessage]);

  useEffect(() => {
    const opener = window.opener;
    if (!opener) return;

    const [err, referrerUrl] = safeWrap(() => new URL(document.referrer));
    const targetOrigin = !err && referrerUrl ? referrerUrl.origin : '*';

    opener.postMessage(
      {
        action: 'receiverReady',
        timestamp: Date.now(),
      },
      targetOrigin,
    );
  }, []);

  return (
    <CastContext.Provider
      value={{
        roomInfo,
        queue,
        statusText,
        roomMode,
        currentSong,
        actualPositionMs,
        updateActualPosition,
        debugMode,
        roomId: roomId || null,
        casterId: casterId || null,
        castToken: castToken || null,
        error,
        apiUrl: API_BASE_URL,
      }}
    >
      {children}
    </CastContext.Provider>
  );
}

export function useCast() {
  const context = useContext(CastContext);
  if (context === undefined) {
    throw new Error('useCast must be used within a CastProvider');
  }
  return context;
}
