export const SEVERITY_COLORS = {
  contraindicated: { bg: 'var(--color-danger-subtle)', text: '#F43F5E', border: 'rgba(244, 63, 94, 0.3)' },
  major:           { bg: 'var(--color-info-subtle)',   text: '#8B5CF6', border: 'rgba(139, 92, 246, 0.3)' },
  moderate:        { bg: 'var(--color-warning-subtle)',text: '#F59E0B', border: 'rgba(245, 158, 11, 0.3)' },
  minor:           { bg: 'var(--color-success-subtle)',text: '#10B981', border: 'rgba(16, 185, 129, 0.3)' },
  unknown:         { bg: 'rgba(113, 113, 122, 0.08)', text: '#71717A', border: 'rgba(113, 113, 122, 0.2)' },
  none:            { bg: 'var(--color-success-subtle)',text: '#10B981', border: 'rgba(16, 185, 129, 0.3)' },
} as const
