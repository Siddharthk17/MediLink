'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Activity, Pill, Stethoscope, Syringe, AlertTriangle,
  FileText, Clock, Calendar
} from 'lucide-react'
import { fhirAPI, getCodeDisplay, formatDate, formatRelative } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

const resourceTypeConfig: Record<string, { icon: React.ElementType; colorVar: string }> = {
  Observation: { icon: Activity, colorVar: 'observation' },
  MedicationRequest: { icon: Pill, colorVar: 'medication' },
  Condition: { icon: Stethoscope, colorVar: 'condition' },
  Immunization: { icon: Syringe, colorVar: 'immunization' },
  AllergyIntolerance: { icon: AlertTriangle, colorVar: 'allergy' },
  Encounter: { icon: Calendar, colorVar: 'encounter' },
  DiagnosticReport: { icon: FileText, colorVar: 'diagnostic' },
}

function getResourceDate(resource: Record<string, unknown>): string | undefined {
  return (resource.effectiveDateTime as string)
    || (resource.authoredOn as string)
    || (resource.recordedDate as string)
    || (resource.occurrenceDateTime as string)
    || (resource.meta as Record<string, string>)?.lastUpdated
}

export default function TimelinePage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'timeline', fhirId],
    queryFn: () => fhirAPI.getTimeline(fhirId!),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const entries = data?.data?.entry ?? []

  return (
    <PageWrapper title="Timeline" subtitle="Complete chronological view of your health events">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-card" />
          ))}
        </div>
      ) : entries.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Clock className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No health events recorded yet</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate">
          <div className="relative">
            <div className="absolute left-5 top-0 bottom-0 w-px bg-[var(--color-border)]" />

            <div className="space-y-4">
              {entries.map((entry, i) => {
                const resource = entry.resource as Record<string, any>
                const type = resource.resourceType
                const config = resourceTypeConfig[type] || { icon: FileText, colorVar: 'diagnostic' }
                const Icon = config.icon
                const display = getCodeDisplay(resource.code)
                  || resource.vaccineCode?.coding?.[0]?.display
                  || resource.medicationCodeableConcept?.text
                  || resource.medicationCodeableConcept?.coding?.[0]?.display
                  || type
                const date = getResourceDate(resource)
                const status = resource.status || resource.clinicalStatus?.coding?.[0]?.code

                return (
                  <motion.div key={resource.id ?? `tl-${i}`} variants={staggerItem} className="relative pl-12">
                    <div
                      className="absolute left-3 top-4 w-5 h-5 rounded-full border-2 border-[var(--color-bg-base)] flex items-center justify-center z-10"
                      style={{
                        backgroundColor: `var(--color-type-${config.colorVar}, var(--color-text-muted))`,
                      }}
                    >
                      <Icon className="w-2.5 h-2.5 text-white" />
                    </div>

                    <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <Badge variant="accent">{type}</Badge>
                            {status && (
                              <Badge variant={status === 'active' ? 'success' : 'default'}>
                                {status}
                              </Badge>
                            )}
                          </div>
                          <p className="mt-1.5 font-medium text-[var(--color-text-primary)] truncate">
                            {display}
                          </p>
                        </div>
                        {date && (
                          <div className="text-right shrink-0">
                            <p className="text-xs text-[var(--color-text-muted)]">{formatDate(date)}</p>
                            <p className="text-[10px] text-[var(--color-text-muted)]">{formatRelative(date)}</p>
                          </div>
                        )}
                      </div>
                    </Card>
                  </motion.div>
                )
              })}
            </div>
          </div>
        </motion.div>
      )}
    </PageWrapper>
  )
}
