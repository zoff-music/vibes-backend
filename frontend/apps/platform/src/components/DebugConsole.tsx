import React, { useEffect, useRef, useState } from 'react';

type LogLevel = 'log' | 'error' | 'warn' | 'info' | 'debug';

interface LogMessage {
  id: string;
  content: any;
}

interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  messages: LogMessage[];
}

const generateId = () => Math.random().toString(36).substring(2, 9);

const DebugConsole: React.FC = () => {
  const [isVisible, setIsVisible] = useState(false);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Check for debug flag in URL
    const searchParams = new URLSearchParams(window.location.search);
    const debug = searchParams.get('debug');
    if (debug !== 'true') return;

    setIsVisible(true);

    // Patch console methods
    const originalConsole = {
      log: console.log,
      error: console.error,
      warn: console.warn,
      info: console.info,
      debug: console.debug,
    };

    const addLog = (level: LogLevel, args: any[]) => {
      const entry: LogEntry = {
        id: generateId(),
        timestamp: new Date().toLocaleTimeString(),
        level,
        messages: args.map((arg) => ({
          id: generateId(),
          content: arg,
        })),
      };
      setLogs((prev) => [...prev, entry]); // Keep all logs, maybe limit later
    };

    console.log = (...args) => {
      originalConsole.log(...args);
      addLog('log', args);
    };

    console.error = (...args) => {
      originalConsole.error(...args);
      addLog('error', args);
    };

    console.warn = (...args) => {
      originalConsole.warn(...args);
      addLog('warn', args);
    };

    console.info = (...args) => {
      originalConsole.info(...args);
      addLog('info', args);
    };

    console.debug = (...args) => {
      originalConsole.debug(...args);
      addLog('debug', args);
    };

    // Cleanup
    return () => {
      console.log = originalConsole.log;
      console.error = originalConsole.error;
      console.warn = originalConsole.warn;
      console.info = originalConsole.info;
      console.debug = originalConsole.debug;
    };
  }, []);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  if (!isVisible) return null;

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        width: '100%',
        height: '50vh',
        backgroundColor: 'rgba(0, 0, 0, 0.9)',
        color: '#fff',
        zIndex: 9999,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'monospace',
        fontSize: '12px',
        borderTop: '2px solid #333',
        pointerEvents: 'none', // Allow clicks to pass through by default
      }}
    >
      <div
        style={{
          padding: '4px',
          backgroundColor: '#333',
          display: 'flex',
          justifyContent: 'space-between',
          pointerEvents: 'auto', // Re-enable pointer events for the header
          userSelect: 'none',
        }}
      >
        <strong>Debug Console</strong>
        <button
          onClick={() => setLogs([])}
          style={{
            background: 'transparent',
            border: '1px solid #666',
            color: '#fff',
            cursor: 'pointer',
            padding: '2px 8px',
          }}
        >
          Clear
        </button>
      </div>
      <div
        ref={scrollRef}
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '8px',
          pointerEvents: 'auto', // Re-enable pointer events for the log area
        }}
      >
        {logs.map((log) => (
          <div
            key={log.id}
            style={{
              marginBottom: '4px',
              borderBottom: '1px solid #222',
              paddingBottom: '2px',
              color:
                log.level === 'error'
                  ? '#ff5555'
                  : log.level === 'warn'
                    ? '#ffcc00'
                    : '#cccccc',
            }}
          >
            <span style={{ color: '#666', marginRight: '8px' }}>
              [{log.timestamp}]
            </span>
            <span
              style={{
                fontWeight: 'bold',
                marginRight: '8px',
                textTransform: 'uppercase',
              }}
            >
              {log.level}
            </span>
            <span>
              {log.messages.map((msg) => (
                <span key={msg.id} style={{ marginRight: '4px' }}>
                  {typeof msg.content === 'object'
                    ? JSON.stringify(msg.content, null, 2)
                    : String(msg.content)}
                </span>
              ))}
            </span>
          </div>
        ))}
        {logs.length === 0 && (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            Waiting for logs...
          </div>
        )}
      </div>
    </div>
  );
};

export default DebugConsole;
