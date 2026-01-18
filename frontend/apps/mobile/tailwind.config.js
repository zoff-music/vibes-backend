/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#0f0f12',
        surface: 'rgba(255, 255, 255, 0.03)',
        surfaceElevated: 'rgba(255, 255, 255, 0.06)',
        surfaceHover: 'rgba(255, 255, 255, 0.08)',
        border: 'rgba(255, 255, 255, 0.08)',
        primary: {
          DEFAULT: '#6366f1',
          muted: '#4f46e5',
          glow: 'rgba(99, 102, 241, 0.3)',
        },
        secondary: '#8b5cf6',
        accent: '#14b8a6',
        text: {
          DEFAULT: '#ffffff',
          muted: 'rgba(255, 255, 255, 0.6)',
          subtle: 'rgba(255, 255, 255, 0.4)',
          inverse: '#0f0f12',
        },
        success: '#10b981',
        warning: '#f59e0b',
        error: '#ef4444',
      },
      fontFamily: {
        body: ['"DM Sans"', 'system-ui', '-apple-system', 'sans-serif'],
        heading: ['"DM Sans"', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
      fontSize: {
        xs: ['11px', { lineHeight: '16px', letterSpacing: '0.02em' }],
        sm: ['13px', { lineHeight: '18px', letterSpacing: '0.01em' }],
        base: ['15px', { lineHeight: '22px', letterSpacing: '0' }],
        md: ['15px', { lineHeight: '22px', letterSpacing: '0' }],
        lg: ['17px', { lineHeight: '24px', letterSpacing: '-0.01em' }],
        xl: ['20px', { lineHeight: '28px', letterSpacing: '-0.02em' }],
        '2xl': ['24px', { lineHeight: '32px', letterSpacing: '-0.02em' }],
        '3xl': ['32px', { lineHeight: '40px', letterSpacing: '-0.03em' }],
        '4xl': ['40px', { lineHeight: '48px', letterSpacing: '-0.03em' }],
      },
      borderRadius: {
        sm: '6px',
        md: '12px',
        lg: '16px',
        xl: '20px',
        '2xl': '24px',
        full: '9999px',
      },
      boxShadow: {
        'glass': '0 8px 32px 0 rgba(0, 0, 0, 0.37)',
        'glass-lg': '0 12px 48px 0 rgba(0, 0, 0, 0.5)',
        'glow': '0 0 24px rgba(99, 102, 241, 0.3)',
        'glow-lg': '0 0 48px rgba(99, 102, 241, 0.4)',
      },
      backdropBlur: {
        xs: '2px',
        sm: '4px',
        DEFAULT: '12px',
        md: '16px',
        lg: '24px',
        xl: '32px',
      },
      animation: {
        'fade-in': 'fadeIn 0.5s ease-out',
        'slide-up': 'slideUp 0.4s ease-out',
        'slide-down': 'slideDown 0.4s ease-out',
        'scale-in': 'scaleIn 0.3s ease-out',
        'glow-pulse': 'glowPulse 2s ease-in-out infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(20px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        slideDown: {
          '0%': { transform: 'translateY(-20px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        scaleIn: {
          '0%': { transform: 'scale(0.95)', opacity: '0' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        glowPulse: {
          '0%, 100%': { opacity: '0.5' },
          '50%': { opacity: '1' },
        },
      },
    },
  },
  plugins: [],
}
