'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { AlertTriangle, Calendar } from 'lucide-react'
import { fhirAPI, getCodeDisplay, formatDate } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function AllergiesPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'AllergyIntolerance'],
    queryFn: () => fhirAPI.searchResources('AllergyIntolerance', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const allergies = data?.data?.entry ?? []

  return (
    <PageWrapper title="Allergies" subtitle="Your known allergies and adverse reactions">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-card" />
          ))}
        </div>
      ) : allergies.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <AlertTriangle className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No allergies recorded</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {allergies.map((entry, i) => {
            const a = entry.resource as Record<string, any>
            const display = getCodeDisplay(a.code) || 'Unknown Allergen'
            const criticality = a.criticality
            const category = a.category?.[0]
            const clinicalStatus = a.clinicalStatus?.coding?.[0]?.code
            const reactions = a.reaction?.map((r: Record<string, any>) =>
              r.manifestation?.[0]?.coding?.[0]?.display
            ).filter(Boolean) ?? []
            const recordedDate = a.recordedDate

            return (
              <motion.div key={a.id ?? `allergy-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-[var(--color-type-allergy-subtle)] flex items-center justify-center shrink-0">
                      <AlertTriangle className="w-5 h-5 text-[var(--color-type-allergy)]" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="font-medium text-[var(--color-text-primary)]">{display}</h3>
                        <div className="flex items-center gap-2 shrink-0">
                          {criticality && (
                            <Badge variant={criticality === 'high' ? 'danger' : criticality === 'low' ? 'success' : 'warning'}>
                              {criticality}
                            </Badge>
                          )}
                        </div>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                        {category && <span className="capitalize">{category}</span>}
                        {clinicalStatus && <span className="capitalize">{clinicalStatus}</span>}
                        {recordedDate && (
                          <span className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" /> {formatDate(recordedDate)}
                          </span>
                        )}
                      </div>
                      {reactions.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1.5">
                          {reactions.map((r: string, ri: number) => (
                            <Badge key={ri} variant="warning">{r}</Badge>
                          ))}
                        </div>
                      )}
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
