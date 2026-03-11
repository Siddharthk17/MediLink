'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Skeleton'
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'
import { consentAPI, formatDate } from '@medilink/shared'
import type { ConsentedPatient } from '@medilink/shared'
import Link from 'next/link'
import toast from 'react-hot-toast'

const statusVariant: Record<string, 'success' | 'warning' | 'danger' | 'muted'> = {
  active: 'success',
  expired: 'warning',
  revoked: 'danger',
}

export default function ConsentsPage() {
  const queryClient = useQueryClient()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['consents'],
    queryFn: async () => {
      const res = await consentAPI.getMyPatients()
      return res.data
    },
    refetchInterval: 30_000,
  })

  const revokeMutation = useMutation({
    mutationFn: (consentId: string) => consentAPI.revokeConsent(consentId),
    onSuccess: () => {
      toast.success('Consent revoked successfully')
      queryClient.invalidateQueries({ queryKey: ['consents'] })
      queryClient.invalidateQueries({ queryKey: ['consent', 'my-patients'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'my-patients'] })
    },
    onError: () => {
      toast.error('Failed to revoke consent')
    },
  })

  const patients: ConsentedPatient[] = data?.patients || []

  return (
    <PageWrapper title="Consent Management" subtitle="View and manage patient consent status">
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10" />)}
        </div>
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load consent data. Please try again later.
          </p>
        </div>
      ) : patients.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>No consented patients found</p>
        </div>
      ) : (
        <div className="glass-card rounded-card border border-[var(--color-border)] shadow-card overflow-hidden">
          <Table>
            <TableHeader>
              <TableHead>Patient</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Granted</TableHead>
              <TableHead>Expires</TableHead>
              <TableHead>Scopes</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableHeader>
            <TableBody>
              {patients.map((entry) => (
                <TableRow key={entry.patient.fhirId} className="group">
                  <TableCell>
                    <Link href={`/patients/${entry.patient.fhirId}`} className="block">
                      <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                        {entry.patient.fullName || 'Unknown Patient'}
                      </span>
                      <span className="block font-mono text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
                        {entry.patient.fhirId}
                      </span>
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[entry.consent.status] || 'muted'} size="sm">
                      {entry.consent.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {formatDate(entry.consent.grantedAt)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {entry.consent.expiresAt ? formatDate(entry.consent.expiresAt) : '—'}
                    </span>
                  </TableCell>
                  <TableCell>
                    {entry.consent.scope.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {entry.consent.scope.map((s) => (
                          <Badge key={s} variant="muted" size="sm">{s}</Badge>
                        ))}
                      </div>
                    ) : (
                      <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    {entry.consent.status === 'active' && (
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => revokeMutation.mutate(entry.consent.id)}
                        disabled={revokeMutation.isPending}
                      >
                        Revoke
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </PageWrapper>
  )
}
