'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Input } from '@/components/ui/Input'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Skeleton'
import { apiClient, getPatientName } from '@medilink/shared'
import Link from 'next/link'
import { cn } from '@/lib/utils'

const RESOURCE_TYPES = ['All', 'Patient', 'Observation', 'MedicationRequest', 'Condition', 'AllergyIntolerance', 'DiagnosticReport']

export default function SearchPage() {
  const [query, setQuery] = useState('')
  const [selectedType, setSelectedType] = useState('All')
  const [submittedQuery, setSubmittedQuery] = useState('')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['search', submittedQuery, selectedType],
    queryFn: async () => {
      const params: Record<string, string> = { q: submittedQuery }
      if (selectedType !== 'All') params.type = selectedType
      const res = await apiClient.get('/search', { params })
      return res.data
    },
    enabled: submittedQuery.length >= 2,
  })

  const entries = data?.entry || []

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.length >= 2) setSubmittedQuery(query)
  }

  return (
    <PageWrapper title="Search Records" subtitle="Search across all FHIR resources">
      <form onSubmit={handleSubmit} className="flex gap-2 mb-6">
        <div className="flex-1 max-w-lg">
          <Input
            placeholder="Search patients, conditions, medications..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            icon={<Search size={14} />}
          />
        </div>
        <Button type="submit" variant="secondary" size="sm" disabled={query.length < 2}>Search</Button>
      </form>

      <div className="flex gap-2 mb-6 overflow-x-auto pb-1">
        {RESOURCE_TYPES.map((type) => (
          <button
            key={type}
            onClick={() => setSelectedType(type)}
            className={cn(
              'px-3 py-1.5 text-xs font-medium rounded-full border whitespace-nowrap transition-colors',
              selectedType === type
                ? 'text-[var(--color-text-accent)] border-[var(--color-border)] bg-[var(--color-accent-subtle)]'
                : 'text-[var(--color-text-muted)] border-[var(--color-border-subtle)] hover:text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'
            )}
          >
            {type}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10" />)}
        </div>
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Search failed. Please try again later.
          </p>
        </div>
      ) : submittedQuery && entries.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            No results found for &ldquo;{submittedQuery}&rdquo;
          </p>
        </div>
      ) : (
        <div className="glass-card rounded-card border border-[var(--color-border)] shadow-card overflow-hidden divide-y divide-[var(--color-border-subtle)]">
          {entries.map((entry: { resource: Record<string, unknown> }, i: number) => {
            const resource = entry.resource
            const type = resource.resourceType as string
            return (
              <div
                key={`${type}-${resource.id ?? i}`}
                className="flex items-center justify-between px-4 py-3 transition-colors hover:bg-[var(--color-bg-hover)]"
              >
                <div className="flex items-center gap-3">
                  <Badge variant="muted" size="sm">{type}</Badge>
                  {type === 'Patient' ? (
                    <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                      {resource && typeof resource === 'object' && 'name' in resource
                        ? getPatientName(resource as unknown as Parameters<typeof getPatientName>[0])
                        : 'Unknown Patient'}
                    </span>
                  ) : (
                    <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {resource.id as string}
                    </span>
                  )}
                </div>
                {type === 'Patient' && (
                  <Link
                    href={`/patients/${resource.id}`}
                    className="text-xs font-medium transition-colors"
                    style={{ color: 'var(--color-text-accent)' }}
                  >
                    View
                  </Link>
                )}
              </div>
            )
          })}
        </div>
      )}
    </PageWrapper>
  )
}
