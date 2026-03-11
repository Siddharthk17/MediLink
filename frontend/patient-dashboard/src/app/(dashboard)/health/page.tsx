'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import Link from 'next/link'
import {
  Heart, Activity, Pill, Stethoscope, Syringe,
  AlertTriangle, ArrowRight
} from 'lucide-react'
import { fhirAPI } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { TiltCard } from '@/components/aura/Interactions'
import { staggerContainer, staggerItem } from '@/lib/motion'

const sections = [
  { key: 'Condition', label: 'Conditions', desc: 'Diagnoses and health issues', icon: Stethoscope, color: '#8B5CF6', href: '/health/conditions' },
  { key: 'MedicationRequest', label: 'Medications', desc: 'Prescriptions and drugs', icon: Pill, color: '#10B981', href: '/health/medications' },
  { key: 'Observation', label: 'Lab Results', desc: 'Tests and observations', icon: Activity, color: '#06B6D4', href: '/health/labs' },
  { key: 'AllergyIntolerance', label: 'Allergies', desc: 'Known allergies and reactions', icon: AlertTriangle, color: '#F43F5E', href: '/health/allergies' },
  { key: 'Immunization', label: 'Immunizations', desc: 'Vaccination records', icon: Syringe, color: '#6366F1', href: '/health/immunizations' },
]

function useHealthSectionQuery(resourceType: string, fhirId: string | undefined) {
  return useQuery({
    queryKey: ['fhir', resourceType],
    queryFn: () => fhirAPI.searchResources(resourceType, { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })
}

export default function HealthPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const conditionQuery = useHealthSectionQuery('Condition', fhirId)
  const medicationQuery = useHealthSectionQuery('MedicationRequest', fhirId)
  const observationQuery = useHealthSectionQuery('Observation', fhirId)
  const allergyQuery = useHealthSectionQuery('AllergyIntolerance', fhirId)
  const immunizationQuery = useHealthSectionQuery('Immunization', fhirId)

  const queries = sections.map((s) => ({
    ...s,
    query: s.key === 'Condition' ? conditionQuery
      : s.key === 'MedicationRequest' ? medicationQuery
      : s.key === 'Observation' ? observationQuery
      : s.key === 'AllergyIntolerance' ? allergyQuery
      : immunizationQuery,
  }))

  return (
    <PageWrapper title="My Health" subtitle="Browse your complete medical records">
      <motion.div variants={staggerContainer} initial="initial" animate="animate" className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        {queries.map(({ key, label, desc, icon: Icon, color, href, query }) => (
          <motion.div key={key} variants={staggerItem}>
            <TiltCard>
              <Link href={href}>
                <Card className="h-full hover:border-[var(--color-border-hover)] transition-all duration-300 group cursor-pointer">
                  <div className="flex items-start gap-4">
                    <div className="w-12 h-12 rounded-2xl flex items-center justify-center shrink-0" style={{ backgroundColor: `${color}15` }}>
                      <Icon className="w-6 h-6" style={{ color }} />
                    </div>
                    <div className="flex-1">
                      <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">{label}</h3>
                      <p className="text-sm text-[var(--color-text-muted)] mt-0.5">{desc}</p>
                      <div className="mt-3 flex items-center justify-between">
                        {query.isLoading ? (
                          <Skeleton className="h-5 w-16" />
                        ) : (
                          <span className="text-2xl font-semibold text-[var(--color-text-primary)]">
                            {query.data?.data?.entry?.length ?? 0}
                          </span>
                        )}
                        <span className="text-sm text-[var(--color-text-muted)] group-hover:text-[var(--color-accent)] transition-colors flex items-center gap-1">
                          View <ArrowRight className="w-3.5 h-3.5" />
                        </span>
                      </div>
                    </div>
                  </div>
                </Card>
              </Link>
            </TiltCard>
          </motion.div>
        ))}
      </motion.div>
    </PageWrapper>
  )
}
