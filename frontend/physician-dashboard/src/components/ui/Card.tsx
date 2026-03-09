import { cn } from '@/lib/utils'

interface CardProps {
  children: React.ReactNode
  className?: string
  hover?: boolean
  padding?: 'sm' | 'md' | 'lg'
}

export function Card({ children, className, hover, padding = 'md' }: CardProps) {
  return (
    <div
      className={cn(
        'glass-card border border-[var(--color-border)] rounded-card shadow-card',
        hover && 'transition-all duration-200 hover:border-[var(--color-border-hover)] hover:shadow-elevated hover:-translate-y-0.5 cursor-pointer',
        {
          'p-4': padding === 'sm',
          'p-5': padding === 'md',
          'p-7': padding === 'lg',
        },
        className
      )}
    >
      {children}
    </div>
  )
}
