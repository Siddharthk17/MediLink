'use client'

import { use, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { fhirAPI, formatDate, getObservationValue } from '@medilink/shared'
import type { Observation } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { LabTrendChart } from '@/components/labs/LabTrendChart'
import { ObservationTable } from '@/components/labs/ObservationTable'
import { cn } from '@/lib/utils'

const LAB_CODES = [
  { code: '4548-4', label: 'HbA1c' },
  { code: '2160-0', label: 'Creatinine' },
  { code: '718-7', label: 'Hemoglobin' },
  { code: '3016-3', label: 'TSH' },
  { code: '1742-6', label: 'ALT' },
  { code: '2093-3', label: 'Cholesterol' },
]

export default function LabsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const [selectedCode, setSelectedCode] = useState(LAB_CODES[0].code)
  const selectedLabel = LAB_CODES.find((l) => l.code === selectedCode)?.label || ''

  const { data, isLoading, isError } = useQuery({
    queryKey: ['patient', id, 'labs', selectedCode],
    queryFn: async () => {
      const res = await fhirAPI.getLabTrends(id, selectedCode)
      return res.data
    },
    refetchInterval: 120_000,
  })

  const observations = (data?.entry?.map((e) => e.resource).filter((r) => r.resourceType === 'Observation') || []) as Observation[]
  const latest = observations[observations.length - 1]

  return (
    <PageWrapper>
      <Link href={`/patients/${id}`} className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to patient
      </Link>
      <h1 className="font-display text-[28px] mb-6" style={{ color: 'var(--color-text-primary)' }}>
        Lab Results & Trends
      </h1>

      {/* LOINC code selector */}
      <div className="flex gap-2 mb-6 overflow-x-auto pb-2">
        {LAB_CODES.map((lab) => (
          <button
            key={lab.code}
            onClick={() => setSelectedCode(lab.code)}
            className={cn(
              'px-4 py-2 text-sm font-medium rounded-full border whitespace-nowrap transition-colors',
              selectedCode === lab.code
                ? 'bg-[var(--color-accent-subtle)] text-[var(--color-accent)] border-[rgba(6,182,212,0.3)]'
                : 'text-[var(--color-text-muted)] border-[var(--color-border)] hover:border-[var(--color-border-hover)]'
            )}
          >
            {lab.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <Skeleton className="h-80 mb-6" />
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load lab results. Please try again later.
          </p>
        </div>
      ) : (
        <>
          <Card padding="md" className="mb-6">
            <LabTrendChart observations={observations} loincCode={selectedCode} />
          </Card>

          {latest && (
            <Card padding="md" className="mb-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-display text-lg" style={{ color: 'var(--color-text-primary)' }}>{selectedLabel}</p>
                  <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
                    Reported: {formatDate(latest.effectiveDateTime ?? '')}
                    {latest.referenceRange?.[0] && ` · Normal: ${latest.referenceRange[0].low?.value ?? '?'}–${latest.referenceRange[0].high?.value ?? '?'}`}
                  </p>
                </div>
                <p className="font-display text-2xl" style={{ color: 'var(--color-text-primary)' }}>
                  {getObservationValue(latest)}
                </p>
              </div>
            </Card>
          )}

          {observations.length > 0 && (
            <Card padding="sm">
              <ObservationTable observations={observations} />
            </Card>
          )}
        </>
      )}
    </PageWrapper>
  )
}
