/// <reference types="chromecast-caf-receiver" />

import { safeWrap } from '@vibez/shared';

let isGlobalDebugEnabled = false;

export const setGlobalDebug = (enabled: boolean) => {
  isGlobalDebugEnabled = enabled;
  if (typeof window !== 'undefined' && window.cast?.framework) {
    const castDebugLogger = cast.debug.CastDebugLogger.getInstance();
    castDebugLogger.setEnabled(enabled);
    castDebugLogger.showDebugLogs(enabled);
    console.log(`[GlobalLogger] Debug mode set to: ${enabled}`);
  }
};

// Initialize logging immediately
const initLogging = () => {
  if (typeof window === 'undefined' || !window.cast?.framework) {
    console.warn('Cast framework not found, skipping global logger init');
    return;
  }

  const castDebugLogger = cast.debug.CastDebugLogger.getInstance();
  const LOG_TAG = 'Zoff';

  // Check URL params for initial state
  const params = new URLSearchParams(window.location.search);
  if (params.get('debug') === 'true') {
    setGlobalDebug(true);
  }

  castDebugLogger.loggerLevelByEvents = {
    'cast.framework.events.category.CORE': cast.framework.LoggerLevel.INFO,
    'cast.framework.events.EventType.MEDIA_STATUS':
      cast.framework.LoggerLevel.DEBUG,
  };
  castDebugLogger.loggerLevelByTags = {
    [LOG_TAG]: cast.framework.LoggerLevel.DEBUG,
  };

  const originalConsole = {
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
    debug: console.debug,
  };

  const serializeArgs = (args: unknown[]) => {
    return args.map((arg) => {
      if (typeof arg === 'object' && arg !== null) {
        try {
          return JSON.stringify(arg, (key, value) => {
            if (
              key === 'source' &&
              value &&
              typeof value === 'object' &&
              'tagName' in value
            )
              return '[DOM Element]';
            return value;
          });
        } catch (_e) {
          return String(arg);
        }
      }
      return String(arg);
    });
  };

  const sendLogToSender = (level: string, args: unknown[]) => {
    if (!isGlobalDebugEnabled && level !== 'error') return; // Only send errors if debug is off

    const ctx = cast.framework.CastReceiverContext.getInstance();
    const senders = ctx.getSenders();

    // Always try to send if we have context, even if senders list looks empty
    // The sender might be in the process of connecting
    const [err] = safeWrap(() => {
      ctx.sendCustomMessage('urn:x-cast:com.vibez.cast', undefined, {
        action: 'LOG',
        level,
        args: serializeArgs(args),
        timestamp: Date.now(),
        _meta: { sendersCount: senders.length },
      });
    });

    if (err) {
      // Silent fail to prevent recursion
    }
  };

  console.log = (...args) => {
    if (isGlobalDebugEnabled) castDebugLogger?.info(LOG_TAG, ...args);
    originalConsole.log(...args);
    sendLogToSender('info', args);
  };

  console.info = (...args) => {
    if (isGlobalDebugEnabled) castDebugLogger?.info(LOG_TAG, ...args);
    originalConsole.info(...args);
    sendLogToSender('info', args);
  };

  console.warn = (...args) => {
    if (isGlobalDebugEnabled) castDebugLogger?.warn(LOG_TAG, ...args);
    originalConsole.warn(...args);
    sendLogToSender('warn', args);
  };

  console.error = (...args) => {
    if (isGlobalDebugEnabled) castDebugLogger?.error(LOG_TAG, ...args);
    originalConsole.error(...args);
    sendLogToSender('error', args);
  };

  console.debug = (...args) => {
    if (isGlobalDebugEnabled) castDebugLogger?.debug(LOG_TAG, ...args);
    originalConsole.debug(...args);
    sendLogToSender('debug', args);
  };

  console.log('[GlobalLogger] Initialized early logging');
};

initLogging();
