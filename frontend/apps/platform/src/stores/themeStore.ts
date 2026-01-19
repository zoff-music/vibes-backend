import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// Theme definitions - easy to add new themes by adding to this object
export const themes = {
  light: {
    id: 'light',
    name: 'Light',
    icon: '☀️',
    class: '', // No class needed for light (default)
  },
  dark: {
    id: 'dark',
    name: 'Dark',
    icon: '🌙',
    class: 'dark',
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

interface ThemeState {
  themeId: ThemeId;
  setTheme: (themeId: ThemeId) => void;
  toggleDarkMode: () => void; // Convenience method for simple light/dark toggle
  // Computed helpers
  isDarkMode: boolean;
  currentTheme: Theme;
}

// Apply theme class to document
const applyTheme = (themeId: ThemeId) => {
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
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      themeId: 'light',

      setTheme: (themeId: ThemeId) => {
        applyTheme(themeId);
        set({ themeId });
      },

      toggleDarkMode: () => {
        const current = get().themeId;
        const newTheme = current === 'dark' ? 'light' : 'dark';
        applyTheme(newTheme);
        set({ themeId: newTheme });
      },

      // Computed getters
      get isDarkMode() {
        return get().themeId === 'dark';
      },

      get currentTheme() {
        return themes[get().themeId];
      },
    }),
    {
      name: 'vibes-theme',
      onRehydrateStorage: () => (state) => {
        // Apply saved theme on page load
        if (state?.themeId) {
          applyTheme(state.themeId);
        }
      },
    },
  ),
);

// Export helper to get all available themes (for theme selector UI)
export const getAvailableThemes = () => Object.values(themes);
