'use client'

import { use } from 'react'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Pill, Upload, Shield, Clock, FlaskConical, FileText } from 'lucide-react'
import { fhirAPI, getPatientName, getPatientAge, formatGender } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Skeleton'
import { PatientTimeline } from '@/components/patients/PatientTimeline'
import { cn } from '@/lib/utils'

const subNavItems = [
  { href: (id: string) => `/patients/${id}`, label: 'Timeline', icon: Clock },
  { href: (id: string) => `/patients/${id}/labs`, label: 'Labs', icon: FlaskConical },
  { href: (id: string) => `/patients/${id}/prescribe`, label: 'Prescribe', icon: Pill },
  { href: (id: string) => `/patients/${id}/documents`, label: 'Documents', icon: FileText },
]

export default function PatientDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)

  const { data: patient, isLoading, isError } = useQuery({
    queryKey: ['patient', id],
    queryFn: async () => {
      const res = await fhirAPI.getPatient(id)
      return res.data
    },
    refetchInterval: 60_000,
  })

  if (isLoading) {
    return (
      <PageWrapper>
        <Skeleton className="h-6 w-48 mb-4" variant="text" />
        <Skeleton className="h-28 mb-6" />
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
      </PageWrapper>
    )
  }

  if (isError) {
    return (
      <PageWrapper>
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load patient details. Please try again later.
          </p>
          <Link
            href="/patients"
            className="inline-flex items-center gap-1 text-xs mt-4 transition-colors hover:text-[var(--color-text-secondary)]"
            style={{ color: 'var(--color-text-muted)' }}
          >
            <ArrowLeft size={12} /> Back to patients
          </Link>
        </div>
      </PageWrapper>
    )
  }

  const name = patient ? getPatientName(patient) : 'Unknown Patient'
  const age = patient ? getPatientAge(patient) : null

  return (
    <PageWrapper>
      <div className="mb-8">
        <Link
          href="/patients"
          className="inline-flex items-center gap-1 text-xs mb-4 transition-colors hover:text-[var(--color-text-secondary)]"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <ArrowLeft size={12} /> Patients
        </Link>
        <div className="flex items-start justify-between">
          <div>
            <h1 className="font-display text-[28px] leading-tight" style={{ color: 'var(--color-text-primary)' }}>
              {name}
            </h1>
            <p className="font-mono text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
              {patient ? formatGender(patient.gender) : '—'}{age ? ` · ${age}y` : ''} · {id}
            </p>
          </div>
          <div className="flex gap-1.5">
            <Link href={`/patients/${id}/prescribe`}>
              <Button variant="secondary" size="sm"><Pill size={13} /> Prescribe</Button>
            </Link>
            <Link href={`/patients/${id}/documents`}>
              <Button variant="secondary" size="sm"><Upload size={13} /> Upload</Button>
            </Link>
            <Link href={`/consents`}>
              <Button variant="ghost" size="sm"><Shield size={13} /> Consent</Button>
            </Link>
          </div>
        </div>
      </div>

      {/* Sub-navigation tabs */}
      <div className="flex gap-2 mb-8 border-b border-[var(--color-border)] pb-2 overflow-x-auto">
        {subNavItems.map((item) => {
          const Icon = item.icon
          const isActive = item.label === 'Timeline'
          return (
            <Link
              key={item.label}
              href={item.href(id)}
              className={cn(
                'inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg whitespace-nowrap transition-colors',
                isActive
                  ? 'bg-[var(--color-accent-subtle)] text-[var(--color-text-accent)]'
                  : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-elevated)]'
              )}
            >
              <Icon size={14} />
              {item.label}
            </Link>
          )
        })}
      </div>

      <PatientTimeline patientId={id} />
    </PageWrapper>
  )
}
