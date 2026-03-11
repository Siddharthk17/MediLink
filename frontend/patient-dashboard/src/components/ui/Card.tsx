'use client'

import { cn } from '@/lib/utils'
import { forwardRef } from 'react'

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'glass' | 'elevated'
  padding?: 'sm' | 'md' | 'lg'
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, variant = 'default', padding = 'md', ...props }, ref) => {
    const variants = {
      default: 'bg-[var(--color-bg-card)] border border-[var(--color-border)] shadow-card',
      glass: 'glass-card border border-[var(--color-border)]',
      elevated: 'bg-[var(--color-bg-elevated)] border border-[var(--color-border)] shadow-elevated',
    }
    const paddings = {
      sm: 'p-4',
      md: 'p-5',
      lg: 'p-7',
    }
    return (
      <div
        ref={ref}
        className={cn('rounded-card', paddings[padding], variants[variant], className)}
        {...props}
      />
    )
  }
)
Card.displayName = 'Card'
