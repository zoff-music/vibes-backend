import { useRouteLoaderData } from 'react-router';
import type { RootLoaderData } from '../root';
import { themes, useThemeStore } from '../stores/themeStore';
import { useHydrated } from './useHydrated';

export function useThemeDisplay() {
  const hydrated = useHydrated();
  const rootData = useRouteLoaderData('root') as RootLoaderData | undefined;
  const { themeId, currentTheme } = useThemeStore();

  if (!hydrated) {
    const ssrThemeId = rootData?.theme ?? 'auto';
    return {
      themeId: ssrThemeId,
      currentTheme: themes[ssrThemeId],
    };
  }

  return { themeId, currentTheme };
}
