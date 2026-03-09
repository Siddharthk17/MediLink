'use client'

import { useState } from 'react'
import { cn } from '@/lib/utils'

interface TooltipProps {
  content: string
  children: React.ReactNode
  side?: 'top' | 'right' | 'bottom' | 'left'
}

export function Tooltip({ content, children, side = 'right' }: TooltipProps) {
  const [visible, setVisible] = useState(false)

  return (
    <div className="relative inline-flex" onMouseEnter={() => setVisible(true)} onMouseLeave={() => setVisible(false)}>
      {children}
      {visible && (
        <div
          className={cn(
            'absolute z-50 px-2.5 py-1.5 text-xs font-medium rounded-[var(--radius-sm)] whitespace-nowrap pointer-events-none',
            'bg-[var(--color-bg-elevated)] text-[var(--color-text-primary)] border border-[var(--color-border)] shadow-elevated',
            {
              'bottom-full left-1/2 -translate-x-1/2 mb-2': side === 'top',
              'left-full top-1/2 -translate-y-1/2 ml-2': side === 'right',
              'top-full left-1/2 -translate-x-1/2 mt-2': side === 'bottom',
              'right-full top-1/2 -translate-y-1/2 mr-2': side === 'left',
            }
          )}
        >
          {content}
        </div>
      )}
    </div>
  )
}
