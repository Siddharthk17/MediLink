'use client'

import { useQuery } from '@tanstack/react-query'
import { formatRelative, consentAPI } from '@medilink/shared'
import type { ConsentedPatient } from '@medilink/shared'
import React from 'react'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { Activity } from 'lucide-react'

export const RecentActivity = React.memo(function RecentActivity() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard', 'recent-consents'],
    queryFn: async () => {
      const res = await consentAPI.getMyPatients()
      return res.data
    },
    staleTime: 60_000,
    refetchInterval: 30_000,
  })

  const patients: ConsentedPatient[] = data?.patients || []

  return (
    <Card padding="lg" className="h-full">
      <h2 className="font-display text-2xl text-[var(--color-text-primary)] mb-6">Recent Consent Activity</h2>
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-10" />)}
        </div>
      ) : patients.length === 0 ? (
        <div className="text-center py-8">
          <Activity className="w-8 h-8 text-[var(--color-text-muted)] mx-auto mb-2" />
          <p className="text-sm text-[var(--color-text-muted)]">No recent consent activity</p>
        </div>
      ) : (
        <div className="space-y-4">
          {patients.map((cp, i) => (
            <div key={cp.consent.id} className="flex gap-3 relative">
              {i < patients.length - 1 && (
                <span className="absolute left-[3px] top-2.5 h-[calc(100%+8px)] w-px bg-[var(--color-border-subtle)]" />
              )}
              <span className="mt-2 h-2 w-2 rounded-full bg-[var(--color-accent)] shrink-0" />
              <div className="min-w-0">
                <p className="text-[13px] leading-snug text-[var(--color-text-secondary)]">
                  Consent {cp.consent.status} for {cp.patient.fullName || 'Unknown'}
                  {cp.consent.scope.includes('*') ? ' (full access)' : ` (${cp.consent.scope.join(', ')})`}
                </p>
                <p className="font-mono text-[10px] mt-1 text-[var(--color-text-muted)]">
                  {formatRelative(cp.consent.grantedAt)}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  )
})
