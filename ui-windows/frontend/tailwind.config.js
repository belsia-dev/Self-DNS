/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        ink: {
          0: '#000000',
          50: '#0a0a0a',
          100: '#0f0f10',
          200: '#141416',
          300: '#1a1a1d',
          400: '#222226',
          500: '#2a2a2f',
          600: '#3a3a40',
        },
        text: {
          primary: '#e8e8ea',
          secondary: '#a3a3a8',
          muted: '#6e6e74',
          dim: '#4a4a50',
        },
        line: {
          DEFAULT: '#1f1f23',
          strong: '#2a2a2f',
        },
        accent: {
          DEFAULT: '#a3e635',
          dim: '#84cc16',
          deep: '#65a30d',
          glow: 'rgba(163, 230, 53, 0.18)',
        },
        good: '#86efac',
        warn: '#fbbf24',
        bad: '#f87171',
        info: '#7dd3fc',
        violet: '#c4b5fd',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        display: ['"Inter Tight"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', '"SF Mono"', 'Menlo', 'monospace'],
      },
      fontSize: {
        '2xs': ['10px', { lineHeight: '14px', letterSpacing: '0.04em' }],
        'xs': ['11px', { lineHeight: '16px' }],
        'sm': ['12.5px', { lineHeight: '18px' }],
        'base': ['13px', { lineHeight: '20px' }],
      },
      letterSpacing: {
        widest: '0.18em',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: 0, transform: 'translateY(2px)' },
          '100%': { opacity: 1, transform: 'translateY(0)' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0.55 },
        },
        'breath': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(163, 230, 53, 0.35)' },
          '50%': { boxShadow: '0 0 0 4px rgba(163, 230, 53, 0)' },
        },
      },
      animation: {
        'fade-in': 'fade-in 0.25s ease-out',
        'pulse-soft': 'pulse-soft 2.4s ease-in-out infinite',
        'breath': 'breath 2s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}
