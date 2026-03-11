'use client'

import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Search, Activity, Pill, Stethoscope, Syringe,
  AlertTriangle, FileText, Calendar
} from 'lucide-react'
import { searchAPI, getCodeDisplay, formatDate } from '@medilink/shared'
import type { FHIRBundle, CodeableConcept } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Input } from '@/components/ui/Input'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

const typeIcons: Record<string, React.ElementType> = {
  Observation: Activity,
  MedicationRequest: Pill,
  Condition: Stethoscope,
  Immunization: Syringe,
  AllergyIntolerance: AlertTriangle,
  DiagnosticReport: FileText,
  Encounter: Calendar,
}

export default function SearchPage() {
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  const handleSearch = (value: string) => {
    setQuery(value)
  }

  useEffect(() => {
    const timeout = setTimeout(() => setDebouncedQuery(query), 400)
    return () => clearTimeout(timeout)
  }, [query])

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['search', debouncedQuery],
    queryFn: () => searchAPI.unifiedSearch(debouncedQuery),
    enabled: debouncedQuery.length >= 2,
  })

  const results = data?.data?.entry ?? []

  return (
    <PageWrapper title="Search" subtitle="Find anything in your health records">
      <div className="max-w-2xl mx-auto mb-8">
        <Input
          placeholder="Search conditions, medications, lab results…"
          value={query}
          onChange={(e) => handleSearch(e.target.value)}
          icon={<Search className="w-4 h-4" />}
          className="h-14 text-base rounded-2xl"
        />
        {query.length > 0 && query.length < 2 && (
          <p className="text-xs text-[var(--color-text-muted)] mt-2 text-center">
            Type at least 2 characters to search
          </p>
        )}
      </div>

      {isLoading || isFetching ? (
        <div className="space-y-3 max-w-2xl mx-auto">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-card" />
          ))}
        </div>
      ) : debouncedQuery.length >= 2 && results.length === 0 ? (
        <Card variant="glass" className="text-center py-12 max-w-2xl mx-auto">
          <Search className="w-8 h-8 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No results for &ldquo;{debouncedQuery}&rdquo;</p>
        </Card>
      ) : results.length > 0 ? (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-3 max-w-2xl mx-auto">
          {results.map((result, i: number) => {
            const resource = result.resource as unknown as Record<string, unknown>
            const type = (resource.resourceType as string) || 'Unknown'
            const Icon = typeIcons[type] || FileText
            const display = getCodeDisplay(resource.code as CodeableConcept)
              || (resource.vaccineCode as CodeableConcept)?.text
              || (resource.medicationCodeableConcept as CodeableConcept)?.text
              || type
            const date = (resource.effectiveDateTime || resource.authoredOn || resource.recordedDate) as string | undefined

            return (
              <motion.div key={(resource.id as string) ?? `sr-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors cursor-pointer">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style={{
                      backgroundColor: `var(--color-type-${type.toLowerCase()}-subtle, var(--color-bg-elevated))`,
                      color: `var(--color-type-${type.toLowerCase()}, var(--color-text-muted))`
                    }}>
                      <Icon className="w-4 h-4" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-[var(--color-text-primary)] truncate">{display}</p>
                      <p className="text-xs text-[var(--color-text-muted)]">{type}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <Badge variant="accent">{type}</Badge>
                      {date && <span className="text-xs text-[var(--color-text-muted)]">{formatDate(date)}</span>}
                    </div>
                  </div>
                </Card>
              </motion.div>
            )
          })}
        </motion.div>
      ) : null}
    </PageWrapper>
  )
}
