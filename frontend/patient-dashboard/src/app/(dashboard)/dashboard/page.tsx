'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Activity, Heart, Pill, Stethoscope, Shield,
  Syringe, AlertTriangle, FileText, ArrowRight, Clock
} from 'lucide-react'
import Link from 'next/link'
import { fhirAPI, consentAPI, getCodeDisplay, formatRelative } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { TiltCard, WordReveal } from '@/components/aura/Interactions'
import { staggerContainer, staggerItem, cardReveal } from '@/lib/motion'

function StatCard({ label, value, icon: Icon, color, href }: {
  label: string
  value: number | string
  icon: React.ElementType
  color: string
  href: string
}) {
  return (
    <TiltCard>
      <Link href={href}>
        <Card className="h-full hover:border-[var(--color-border-hover)] transition-all duration-300 group cursor-pointer">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-sm text-[var(--color-text-muted)] mb-1">{label}</p>
              <p className="text-3xl font-semibold tracking-tight text-[var(--color-text-primary)]">
                {value}
              </p>
            </div>
            <div className="w-10 h-10 rounded-2xl flex items-center justify-center" style={{ backgroundColor: `${color}15` }}>
              <Icon className="w-5 h-5" style={{ color }} />
            </div>
          </div>
          <div className="mt-3 flex items-center gap-1 text-xs text-[var(--color-text-muted)] group-hover:text-[var(--color-accent)] transition-colors">
            View details <ArrowRight className="w-3 h-3" />
          </div>
        </Card>
      </Link>
    </TiltCard>
  )
}

export default function DashboardPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data: conditions, isLoading: loadingConditions } = useQuery({
    queryKey: ['fhir', 'Condition'],
    queryFn: () => fhirAPI.searchResources('Condition', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const { data: medications, isLoading: loadingMeds } = useQuery({
    queryKey: ['fhir', 'MedicationRequest'],
    queryFn: () => fhirAPI.searchResources('MedicationRequest', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const { data: observations, isLoading: loadingObs } = useQuery({
    queryKey: ['fhir', 'Observation'],
    queryFn: () => fhirAPI.searchResources('Observation', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const { data: allergies, isLoading: loadingAllergies } = useQuery({
    queryKey: ['fhir', 'AllergyIntolerance'],
    queryFn: () => fhirAPI.searchResources('AllergyIntolerance', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const { data: immunizations, isLoading: loadingImm } = useQuery({
    queryKey: ['fhir', 'Immunization'],
    queryFn: () => fhirAPI.searchResources('Immunization', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const { data: consents, isLoading: loadingConsents } = useQuery({
    queryKey: ['consents', 'my-grants'],
    queryFn: () => consentAPI.getMyGrants(),
    refetchInterval: 30_000,
  })

  const { data: timeline, isLoading: loadingTimeline } = useQuery({
    queryKey: ['fhir', 'timeline', fhirId],
    queryFn: () => fhirAPI.getTimeline(fhirId!),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const conditionCount = conditions?.data?.entry?.length ?? 0
  const medCount = medications?.data?.entry?.length ?? 0
  const obsCount = observations?.data?.entry?.length ?? 0
  const allergyCount = allergies?.data?.entry?.length ?? 0
  const immCount = immunizations?.data?.entry?.length ?? 0
  const activeConsents = consents?.data?.consents?.filter((c) => c.status === 'active')?.length ?? 0

  const recentTimeline = timeline?.data?.entry?.slice(0, 5) ?? []
  const isLoading = loadingConditions || loadingMeds || loadingObs || loadingAllergies || loadingImm || loadingConsents

  const firstName = user?.fullName?.split(' ')[0] || 'there'

  return (
    <PageWrapper>
      <motion.div variants={staggerContainer} initial="initial" animate="animate">
        <div className="mb-10">
          <WordReveal
            text={`Hello, ${firstName}`}
            className="font-display text-5xl md:text-6xl tracking-tight text-gradient"
          />
          <motion.p variants={staggerItem} className="mt-3 text-lg text-[var(--color-text-muted)]">
            Here&apos;s your health at a glance
          </motion.p>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-10">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-[130px] rounded-card" />
            ))}
          </div>
        ) : (
          <motion.div variants={staggerItem} className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-10">
            <StatCard label="Conditions" value={conditionCount} icon={Stethoscope} color="#8B5CF6" href="/health/conditions" />
            <StatCard label="Medications" value={medCount} icon={Pill} color="#10B981" href="/health/medications" />
            <StatCard label="Lab Results" value={obsCount} icon={Activity} color="#06B6D4" href="/health/labs" />
            <StatCard label="Allergies" value={allergyCount} icon={AlertTriangle} color="#F43F5E" href="/health/allergies" />
            <StatCard label="Vaccinations" value={immCount} icon={Syringe} color="#6366F1" href="/health/immunizations" />
            <StatCard label="Consents" value={activeConsents} icon={Shield} color="#0d9488" href="/consents" />
          </motion.div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
          <motion.div variants={cardReveal} className="lg:col-span-3">
            <Card variant="glass" className="h-full">
              <div className="flex items-center justify-between mb-5">
                <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Recent Activity</h2>
                <Link href="/timeline" className="text-sm text-[var(--color-accent)] hover:underline flex items-center gap-1">
                  View all <ArrowRight className="w-3 h-3" />
                </Link>
              </div>
              {loadingTimeline ? (
                <div className="space-y-3">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-14 rounded-xl" />
                  ))}
                </div>
              ) : recentTimeline.length === 0 ? (
                <p className="text-sm text-[var(--color-text-muted)] py-8 text-center">
                  No recent health events
                </p>
              ) : (
                <div className="space-y-2">
                  {recentTimeline.map((entry, i) => {
                    const resource = entry.resource as Record<string, any>
                    const type = resource.resourceType
                    const display = getCodeDisplay(resource.code) || type
                    const date = resource.effectiveDateTime || resource.authoredOn || resource.recordedDate || resource.occurrenceDateTime || resource.meta?.lastUpdated
                    return (
                      <div key={resource.id ?? `tl-${i}`} className="flex items-center gap-3 p-3 rounded-xl hover:bg-[var(--color-bg-hover)] transition-colors">
                        <div className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style={{
                          backgroundColor: `var(--color-type-${type.toLowerCase()}-subtle, var(--color-bg-elevated))`,
                          color: `var(--color-type-${type.toLowerCase()}, var(--color-text-muted))`
                        }}>
                          {type === 'MedicationRequest' ? <Pill className="w-4 h-4" /> :
                           type === 'Observation' ? <Activity className="w-4 h-4" /> :
                           type === 'Condition' ? <Stethoscope className="w-4 h-4" /> :
                           type === 'Immunization' ? <Syringe className="w-4 h-4" /> :
                           type === 'AllergyIntolerance' ? <AlertTriangle className="w-4 h-4" /> :
                           <FileText className="w-4 h-4" />}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-[var(--color-text-primary)] truncate">{display}</p>
                          <p className="text-xs text-[var(--color-text-muted)]">{type}</p>
                        </div>
                        {date && (
                          <span className="text-xs text-[var(--color-text-muted)] flex items-center gap-1 shrink-0">
                            <Clock className="w-3 h-3" />
                            {formatRelative(date)}
                          </span>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </Card>
          </motion.div>

          <motion.div variants={cardReveal} className="lg:col-span-2 space-y-6">
            <Card variant="glass">
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)] mb-4">Active Conditions</h2>
              {loadingConditions ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-10 rounded-lg" />)}
                </div>
              ) : conditionCount === 0 ? (
                <p className="text-sm text-[var(--color-text-muted)] py-4 text-center">No conditions on file</p>
              ) : (
                <div className="space-y-2">
                  {conditions?.data?.entry?.slice(0, 4).map((entry, i) => {
                    const c = entry.resource as Record<string, any>
                    const display = getCodeDisplay(c.code) || 'Unknown Condition'
                    const severity = c.severity?.coding?.[0]?.display
                    return (
                      <div key={c.id ?? `cond-${i}`} className="flex items-center justify-between p-2.5 rounded-lg bg-[var(--color-bg-elevated)]">
                        <span className="text-sm text-[var(--color-text-primary)]">{display}</span>
                        {severity && <Badge variant={severity === 'Severe' ? 'danger' : severity === 'Moderate' ? 'warning' : 'default'}>{severity}</Badge>}
                      </div>
                    )
                  })}
                  {conditionCount > 4 && (
                    <Link href="/health/conditions" className="block text-center text-xs text-[var(--color-accent)] hover:underline pt-1">
                      +{conditionCount - 4} more
                    </Link>
                  )}
                </div>
              )}
            </Card>

            <Card variant="glass">
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)] mb-4">Current Medications</h2>
              {loadingMeds ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-10 rounded-lg" />)}
                </div>
              ) : medCount === 0 ? (
                <p className="text-sm text-[var(--color-text-muted)] py-4 text-center">No active medications</p>
              ) : (
                <div className="space-y-2">
                  {medications?.data?.entry?.slice(0, 4).map((entry, i) => {
                    const med = entry.resource as Record<string, any>
                    const display = med.medicationCodeableConcept?.text
                      || med.medicationCodeableConcept?.coding?.[0]?.display
                      || 'Medication'
                    const status = med.status
                    return (
                      <div key={med.id ?? `med-${i}`} className="flex items-center justify-between p-2.5 rounded-lg bg-[var(--color-bg-elevated)]">
                        <span className="text-sm text-[var(--color-text-primary)] truncate mr-2">{display}</span>
                        <Badge variant={status === 'active' ? 'success' : 'default'}>{status}</Badge>
                      </div>
                    )
                  })}
                  {medCount > 4 && (
                    <Link href="/health/medications" className="block text-center text-xs text-[var(--color-accent)] hover:underline pt-1">
                      +{medCount - 4} more
                    </Link>
                  )}
                </div>
              )}
            </Card>
          </motion.div>
        </div>
      </motion.div>
    </PageWrapper>
  )
}
