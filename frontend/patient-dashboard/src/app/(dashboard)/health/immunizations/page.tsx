'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Syringe, Calendar, CheckCircle } from 'lucide-react'
import { fhirAPI, formatDate } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function ImmunizationsPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'Immunization'],
    queryFn: () => fhirAPI.searchResources('Immunization', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const immunizations = data?.data?.entry ?? []

  return (
    <PageWrapper title="Immunizations" subtitle="Your vaccination history">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-card" />
          ))}
        </div>
      ) : immunizations.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Syringe className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No immunization records</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {immunizations.map((entry, i) => {
            const imm = entry.resource as Record<string, any>
            const name = imm.vaccineCode?.coding?.[0]?.display
              || imm.vaccineCode?.text
              || 'Unknown Vaccine'
            const status = imm.status || 'completed'
            const date = imm.occurrenceDateTime
            const lotNumber = imm.lotNumber
            const site = imm.site?.coding?.[0]?.display
            const doseNumber = imm.protocolApplied?.[0]?.doseNumberPositiveInt

            return (
              <motion.div key={imm.id ?? `imm-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-[var(--color-type-immunization-subtle)] flex items-center justify-center shrink-0">
                      <Syringe className="w-5 h-5 text-[var(--color-type-immunization)]" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="font-medium text-[var(--color-text-primary)]">{name}</h3>
                        <Badge variant={status === 'completed' ? 'success' : 'default'}>
                          <CheckCircle className="w-3 h-3 mr-1" />
                          {status}
                        </Badge>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                        {date && (
                          <span className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" /> {formatDate(date)}
                          </span>
                        )}
                        {doseNumber && <span>Dose #{doseNumber}</span>}
                        {lotNumber && <span>Lot: {lotNumber}</span>}
                        {site && <span>Site: {site}</span>}
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
