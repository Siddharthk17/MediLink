'use client'

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import { consentAPI } from '@medilink/shared'
import type { ConsentedPatient } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { PatientSearch } from '@/components/patients/PatientSearch'
import { Skeleton } from '@/components/ui/Skeleton'
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'

const FILTERS = ['All', 'Active consent', 'Expiring (≤7d)', 'Revoked']

function matchesFilter(cp: ConsentedPatient, filter: string): boolean {
  if (filter === 'All') return true
  if (filter === 'Active consent') return cp.consent.status === 'active'
  if (filter === 'Expiring (≤7d)') {
    if (cp.consent.status !== 'active' || !cp.consent.expiresAt) return false
    const days = Math.ceil((new Date(cp.consent.expiresAt).getTime() - Date.now()) / 86_400_000)
    return days <= 7 && days > 0
  }
  if (filter === 'Revoked') return cp.consent.status === 'revoked'
  return true
}

const statusDot: Record<string, string> = {
  active: 'var(--color-success)',
  revoked: 'var(--color-danger)',
  expired: 'var(--color-warning)',
}

export default function PatientsPage() {
  const [search, setSearch] = useState('')
  const [activeFilter, setActiveFilter] = useState('All')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['consent', 'my-patients'],
    queryFn: async () => {
      const res = await consentAPI.getMyPatients()
      return res.data.patients
    },
  })

  const patients = useMemo(() => data || [], [data])

  const filtered = useMemo(() => {
    return patients
      .filter((cp) => matchesFilter(cp, activeFilter))
      .filter((cp) =>
        search ? (cp.patient.fullName || '').toLowerCase().includes(search.toLowerCase()) : true
      )
  }, [patients, activeFilter, search])

  return (
    <PageWrapper title="Patients" subtitle={`${filtered.length} patients with active consent`}>
      <PatientSearch
        onSearch={setSearch}
        total={patients.length}
        filters={FILTERS}
        activeFilter={activeFilter}
        onFilterChange={setActiveFilter}
      />

      {isLoading ? (
        <div className="mt-6 space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-10" />
          ))}
        </div>
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load patients. Please try again later.
          </p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {search ? `No patients matching "${search}"` : 'No patients found'}
          </p>
        </div>
      ) : (
        <div className="mt-6 glass-card rounded-card border border-[var(--color-border)] shadow-card overflow-hidden">
          <Table>
            <TableHeader>
              <TableHead>Patient</TableHead>
              <TableHead>FHIR ID</TableHead>
              <TableHead>Consent</TableHead>
              <TableHead className="w-[120px] text-right pr-12">Actions</TableHead>
            </TableHeader>
            <TableBody>
              {filtered.map((cp) => {
                const age = cp.patient.birthDate
                  ? new Date().getFullYear() - parseInt(cp.patient.birthDate.slice(0, 4), 10)
                  : null
                const gender = cp.patient.gender
                  ? cp.patient.gender.charAt(0).toUpperCase() + cp.patient.gender.slice(1)
                  : null

                return (
                  <TableRow key={cp.patient.id} className="group">
                    <TableCell>
                      <div>
                        <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                          {cp.patient.fullName || 'Unknown'}
                        </span>
                        <span className="block text-xs" style={{ color: 'var(--color-text-muted)' }}>
                          {[gender, age ? `${age}y` : null].filter(Boolean).join(' · ')}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                        {cp.patient.fhirId}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="inline-flex items-center gap-1.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                        <span
                          className="h-1.5 w-1.5 rounded-full shrink-0"
                          style={{ backgroundColor: statusDot[cp.consent.status] || 'var(--color-text-muted)' }}
                        />
                        {cp.consent.status}
                      </span>
                    </TableCell>
                    <TableCell className="w-[120px] text-right pr-12">
                      <Link
                        href={`/patients/${cp.patient.fhirId}`}
                        className="text-xs font-medium transition-colors"
                        style={{ color: 'var(--color-text-accent)' }}
                      >
                        View
                      </Link>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </PageWrapper>
  )
}
