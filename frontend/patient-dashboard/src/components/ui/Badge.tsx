import { cn } from '@/lib/utils'

interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'success' | 'warning' | 'danger' | 'info' | 'accent'
  className?: string
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  const variants = {
    default: 'bg-[var(--color-bg-elevated)] text-[var(--color-text-secondary)]',
    success: 'bg-[var(--color-success-subtle)] text-[var(--color-success)]',
    warning: 'bg-[var(--color-warning-subtle)] text-[var(--color-warning)]',
    danger: 'bg-[var(--color-danger-subtle)] text-[var(--color-danger)]',
    info: 'bg-[var(--color-info-subtle)] text-[var(--color-info)]',
    accent: 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)]',
  }

  return (
    <span className={cn(
      'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
      variants[variant],
      className
    )}>
      {children}
    </span>
  )
}
