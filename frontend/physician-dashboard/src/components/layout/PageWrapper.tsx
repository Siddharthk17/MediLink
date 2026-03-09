'use client'

import { motion } from 'framer-motion'
import { pageVariants } from '@/lib/motion'

interface PageWrapperProps {
  children: React.ReactNode
  title?: string
  subtitle?: string
  actions?: React.ReactNode
}

export function PageWrapper({ children, title, subtitle, actions }: PageWrapperProps) {
  return (
    <motion.div
      variants={pageVariants}
      initial="initial"
      animate="animate"
      exit="exit"
      className="max-w-7xl mx-auto"
    >
      {(title || actions) && (
        <div className="mb-8 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            {title && (
              <h1 className="font-display text-4xl md:text-5xl tracking-tight text-gradient">
                {title}
              </h1>
            )}
            {subtitle && (
              <p className="mt-2 text-base text-[var(--color-text-muted)]">
                {subtitle}
              </p>
            )}
          </div>
          {actions && <div className="flex items-center gap-3">{actions}</div>}
        </div>
      )}
      {children}
    </motion.div>
  )
}
