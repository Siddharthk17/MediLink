'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Stethoscope, Calendar } from 'lucide-react'
import { fhirAPI, getCodeDisplay, formatDate } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function ConditionsPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'Condition'],
    queryFn: () => fhirAPI.searchResources('Condition', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const conditions = data?.data?.entry ?? []

  return (
    <PageWrapper title="Conditions" subtitle="Your diagnosed conditions and health issues">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-card" />
          ))}
        </div>
      ) : conditions.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Stethoscope className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No conditions on file</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {conditions.map((entry, i) => {
            const c = entry.resource as Record<string, any>
            const display = getCodeDisplay(c.code) || 'Unknown Condition'
            const severity = c.severity?.coding?.[0]?.display
            const clinicalStatus = c.clinicalStatus?.coding?.[0]?.code
            const onsetDate = c.onsetDateTime
            const category = c.category?.[0]?.coding?.[0]?.display

            return (
              <motion.div key={c.id ?? `cond-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-[var(--color-type-condition-subtle)] flex items-center justify-center shrink-0">
                      <Stethoscope className="w-5 h-5 text-[var(--color-type-condition)]" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="font-medium text-[var(--color-text-primary)]">{display}</h3>
                        <div className="flex items-center gap-2 shrink-0">
                          {severity && (
                            <Badge variant={severity === 'Severe' ? 'danger' : severity === 'Moderate' ? 'warning' : 'default'}>
                              {severity}
                            </Badge>
                          )}
                          {clinicalStatus && (
                            <Badge variant={clinicalStatus === 'active' ? 'accent' : 'default'}>
                              {clinicalStatus}
                            </Badge>
                          )}
                        </div>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                        {category && <span>{category}</span>}
                        {onsetDate && (
                          <span className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" /> Onset: {formatDate(onsetDate)}
                          </span>
                        )}
                      </div>
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
