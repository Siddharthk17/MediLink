'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Activity, TrendingUp, TrendingDown, Minus } from 'lucide-react'
import { fhirAPI, getCodeDisplay, getObservationValue, formatDate } from '@medilink/shared'
import type { CodeableConcept, Observation } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

function getInterpretationBadge(interpretation?: { coding?: Array<{ code?: string; display?: string }> }) {
  const code = interpretation?.coding?.[0]?.code
  if (!code) return null
  if (code === 'H' || code === 'HH') return <Badge variant="danger">High</Badge>
  if (code === 'L' || code === 'LL') return <Badge variant="warning">Low</Badge>
  if (code === 'N') return <Badge variant="success">Normal</Badge>
  return <Badge>{interpretation?.coding?.[0]?.display || code}</Badge>
}

export default function LabsPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'Observation'],
    queryFn: () => fhirAPI.searchResources('Observation', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const observations = data?.data?.entry ?? []

  const grouped = observations.reduce<Record<string, Array<{ resource: Observation }>>>((acc, entry) => {
    const resource = entry.resource as unknown as Observation
    const display = getCodeDisplay(resource.code) || 'Unknown'
    if (!acc[display]) acc[display] = []
    acc[display].push({ resource })
    return acc
  }, {})

  return (
    <PageWrapper title="Lab Results" subtitle="Your laboratory test results and trends">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-card" />
          ))}
        </div>
      ) : observations.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Activity className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No lab results on file</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {Object.entries(grouped).map(([name, entries]) => {
            const latest = entries[0].resource
            const value = getObservationValue(latest)
            const date = latest.effectiveDateTime || latest.meta?.lastUpdated
            const interp = latest.interpretation
            const ref = latest.referenceRange
            const refText = ref?.[0]?.text || (ref?.[0]?.low && ref?.[0]?.high ? `${ref[0].low.value}–${ref[0].high.value} ${ref[0].low.unit || ''}` : '')

            return (
              <motion.div key={name} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-xl bg-[var(--color-type-observation-subtle)] flex items-center justify-center">
                        <Activity className="w-5 h-5 text-[var(--color-type-observation)]" />
                      </div>
                      <div>
                        <p className="font-medium text-[var(--color-text-primary)]">{name}</p>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {date ? formatDate(date) : 'No date'}{entries.length > 1 ? ` · ${entries.length} results` : ''}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      {refText && (
                        <span className="text-xs text-[var(--color-text-muted)] hidden sm:block">
                          Ref: {refText}
                        </span>
                      )}
                      {getInterpretationBadge(interp?.[0])}
                      <span className="text-lg font-semibold text-[var(--color-text-primary)] font-mono">
                        {value || '—'}
                      </span>
                    </div>
                  </div>
                </Card>
              </motion.div>
            )
          })}
        </motion.div>
      )}
    </PageWrapper>
  )
}
