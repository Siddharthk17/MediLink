'use client'

import { useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface DrawerProps {
  open: boolean
  onClose: () => void
  children: React.ReactNode
  title?: string
  width?: number
}

export function Drawer({ open, onClose, children, title, width = 380 }: DrawerProps) {
  useEffect(() => {
    if (!open) return
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [open, onClose])

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-50">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-[var(--color-bg-overlay)]"
            onClick={onClose}
          />
          <motion.aside
            initial={{ x: width }}
            animate={{ x: 0 }}
            exit={{ x: width }}
            transition={{ type: 'spring', stiffness: 300, damping: 30 }}
            style={{ width }}
            className="absolute right-3 top-3 bottom-3 h-auto glass border border-[var(--color-border)] rounded-[24px] shadow-elevated overflow-y-auto"
            role="dialog"
            aria-modal="true"
            aria-label={title}
          >
            {title && (
              <div className="flex items-center justify-between p-4 border-b border-[var(--color-border-subtle)]">
                <h2 className="text-sm font-medium text-[var(--color-text-primary)]">
                  {title}
                </h2>
                <button
                  onClick={onClose}
                  className="p-1 rounded-button hover:bg-[var(--color-bg-elevated)] transition-colors"
                  aria-label="Close"
                >
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M18 6L6 18M6 6l12 12" />
                  </svg>
                </button>
              </div>
            )}
            <div className="p-4">{children}</div>
          </motion.aside>
        </div>
      )}
    </AnimatePresence>
  )
}
