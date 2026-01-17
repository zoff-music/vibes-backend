export const darkTheme = {
  colors: {
    // Dark theme base
    background: '#0a0a0a',
    surface: '#141414',
    surfaceElevated: '#1c1c1c',

    // Accent (vibrant purple/pink gradient feel)
    primary: '#a855f7',
    primaryMuted: '#7c3aed',
    secondary: '#ec4899',

    // Text
    text: '#fafafa',
    textMuted: '#a1a1aa',
    textInverse: '#0a0a0a',

    // Semantic
    success: '#22c55e',
    warning: '#f59e0b',
    error: '#ef4444',

    // Overlay
    overlay: 'rgba(0, 0, 0, 0.7)',
  },

  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },

  radii: {
    sm: 4,
    md: 8,
    lg: 12,
    xl: 16,
    full: 9999,
  },

  fontSizes: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 24,
    xxl: 32,
    xxxl: 48,
  },
} as const;

export type AppTheme = typeof darkTheme;

// Breakpoints for responsive design
export const breakpoints = {
  xs: 0,
  sm: 576,
  md: 768,
  lg: 992,
  xl: 1200,
} as const;
