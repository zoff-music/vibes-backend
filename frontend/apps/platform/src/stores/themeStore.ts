import { safeWrap } from '@vibez/shared';
import { create } from 'zustand';

// Theme definitions - easy to add new themes by adding to this object
export const themes = {
  light: {
    id: 'light',
    name: 'Light',
    class: 'theme-light',
  },
  dark: {
    id: 'dark',
    name: 'Dark',
    class: 'dark',
  },
  auto: {
    id: 'auto',
    name: 'Auto',
    class: '', // Auto is the baseline (no class)
  },
  // Add new themes here:
  // midnight: {
  //     id: 'midnight',
  //     name: 'Midnight Blue',
  //     icon: '🌌',
  //     class: 'midnight',
  // },
  // sunset: {
  //     id: 'sunset',
  //     name: 'Sunset',
  //     icon: '🌅',
  //     class: 'sunset',
  // },
} as const;

export type ThemeId = keyof typeof themes;
export type Theme = (typeof themes)[ThemeId];

interface Preferences {
  theme: ThemeId;
  version: number;
}

interface ThemeState {
  themeId: ThemeId;
  resolvedTheme: 'light' | 'dark';
  isDarkMode: boolean;
  isWarping: boolean;
  currentTheme: Theme;
  setTheme: (themeId: ThemeId) => void;
  toggleDarkMode: () => void; // Convenience method for simple light/dark toggle
  setIsWarping: (warp: boolean) => void;
}

// Cookie utilities
const COOKIE_NAME = 'preferences';
const CURRENT_VERSION = 1;

function setCookie(name: string, value: string, days: number = 365) {
  if (typeof document === 'undefined') return;

  const expires = new Date();
  expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000);

  const cookieString = `${name}=${value};expires=${expires.toUTCString()};path=/;SameSite=Lax`;

  // Use a function to set the cookie to avoid direct assignment warning
  function setCookieValue(value: string) {
    // biome-ignore lint/suspicious/noDocumentCookie: Cookie setting is necessary for theme persistence
    document.cookie = value;
  }

  setCookieValue(cookieString);
}

function savePreferences(preferences: Preferences) {
  const encoded = btoa(JSON.stringify(preferences));
  setCookie(COOKIE_NAME, encoded);
}

// Apply theme class to document
function applyTheme(themeId: ThemeId) {
  if (typeof document === 'undefined') return;
  const html = document.documentElement;

  // Remove all theme classes
  Object.values(themes).forEach((theme) => {
    if (theme.class) {
      html.classList.remove(theme.class);
    }
  });

  // Add new theme class
  const newTheme = themes[themeId];
  if (newTheme.class) {
    html.classList.add(newTheme.class);
  }
}

// Initialize theme by checking the initial data provided by SSR, then DOM state
function getInitialThemeSync(): ThemeId {
  if (typeof document === 'undefined') {
    return 'auto'; // Server-side fallback
  }

  // 1. Try to get theme from SSR data (most reliable source of truth)
  const dataElement = document.getElementById('ssr-data');
  if (dataElement) {
    const [err, data] = safeWrap(() =>
      JSON.parse(dataElement.textContent || '{}'),
    );

    if (!err && data && data.theme) {
      // Validate that the theme is valid
      if (
        data.theme === 'dark' ||
        data.theme === 'light' ||
        data.theme === 'auto'
      ) {
        console.log('[Theme] Initialized from SSR data:', data.theme);
        return data.theme as ThemeId;
      }
    }
  }

  // 2. Fallback: Check if the HTML element has any theme class
  const classList = document.documentElement.classList;
  if (classList.contains('dark')) return 'dark';
  if (classList.contains('theme-light')) return 'light';

  return 'auto';
}

function resolveTheme(themeId: ThemeId): 'light' | 'dark' {
  if (themeId !== 'auto') return themeId as 'light' | 'dark';
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light';
}

// Initialize theme immediately - before any React rendering
const INITIAL_THEME = getInitialThemeSync();

// No need to apply theme here since SSR already set it correctly
console.log('[Theme] Initial theme detected from DOM:', INITIAL_THEME);

export const useThemeStore = create<ThemeState>((set, get) => {
  console.log('[Theme] Store initialized with theme:', INITIAL_THEME);

  // Set up listener for system theme changes if in auto mode
  if (typeof window !== 'undefined') {
    window
      .matchMedia('(prefers-color-scheme: dark)')
      .addEventListener('change', (e) => {
        if (get().themeId === 'auto') {
          const newResolved = e.matches ? 'dark' : 'light';
          set({
            resolvedTheme: newResolved,
            isDarkMode: newResolved === 'dark',
          });
        }
      });
  }

  const initialResolved = resolveTheme(INITIAL_THEME);

  return {
    themeId: INITIAL_THEME,
    resolvedTheme: initialResolved,
    isDarkMode: initialResolved === 'dark',
    isWarping: false,
    currentTheme: themes[INITIAL_THEME],

    setTheme: (themeId: ThemeId) => {
      console.log('[Theme] Setting theme to:', themeId);
      applyTheme(themeId);
      savePreferences({ theme: themeId, version: CURRENT_VERSION });

      const resolved = resolveTheme(themeId);
      set({
        themeId,
        resolvedTheme: resolved,
        isDarkMode: resolved === 'dark',
        currentTheme: themes[themeId],
      });
    },

    toggleDarkMode: () => {
      const current = get().themeId;
      // Toggle cycles: light -> dark -> auto -> light
      let newTheme: ThemeId;
      if (current === 'light') newTheme = 'dark';
      else if (current === 'dark') newTheme = 'auto';
      else newTheme = 'light';

      console.log('[Theme] Toggling from', current, 'to', newTheme);
      applyTheme(newTheme);
      savePreferences({ theme: newTheme, version: CURRENT_VERSION });

      const resolved = resolveTheme(newTheme);
      set({
        themeId: newTheme,
        resolvedTheme: resolved,
        isDarkMode: resolved === 'dark',
        currentTheme: themes[newTheme],
      });
    },

    setIsWarping: (warp: boolean) => {
      set({ isWarping: warp });
    },
  };
});

// Export helper to get all available themes (for theme selector UI)
export const getAvailableThemes = () => Object.values(themes);
