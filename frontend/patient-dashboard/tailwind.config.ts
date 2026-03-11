import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        'bg-base':     'var(--color-bg-base)',
        'bg-surface':  'var(--color-bg-surface)',
        'bg-card':     'var(--color-bg-card)',
        'bg-elevated': 'var(--color-bg-elevated)',
        'bg-hover':    'var(--color-bg-hover)',
        'text-secondary': 'var(--color-text-secondary)',
        'text-muted':     'var(--color-text-muted)',
        'text-accent':    'var(--color-text-accent)',
        accent:     'var(--color-accent)',
        success:    'var(--color-success)',
        warning:    'var(--color-warning)',
        danger:     'var(--color-danger)',
        info:       'var(--color-info)',
        border:     'var(--color-border)',
      },
      fontFamily: {
        display: ['var(--font-display)'],
        body:    ['var(--font-body)'],
        mono:    ['var(--font-mono)'],
      },
      borderRadius: {
        card: 'var(--radius-card)',
        button: 'var(--radius)',
      },
      boxShadow: {
        xs:       'var(--shadow-xs)',
        card:     'var(--shadow-card)',
        elevated: 'var(--shadow-elevated)',
        glow:     'var(--shadow-glow-accent)',
      },
      keyframes: {
        shimmer: {
          '0%':   { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'fade-in': {
          '0%':   { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-in-right': {
          '0%':   { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: '1' },
          '50%':      { opacity: '0.4' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-6px)' },
        },
      },
      animation: {
        shimmer:        'shimmer 2s ease-in-out infinite',
        'fade-in':      'fade-in 0.2s ease-out',
        'slide-in':     'slide-in-right 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
        'pulse-soft':   'pulse-soft 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        float:          'float 3s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}

export default config
