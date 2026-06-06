/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{js,ts,jsx,tsx}',
    '../../packages/ui/src/**/*.{js,ts,jsx,tsx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        background: '#fefefe',
        paper: '#ffffff',
        surface: '#f8f8fb',
        surfaceElevated: '#ffffff',
        surfaceHover: '#f0f0f8',
        border: '#e5e5f0',

        'dark-background': '#0d0d0f',
        'dark-paper': '#141416',
        'dark-surface': '#1a1a1e',
        'dark-surfaceElevated': '#222226',
        'dark-surfaceHover': '#2a2a30',
        'dark-border': '#2a2a30',

        'dark-text': '#f5f5f7',
        'dark-text-muted': '#a1a1a8',
        'dark-text-subtle': '#6b6b73',
        'dark-text-inverse': '#0d0d0f',

        primary: '#ff2e97',
        'primary-muted': '#ff1493',
        'primary-light': '#ffb3e6',

        secondary: '#00d9ff',
        accent: '#ffd700',
        purple: '#b24bf3',
        sakura: '#ffb7c5',
        matcha: '#a8d8b9',
        ink: '#2d3142',

        text: '#2d3142',
        'text-muted': '#6b7280',
        'text-subtle': '#9ca3af',
        'text-inverse': '#ffffff',

        success: '#00d9a3',
        warning: '#ffd93d',
        error: '#ff2e63',
      },
      fontFamily: {
        body: [
          '"Pixelify Sans"',
          '"Zen Maru Gothic"',
          'system-ui',
          'sans-serif',
        ],
        heading: ['"Pixelify Sans"', 'sans-serif'],
        mono: ['"Pixelify Sans"', '"Courier New"', 'monospace'],
        pixel: ['"Pixelify Sans"', 'cursive'],
        japanese: ['"Zen Maru Gothic"', '"Hiragino Sans"', 'sans-serif'],
      },
      fontSize: {
        xs: ['12px', { lineHeight: '16px', letterSpacing: '0.02em' }],
        sm: ['14px', { lineHeight: '20px', letterSpacing: '0.01em' }],
        base: ['16px', { lineHeight: '24px', letterSpacing: '0' }],
        md: ['16px', { lineHeight: '24px', letterSpacing: '0' }],
        lg: ['18px', { lineHeight: '28px', letterSpacing: '-0.01em' }],
        xl: ['22px', { lineHeight: '32px', letterSpacing: '-0.02em' }],
        '2xl': ['28px', { lineHeight: '36px', letterSpacing: '-0.02em' }],
        '3xl': ['36px', { lineHeight: '44px', letterSpacing: '-0.03em' }],
        '4xl': ['48px', { lineHeight: '56px', letterSpacing: '-0.03em' }],
      },
      borderRadius: {
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px',
        '2xl': '20px',
        full: '9999px',
      },
      boxShadow: {
        retro: '4px 4px 0px 0px rgba(0, 0, 0, 0.15)',
        'retro-lg': '6px 6px 0px 0px rgba(0, 0, 0, 0.15)',
        'retro-pink': '4px 4px 0px 0px rgba(255, 46, 151, 0.3)',
        'retro-cyan': '4px 4px 0px 0px rgba(0, 217, 255, 0.3)',
        'neon-pink': '0 0 20px rgba(255, 46, 151, 0.5)',
        'neon-cyan': '0 0 20px rgba(0, 217, 255, 0.5)',
        soft: '0 2px 8px rgba(45, 49, 66, 0.08)',
      },
      backdropBlur: {
        xs: '2px',
        sm: '4px',
        DEFAULT: '8px',
        md: '12px',
        lg: '16px',
      },
      animation: {
        'fade-in': 'fadeIn 0.4s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'neon-pulse': 'neonPulse 2s ease-in-out infinite',
        float: 'float 3s ease-in-out infinite',
        scanlines: 'scanlines 1.8s linear infinite',
        'glow-pulse': 'glowPulse 2.6s ease-in-out infinite',
        'gradient-shift': 'gradientShift 10s ease-in-out infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        slideDown: {
          '0%': { transform: 'translateY(-10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        scaleIn: {
          '0%': { transform: 'scale(0.98)', opacity: '0' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        neonPulse: {
          '0%, 100%': { opacity: '1', filter: 'brightness(1)' },
          '50%': { opacity: '0.8', filter: 'brightness(1.2)' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-10px)' },
        },
        scanlines: {
          '0%': { backgroundPosition: '0 0' },
          '100%': { backgroundPosition: '0 12px' },
        },
        glowPulse: {
          '0%, 100%': {
            filter: 'drop-shadow(0 0 12px rgba(255, 63, 164, 0.6))',
          },
          '50%': { filter: 'drop-shadow(0 0 18px rgba(0, 217, 255, 0.5))' },
        },
        gradientShift: {
          '0%': { backgroundPosition: '0% 50%' },
          '50%': { backgroundPosition: '100% 50%' },
          '100%': { backgroundPosition: '0% 50%' },
        },
      },
    },
  },
  plugins: [
    ({ addVariant }) => {
      addVariant('dark', [
        '.dark &',
        '@media (prefers-color-scheme: dark) { html:not(.theme-light) & }',
      ]);
    },
  ],
};
