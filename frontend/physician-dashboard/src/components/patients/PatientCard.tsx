'use client'

import Link from 'next/link'
import { ConsentStatusBadge } from './ConsentStatusBadge'
import { Button } from '@/components/ui/Button'
import { getInitials } from '@medilink/shared'
import React from 'react'

interface PatientCardProps {
  patient: {
    id: string
    fhirId: string
    fullName: string
    gender?: string
    birthDate?: string
  }
  consent: {
    status: 'active' | 'revoked' | 'expired'
    expiresAt?: string
  }
  healthStatus?: 'normal' | 'abnormal' | 'critical'
}

export const PatientCard = React.memo(function PatientCard({ patient, consent }: PatientCardProps) {
  const age = patient.birthDate
    ? new Date().getFullYear() - parseInt(patient.birthDate.slice(0, 4), 10)
    : null

  return (
    <div
      className="rounded-[var(--radius)] border border-[var(--color-border)] p-4 transition-colors hover:bg-[var(--color-bg-hover)] hover:border-[var(--color-border-hover)]"
      style={{ backgroundColor: 'var(--color-bg-card)' }}
    >
      <div className="flex items-start gap-3">
        <span className="text-xs font-medium shrink-0" style={{ color: 'var(--color-text-accent)' }}>
          {getInitials(patient.fullName)}
        </span>
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <h3 className="text-sm font-medium truncate" style={{ color: 'var(--color-text-primary)' }}>
              {patient.fullName}
            </h3>
            <ConsentStatusBadge status={consent.status} expiresAt={consent.expiresAt} />
          </div>
          <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
            {patient.gender ? patient.gender.charAt(0).toUpperCase() + patient.gender.slice(1) : '—'}
            {age ? ` · ${age} years` : ''}
          </p>
        </div>
      </div>

      <div className="flex gap-2 mt-3">
        <Link href={`/patients/${patient.fhirId}`} className="flex-1">
          <Button variant="ghost" size="sm" className="w-full">View</Button>
        </Link>
        <Link href={`/patients/${patient.fhirId}/prescribe`}>
          <Button variant="ghost" size="sm">Interactions</Button>
        </Link>
      </div>
    </div>
  )
})
