'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity } from 'lucide-react'
import { fhirAPI } from '@medilink/shared'
import { TimelineEvent } from './TimelineEvent'
import { Skeleton } from '@/components/ui/Skeleton'
import { cn } from '@/lib/utils'
import React from 'react'

interface PatientTimelineProps {
  patientId: string
}

const RESOURCE_FILTERS = ['All', 'Encounters', 'Conditions', 'Medications', 'Labs', 'Reports', 'Allergies', 'Immunizations']

const filterToType: Record<string, string | undefined> = {
  All: undefined,
  Encounters: 'Encounter',
  Conditions: 'Condition',
  Medications: 'MedicationRequest',
  Labs: 'Observation',
  Reports: 'DiagnosticReport',
  Allergies: 'AllergyIntolerance',
  Immunizations: 'Immunization',
}

export const PatientTimeline = React.memo(function PatientTimeline({ patientId }: PatientTimelineProps) {
  const [activeFilter, setActiveFilter] = useState('All')

  const { data, isLoading, error } = useQuery({
    queryKey: ['patient', patientId, 'timeline', activeFilter],
    queryFn: async () => {
      const params: Record<string, string> = {}
      const filterType = filterToType[activeFilter]
      if (filterType) params._type = filterType
      const res = await fhirAPI.getTimeline(patientId, params)
      return res.data
    },
    refetchInterval: 120_000,
  })

  const entries = data?.entry || []

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>Failed to load timeline.</p>
      </div>
    )
  }

  return (
    <div>
      {/* Filter pills */}
      <div className="flex gap-2 mb-6 flex-wrap">
        {RESOURCE_FILTERS.map((filter) => (
          <button
            key={filter}
            onClick={() => setActiveFilter(filter)}
            className={cn(
              'px-3 py-1.5 text-xs font-medium rounded-full border transition-colors',
              activeFilter === filter
                ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)] border-[rgba(6,182,212,0.3)]'
                : 'text-[var(--color-text-muted)] border-[var(--color-border)] hover:border-[var(--color-border-hover)]'
            )}
          >
            {filter}
          </button>
        ))}
      </div>

      {/* Timeline */}
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex gap-4">
              <Skeleton variant="circular" className="w-10 h-10" />
              <Skeleton className="flex-1 h-24" />
            </div>
          ))}
        </div>
      ) : entries.length === 0 ? (
        <div className="text-center py-12">
          <Activity size={48} className="mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>No records found for this patient</p>
        </div>
      ) : (
        <div>
          {entries.map((entry, i: number) => (
            <TimelineEvent key={(entry.resource as unknown as { id?: string }).id ?? `timeline-${i}`} resource={entry.resource as unknown as { resourceType: string; id: string; [key: string]: unknown }} isLast={i === entries.length - 1} />
          ))}
        </div>
      )}
    </div>
  )
})
