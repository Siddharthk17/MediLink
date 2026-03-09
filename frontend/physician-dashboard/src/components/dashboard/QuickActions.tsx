'use client'

import Link from 'next/link'

const actions = [
  { href: '/patients', label: 'Patients', key: 'P' },
  { href: '/search', label: 'Search records', key: 'S' },
  { href: '/consents', label: 'Consents', key: 'C' },
]

export function QuickActions() {
  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((action) => (
        <Link
          key={action.href}
          href={action.href}
          className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 h-10 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:border-[var(--color-border-hover)] hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <kbd>{action.key}</kbd>
          <span>{action.label}</span>
        </Link>
      ))}
    </div>
  )
}
