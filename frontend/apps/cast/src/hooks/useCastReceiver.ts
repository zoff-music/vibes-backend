import { setCachedToken } from '@vibez/api';
import type { Song } from '@vibez/shared';
import { safeWrap, usePlaybackStore } from '@vibez/shared';
import type { framework } from 'chromecast-caf-receiver';
import { useEffect, useRef } from 'react';
import type { LocalCastMessage } from '../types';
import { normalizeSong } from '../utils/songUtils';

let isCastReceiverInitialized = false;

interface TokenData {
  token: string;
  expiresAt?: string;
}

interface CustomLoadData {
  tokens?: Record<string, TokenData>;
  debug?: boolean;
  currentSong?: Song;
  positionMs?: number;
}

function isCustomLoadData(data: unknown): data is CustomLoadData {
  return data !== null && typeof data === 'object';
}

function isLocalCastMessage(msg: unknown): msg is LocalCastMessage {
  return !!msg && typeof msg === 'object' && 'action' in msg;
}

interface CastSender {
  id: string;
}

type CastReceiverContextWithSenders = framework.CastReceiverContext & {
  getSenders: () => CastSender[];
  sendCustomMessage: (
    namespace: string,
    senderId: string,
    message: string,
  ) => void;
};

interface CastReceiverContextEventTypes {
  SENDER_CONNECTED: string;
}

type CastFrameworkWithEvents = typeof cast.framework & {
  CastReceiverContextEventType?: CastReceiverContextEventTypes;
};

interface UseCastReceiverProps {
  debugMode: boolean;
  setDebugMode: (mode: boolean) => void;
  handleCastMessage: (msg: LocalCastMessage) => void;
  updateMediaMetadata: (song: Song) => void;
  setStatusText: (text: string) => void;
}

export const useCastReceiver = ({
  debugMode,
  setDebugMode,
  handleCastMessage,
  updateMediaMetadata,
  setStatusText,
}: UseCastReceiverProps) => {
  const setPlaybackState = usePlaybackStore((state) => state.setPlaybackState);
  const setIsPlaying = usePlaybackStore((state) => state.setIsPlaying);

  const contextRef = useRef<framework.CastReceiverContext | null>(null);
  const playerManagerRef = useRef<framework.PlayerManager | null>(null);

  useEffect(() => {
    const initCast = () => {
      // Prevent double initialization using global flag
      if (isCastReceiverInitialized) {
        return;
      }

      if (!window.cast || !window.cast.framework) return;

      const context = cast.framework.CastReceiverContext.getInstance();

      if (debugMode) {
        context.setLoggerLevel(cast.framework.LoggerLevel.DEBUG);
      }

      const playerManager = context.getPlayerManager();

      contextRef.current = context;
      playerManagerRef.current = playerManager;

      playerManager.setMessageInterceptor(
        cast.framework.messages.MessageType.LOAD,
        (loadRequestData) => {
          // Return the request data as-is if it's our technical URL
          // but if it's a "real" media request from the sender, we handle it
          // Returning null CAN cause session_error on some senders,
          // but it's the official way to tell CAF "I got this, don't play anything".

          const media = loadRequestData.media;
          if (media?.customData && isCustomLoadData(media.customData)) {
            const data = media.customData;
            if (data.tokens) {
              for (const [provider, tokenData] of Object.entries(data.tokens)) {
                if (tokenData?.token) {
                  setCachedToken(
                    provider,
                    tokenData.token,
                    tokenData.expiresAt ||
                      new Date(Date.now() + 3600000).toISOString(),
                  );
                }
              }
            }

            if (data.debug) setDebugMode(true);

            if (data.currentSong) {
              const normalizedSong = normalizeSong(data.currentSong);
              setPlaybackState({
                currentSong: normalizedSong,
                isPlaying: true,
                positionMs: data.positionMs || 0,
                updatedAt: new Date().toISOString(),
                serverTimeMs: Date.now(),
              });
              setIsPlaying(true);
              setStatusText(`Now Playing: ${normalizedSong.title}`);
              updateMediaMetadata(normalizedSong);
            }
          }

          // Return null to prevent the default player from trying to load this as media
          return null;
        },
      );

      context.addCustomMessageListener(
        'urn:x-cast:com.vibez.cast',
        (customEvent) => {
          const message = customEvent.data;
          if (isLocalCastMessage(message)) {
            handleCastMessage(message);
          }
        },
      );

      const options = new cast.framework.CastReceiverOptions();
      options.maxInactivity = 3600;
      options.statusText = 'Zoff';
      options.disableIdleTimeout = true;

      safeWrap(() => {
        context.start(options);
        isCastReceiverInitialized = true;
      });

      const contextWithSenders = context as CastReceiverContextWithSenders;

      const sendReceiverReady = () => {
        const [senderErr, senders] = safeWrap(() =>
          contextWithSenders.getSenders(),
        );
        if (senderErr) {
          console.error('Failed to get cast senders:', senderErr);
          return;
        }

        const message = JSON.stringify({
          action: 'receiverReady',
          timestamp: Date.now(),
        });

        senders?.forEach((sender) => {
          const [sendErr] = safeWrap(() => {
            contextWithSenders.sendCustomMessage(
              'urn:x-cast:com.vibez.cast',
              sender.id,
              message,
            );
          });
          if (sendErr) {
            console.error('Failed to send receiverReady message:', sendErr);
          }
        });
      };

      sendReceiverReady();

      const frameworkEvents = cast.framework as CastFrameworkWithEvents;
      const senderConnectedEvent =
        frameworkEvents.CastReceiverContextEventType?.SENDER_CONNECTED;
      if (senderConnectedEvent) {
        // Cast to any to work around Cast SDK type limitations
        (
          contextWithSenders as unknown as {
            addEventListener: (type: unknown, handler: () => void) => void;
          }
        ).addEventListener(senderConnectedEvent, () => {
          sendReceiverReady();
        });
      }
    };

    if (window.cast?.framework) {
      initCast();
    }
  }, [
    debugMode,
    handleCastMessage,
    setDebugMode,
    setIsPlaying,
    setPlaybackState,
    setStatusText,
    updateMediaMetadata,
  ]);

  return { contextRef, playerManagerRef };
};
