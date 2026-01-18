/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Retro Japanese Minimalist with Synthwave
        background: '#fefefe',
        paper: '#ffffff',
        surface: '#f8f8fb',
        surfaceElevated: '#ffffff',
        surfaceHover: '#f0f0f8',
        border: '#e5e5f0',

        // Dark mode palette - sleek dark grey with pink accent
        dark: {
          background: '#0d0d0f', // Near-black with slight cool tint
          paper: '#141416',      // Dark charcoal
          surface: '#1a1a1e',    // Surface grey
          surfaceElevated: '#222226', // Elevated surface
          surfaceHover: '#2a2a30',    // Hover state
          border: '#2a2a30',          // Border color
          text: {
            DEFAULT: '#f5f5f7',       // Clean white
            muted: '#a1a1a8',         // Muted grey
            subtle: '#6b6b73',        // Subtle grey
            inverse: '#0d0d0f',       // Inverse text
          },
        },

        // Synthwave Palette
        primary: {
          DEFAULT: '#ff2e97', // Hot pink
          muted: '#ff1493',
          light: '#ffb3e6',
        },
        secondary: '#00d9ff', // Cyan
        accent: '#ffd700', // Gold
        purple: '#b24bf3', // Vibrant purple

        // Japanese aesthetic colors
        sakura: '#ffb7c5',
        matcha: '#a8d8b9',
        ink: '#2d3142',

        text: {
          DEFAULT: '#2d3142',
          muted: '#6b7280',
          subtle: '#9ca3af',
          inverse: '#ffffff',
        },
        success: '#00d9a3',
        warning: '#ffd93d',
        error: '#ff2e63',
      },
      fontFamily: {
        body: ['Inter', 'Noto Sans JP', 'system-ui', 'sans-serif'],
        heading: ['Poppins', 'Noto Sans JP', 'system-ui', 'sans-serif'],
        mono: ['"Courier New"', 'MS PGothic', 'monospace'],
        japanese: ['"Noto Sans JP"', '"Hiragino Sans"', 'sans-serif'],
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
        'retro': '4px 4px 0px 0px rgba(0, 0, 0, 0.15)',
        'retro-lg': '6px 6px 0px 0px rgba(0, 0, 0, 0.15)',
        'retro-pink': '4px 4px 0px 0px rgba(255, 46, 151, 0.3)',
        'retro-cyan': '4px 4px 0px 0px rgba(0, 217, 255, 0.3)',
        'neon-pink': '0 0 20px rgba(255, 46, 151, 0.5)',
        'neon-cyan': '0 0 20px rgba(0, 217, 255, 0.5)',
        'soft': '0 2px 8px rgba(45, 49, 66, 0.08)',
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
        'float': 'float 3s ease-in-out infinite',
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
      },
    },
  },
  plugins: [],
}
