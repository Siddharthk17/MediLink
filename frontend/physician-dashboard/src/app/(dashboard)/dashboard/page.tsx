'use client'

import Link from 'next/link'
import { motion } from 'framer-motion'
import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  ArrowUpRight,
  Calendar,
  Clock,
  FileText,
  Plus,
  Users,
} from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import { Magnetic, TiltCard, WordReveal } from '@/components/aura/Interactions'
import { QuickActions } from '@/components/dashboard/QuickActions'
import { RecentActivity } from '@/components/dashboard/RecentActivity'
import { consentAPI, apiClient } from '@medilink/shared'
import type { ConsentedPatient } from '@medilink/shared'
import { Skeleton } from '@/components/ui/Skeleton'

function getGreeting() {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

function formatName(fullName?: string) {
  if (!fullName) return ''
  const parts = fullName.split(' ')
  return parts[parts.length - 1]
}

export default function DashboardPage() {
  const { user } = useAuthStore()
  const today = new Date().toLocaleDateString('en-IN', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })

  const { data: patientsData, isLoading: patientsLoading } = useQuery({
    queryKey: ['dashboard', 'my-patients'],
    queryFn: async () => {
      const res = await consentAPI.getMyPatients()
      return res.data
    },
  })

  const { data: docsData } = useQuery({
    queryKey: ['dashboard', 'document-jobs'],
    queryFn: async () => {
      const res = await apiClient.get<{ jobs: Array<{ jobId: string; status: string }>; total: number }>('/documents/jobs')
      return res.data
    },
  })

  const patients: ConsentedPatient[] = patientsData?.patients || []
  const patientCount = patients.length
  const firstPatient = patients[0] || null

  const pendingDocs = docsData?.jobs?.filter((j) => j.status === 'pending' || j.status === 'processing').length ?? 0
  const totalDocs = docsData?.total ?? 0

  return (
    <div className="max-w-7xl mx-auto relative z-10">
      <header className="mb-10 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <WordReveal
            text={`${getGreeting()}, Dr. ${formatName(user?.fullName) || 'Physician'}.`}
            className="text-4xl md:text-6xl font-medium tracking-tighter text-[var(--color-text-primary)]"
          />
          <motion.p
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.35, duration: 0.8 }}
            className="text-[var(--color-text-muted)] text-lg"
          >
            {today} • <span className="text-[var(--color-text-primary)] font-medium">{patientCount} consented patient{patientCount !== 1 ? 's' : ''}</span>
          </motion.p>
        </div>
        <Magnetic>
          <Link
            href="/patients"
            className="inline-flex items-center gap-2 bg-[var(--color-text-primary)] text-[var(--color-text-inverse)] px-6 py-3.5 rounded-full text-sm font-medium shadow-lg shadow-black/10"
          >
            <Plus className="w-4 h-4" /> New Consultation
          </Link>
        </Magnetic>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 auto-rows-[220px]">
        {/* First patient card */}
        <TiltCard className="md:col-span-2 md:row-span-2 bg-[var(--color-bg-surface)] rounded-[2rem] p-8 shadow-card border border-[var(--color-border)] flex flex-col justify-between relative overflow-hidden">
          {patientsLoading ? (
            <div className="space-y-4 flex-1 flex flex-col justify-center">
              <Skeleton className="h-16 w-3/4" />
              <Skeleton className="h-8 w-1/2" />
              <div className="grid grid-cols-3 gap-3 mt-4">
                <Skeleton className="h-20" />
                <Skeleton className="h-20" />
                <Skeleton className="h-20" />
              </div>
            </div>
          ) : firstPatient ? (
            <>
              <div className="flex justify-between items-start">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--color-accent-subtle)] text-[var(--color-text-accent)] text-xs font-medium mb-6">
                  <Clock className="w-3 h-3" /> Most Recent Patient
                </div>
                <Link
                  href={`/patients/${firstPatient.patient.fhirId}`}
                  className="w-10 h-10 rounded-full border border-[var(--color-border)] flex items-center justify-center hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  <ArrowUpRight className="w-4 h-4 text-[var(--color-text-secondary)]" />
                </Link>
              </div>
              <div className="space-y-6">
                <div className="flex items-center gap-5">
                  <div className="w-20 h-20 rounded-full border-2 border-[var(--color-bg-surface)] shadow-sm bg-[var(--color-bg-elevated)] flex items-center justify-center text-2xl font-display text-[var(--color-text-primary)]">
                    {firstPatient.patient.fullName?.split(' ').map((n) => n[0]).join('').slice(0, 2) || '?'}
                  </div>
                  <div>
                    <h3 className="text-3xl font-medium tracking-tight text-[var(--color-text-primary)]">
                      {firstPatient.patient.fullName || 'Unknown Patient'}
                    </h3>
                    <p className="text-[var(--color-text-muted)] text-lg">
                      {firstPatient.patient.gender ? `${firstPatient.patient.gender.charAt(0).toUpperCase()}${firstPatient.patient.gender.slice(1)}` : ''}
                      {firstPatient.patient.birthDate ? ` • Born ${firstPatient.patient.birthDate}` : ''}
                    </p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="bg-[var(--color-bg-elevated)] rounded-2xl p-4">
                    <div className="text-[var(--color-text-muted)] text-xs uppercase tracking-wider mb-2">Consent</div>
                    <div className="text-lg font-medium text-[var(--color-success)] capitalize">{firstPatient.consent.status}</div>
                  </div>
                  <div className="bg-[var(--color-bg-elevated)] rounded-2xl p-4">
                    <div className="text-[var(--color-text-muted)] text-xs uppercase tracking-wider mb-2">Scope</div>
                    <div className="text-lg font-medium text-[var(--color-text-primary)]">
                      {firstPatient.consent.scope.includes('*') ? 'Full Access' : firstPatient.consent.scope.join(', ')}
                    </div>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center justify-center flex-1 text-center">
              <Users className="w-12 h-12 text-[var(--color-text-muted)] mb-4" />
              <h3 className="text-xl font-medium text-[var(--color-text-primary)] mb-2">No patients yet</h3>
              <p className="text-sm text-[var(--color-text-muted)]">Patient consents will appear here once granted</p>
            </div>
          )}
        </TiltCard>

        {/* Total patients stat */}
        <TiltCard
          className="rounded-[2rem] p-8 shadow-sm border flex flex-col justify-between relative overflow-hidden bg-[var(--color-bg-surface)] border-[var(--color-border)] text-[var(--color-text-primary)]"
        >
          <div>
            <Users className="w-6 h-6 mb-4 text-[var(--color-text-muted)]" />
            <p className="text-[var(--color-text-muted)] text-sm uppercase tracking-wider mb-1">
              My Patients
            </p>
            {patientsLoading ? (
              <Skeleton className="h-10 w-20" />
            ) : (
              <h3 className="text-4xl font-medium tracking-tight">{patientCount}</h3>
            )}
          </div>
          <Link href="/patients" className="text-sm font-medium text-[var(--color-accent)] hover:underline">
            View all →
          </Link>
        </TiltCard>

        {/* Pending reports stat */}
        <TiltCard className="bg-[var(--color-bg-surface)] rounded-[2rem] p-8 shadow-card border border-[var(--color-border)] flex flex-col justify-between">
          <div>
            <FileText className="w-6 h-6 text-[var(--color-text-muted)] mb-4" />
            <p className="text-[var(--color-text-muted)] text-sm uppercase tracking-wider mb-1">Documents</p>
            <h3 className="text-4xl font-medium tracking-tight text-[var(--color-text-primary)]">{totalDocs}</h3>
          </div>
          {pendingDocs > 0 && (
            <div className="text-[var(--color-warning)] text-sm font-medium">{pendingDocs} processing</div>
          )}
        </TiltCard>

        {/* Consented patients list */}
        <div className="md:col-span-2 bg-[var(--color-bg-surface)] rounded-[2rem] p-8 shadow-card border border-[var(--color-border)] flex flex-col">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-xl font-medium tracking-tight text-[var(--color-text-primary)]">Consented Patients</h3>
            <Link href="/consents" className="text-sm text-[var(--color-text-accent)]">Manage Consents</Link>
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto pr-1 space-y-3">
            {patientsLoading ? (
              Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-16" />)
            ) : patients.length === 0 ? (
              <div className="text-center py-8">
                <p className="text-sm text-[var(--color-text-muted)]">No patients with active consent</p>
              </div>
            ) : (
              patients.map((cp) => (
                <Link
                  key={cp.patient.fhirId}
                  href={`/patients/${cp.patient.fhirId}`}
                  className="flex items-start gap-4 p-4 rounded-2xl bg-[var(--color-bg-elevated)] hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  <div className="w-9 h-9 rounded-full bg-[var(--color-accent-subtle)] flex items-center justify-center text-xs font-semibold text-[var(--color-accent)] shrink-0">
                    {cp.patient.fullName?.split(' ').map((n) => n[0]).join('').slice(0, 2) || '?'}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-[var(--color-text-primary)] font-medium text-sm truncate">
                      {cp.patient.fullName || 'Unknown'}
                    </p>
                    <p className="text-[var(--color-text-muted)] text-xs">
                      {cp.patient.gender || ''}{cp.patient.birthDate ? ` • ${cp.patient.birthDate}` : ''} • Scope: {cp.consent.scope.join(', ')}
                    </p>
                  </div>
                </Link>
              ))
            )}
          </div>
        </div>
      </div>

      <div className="mt-6">
        <QuickActions />
      </div>

      <div className="mt-6">
        <RecentActivity />
      </div>

      <div className="mt-6">
        <Link
          href="/patients"
          className="inline-flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
        >
          <Calendar size={15} />
          Open patient list
        </Link>
      </div>
    </div>
  )
}
