'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Pill, Calendar, User } from 'lucide-react'
import { fhirAPI, formatDate } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function MedicationsPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'MedicationRequest'],
    queryFn: () => fhirAPI.searchResources('MedicationRequest', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const medications = data?.data?.entry ?? []

  return (
    <PageWrapper title="Medications" subtitle="Your current and past prescriptions">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-card" />
          ))}
        </div>
      ) : medications.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Pill className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No medications on file</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {medications.map((entry, i) => {
            const med = entry.resource as Record<string, any>
            const name = med.medicationCodeableConcept?.text
              || med.medicationCodeableConcept?.coding?.[0]?.display
              || 'Unknown Medication'
            const status = med.status || 'unknown'
            const dosage = med.dosageInstruction?.[0]
            const dosageText = dosage?.text || (dosage?.doseAndRate?.[0]?.doseQuantity
              ? `${dosage.doseAndRate[0].doseQuantity.value} ${dosage.doseAndRate[0].doseQuantity.unit || ''}`
              : '')
            const frequency = dosage?.timing?.repeat?.frequency
              ? `${dosage.timing.repeat.frequency}x/${dosage.timing.repeat.period} ${dosage.timing.repeat.periodUnit || ''}`
              : ''
            const prescriber = med.requester?.display
            const date = med.authoredOn

            return (
              <motion.div key={med.id ?? `med-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-[var(--color-type-medication-subtle)] flex items-center justify-center shrink-0">
                      <Pill className="w-5 h-5 text-[var(--color-type-medication)]" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="font-medium text-[var(--color-text-primary)]">{name}</h3>
                        <Badge variant={status === 'active' ? 'success' : status === 'completed' ? 'default' : 'warning'}>
                          {status}
                        </Badge>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                        {dosageText && <span>{dosageText}</span>}
                        {frequency && <span>{frequency}</span>}
                        {prescriber && (
                          <span className="flex items-center gap-1">
                            <User className="w-3 h-3" /> {prescriber}
                          </span>
                        )}
                        {date && (
                          <span className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" /> {formatDate(date)}
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
