'use client'

import Link from 'next/link'
import React from 'react'
import { ChevronRight, Users } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { consentAPI } from '@medilink/shared'
import type { ConsentedPatient } from '@medilink/shared'

const consentDotColor: Record<string, string> = {
  active: 'var(--color-success)',
  expired: 'var(--color-warning)',
  revoked: 'var(--color-danger)',
}

export const TodaysPatients = React.memo(function TodaysPatients() {
  const { data, isLoading } = useQuery({
    queryKey: ['sidebar', 'my-patients'],
    queryFn: async () => {
      const res = await consentAPI.getMyPatients()
      return res.data
    },
    staleTime: 60_000,
  })

  const patients: ConsentedPatient[] = data?.patients || []

  return (
    <Card padding="lg" className="h-full">
      <div className="flex items-center justify-between mb-6">
        <h2 className="font-display text-2xl text-[var(--color-text-primary)]">My Patients</h2>
        <Link
          href="/patients"
          className="text-sm font-medium text-[var(--color-text-accent)] hover:opacity-80 transition-opacity"
        >
          View all
        </Link>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-14" />)}
        </div>
      ) : patients.length === 0 ? (
        <div className="text-center py-8">
          <Users className="w-8 h-8 text-[var(--color-text-muted)] mx-auto mb-2" />
          <p className="text-sm text-[var(--color-text-muted)]">No consented patients</p>
        </div>
      ) : (
        <div className="space-y-2">
          {patients.map((cp) => (
            <Link
              key={cp.patient.fhirId}
              href={`/patients/${cp.patient.fhirId}`}
              className="flex items-center justify-between gap-4 rounded-[18px] px-4 py-3 hover:bg-[var(--color-bg-hover)] transition-colors group"
            >
              <div className="flex items-center gap-4">
                <div className="w-8 h-8 rounded-full bg-[var(--color-accent-subtle)] flex items-center justify-center text-xs font-semibold text-[var(--color-accent)]">
                  {cp.patient.fullName?.split(' ').map((n) => n[0]).join('').slice(0, 2) || '?'}
                </div>
                <div>
                  <p className="text-sm font-medium text-[var(--color-text-primary)]">{cp.patient.fullName || 'Unknown'}</p>
                  <p className="text-xs text-[var(--color-text-muted)]">
                    {cp.patient.gender || ''}{cp.patient.birthDate ? ` • ${cp.patient.birthDate}` : ''}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ backgroundColor: consentDotColor[cp.consent.status] || 'var(--color-text-muted)' }}
                  aria-hidden
                />
                <ChevronRight size={14} className="text-[var(--color-text-muted)] transition-transform group-hover:translate-x-0.5" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </Card>
  )
})
