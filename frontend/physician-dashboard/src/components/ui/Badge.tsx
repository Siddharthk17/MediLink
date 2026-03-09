import { cn } from '@/lib/utils'

interface BadgeProps {
  variant: 'success' | 'warning' | 'danger' | 'info' | 'muted' | 'accent'
  size?: 'sm' | 'md'
  dot?: boolean
  children: React.ReactNode
  className?: string
}

const variantStyles: Record<BadgeProps['variant'], string> = {
  success: 'bg-[var(--color-success-subtle)] text-[var(--color-success)]',
  warning: 'bg-[var(--color-warning-subtle)] text-[var(--color-warning)]',
  danger:  'bg-[var(--color-danger-subtle)] text-[var(--color-danger)]',
  info:    'bg-[var(--color-info-subtle)] text-[var(--color-info)]',
  muted:   'bg-[var(--color-bg-elevated)] text-[var(--color-text-muted)]',
  accent:  'bg-[var(--color-accent-subtle)] text-[var(--color-accent)]',
}

export function Badge({ variant, size = 'sm', dot, children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 border-transparent font-medium rounded-full',
        variantStyles[variant],
        {
          'px-1.5 py-0.5 text-[10px]': size === 'sm',
          'px-2 py-0.5 text-[11px]': size === 'md',
        },
        className
      )}
    >
      {dot && (
        <span
          className="h-1.5 w-1.5 rounded-full pulse-soft"
          style={{ backgroundColor: 'currentColor' }}
        />
      )}
      {children}
    </span>
  )
}
