import { isTruthyFlag, safeWrapAsync } from '@vibez/shared';
import { type ComponentType, useEffect, useState } from 'react';
import { Outlet, useLocation } from 'react-router';

import { Background } from './components/layout/Background';
import { updateNavigationHistory } from './utils/navigationHistory';

const debugEnabled = isTruthyFlag(import.meta.env.VITE_DEBUG);

type DebugConsoleComponent = ComponentType;

function DebugConsoleLoader() {
  const [DebugConsole, setDebugConsole] =
    useState<DebugConsoleComponent | null>(null);

  useEffect(() => {
    if (!debugEnabled) return;

    let isMounted = true;

    const loadDebugConsole = async () => {
      const [loadErr, module] = await safeWrapAsync(import('@vibez/ui'));
      if (!isMounted || loadErr || !module?.DebugConsole) {
        if (loadErr) {
          console.error('[DebugConsole] Failed to load', loadErr);
        }
        return;
      }
      setDebugConsole(() => module.DebugConsole);
    };

    loadDebugConsole();

    return () => {
      isMounted = false;
    };
  }, []);

  if (!debugEnabled || !DebugConsole) return null;

  return <DebugConsole />;
}

export default function App() {
  const location = useLocation();

  useEffect(() => {
    updateNavigationHistory(location.pathname);
  }, [location.pathname]);

  return (
    <>
      <DebugConsoleLoader />
      <Background />
      <Outlet />
    </>
  );
}
