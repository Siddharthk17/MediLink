'use client'

import { useState, useDeferredValue } from 'react'
import { Search } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { cn } from '@/lib/utils'

interface PatientSearchProps {
  onSearch: (query: string) => void
  total?: number
  filters: string[]
  activeFilter: string
  onFilterChange: (filter: string) => void
}

export function PatientSearch({ onSearch, total, filters, activeFilter, onFilterChange }: PatientSearchProps) {
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query)

  const handleChange = (value: string) => {
    setQuery(value)
    onSearch(value)
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <div className="flex-1 max-w-sm">
          <Input
            placeholder="Search patients..."
            value={query}
            onChange={(e) => handleChange(e.target.value)}
            icon={<Search size={14} />}
          />
        </div>
        {total !== undefined && (
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {total} total
          </span>
        )}
      </div>
      <div className="flex gap-1.5">
        {filters.map((filter) => (
          <button
            key={filter}
            onClick={() => onFilterChange(filter)}
            className={cn(
              'px-3 py-1.5 text-xs font-medium rounded-full border transition-colors',
              activeFilter === filter
                ? 'text-[var(--color-text-accent)] border-[var(--color-border)] bg-[var(--color-accent-subtle)]'
                : 'text-[var(--color-text-muted)] border-[var(--color-border-subtle)] hover:text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'
            )}
          >
            {filter}
          </button>
        ))}
      </div>
    </div>
  )
}
