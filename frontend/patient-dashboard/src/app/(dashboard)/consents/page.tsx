'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Shield, ShieldOff, Calendar, Clock, User } from 'lucide-react'
import Link from 'next/link'
import toast from 'react-hot-toast'
import { consentAPI, formatDate, formatRelative } from '@medilink/shared'
import type { ConsentGrant } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function ConsentsPage() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['consents', 'my-grants'],
    queryFn: () => consentAPI.getMyGrants(),
    refetchInterval: 30_000,
  })

  const revokeMutation = useMutation({
    mutationFn: (consentId: string) => consentAPI.revokeConsent(consentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['consents'] })
      queryClient.invalidateQueries({ queryKey: ['my-grants'] })
      toast.success('Consent revoked successfully')
    },
    onError: () => {
      toast.error('Failed to revoke consent')
    },
  })

  const consents = data?.data?.consents ?? []
  const pendingConsents = consents.filter((c) => c.status === 'pending')
  const activeConsents = consents.filter((c) => c.status === 'active')
  const rejectedConsents = consents.filter((c) => c.status === 'rejected')
  const revokedConsents = consents.filter((c) => c.status === 'revoked')

  return (
    <PageWrapper
      title="My Consents"
      subtitle="Manage who can access your health records"
      actions={
        <Link href="/consents/access-log">
          <Button variant="secondary" size="sm">
            <Clock className="w-3.5 h-3.5" /> Access Log
          </Button>
        </Link>
      }
    >
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-card" />
          ))}
        </div>
      ) : consents.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Shield className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No consent grants found</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            Your physician will request access to your records
          </p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-8">
          {pendingConsents.length > 0 && (
            <div>
              <h2 className="text-sm font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
                Pending ({pendingConsents.length})
              </h2>
              <div className="space-y-3">
                {pendingConsents.map((consent) => (
                  <ConsentCard
                    key={consent.id}
                    consent={consent}
                    onRevoke={() => revokeMutation.mutate(consent.id)}
                    revoking={revokeMutation.isPending}
                    revokeLabel="Cancel Request"
                  />
                ))}
              </div>
            </div>
          )}

          {activeConsents.length > 0 && (
            <div>
              <h2 className="text-sm font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
                Active ({activeConsents.length})
              </h2>
              <div className="space-y-3">
                {activeConsents.map((consent) => (
                  <ConsentCard
                    key={consent.id}
                    consent={consent}
                    onRevoke={() => revokeMutation.mutate(consent.id)}
                    revoking={revokeMutation.isPending}
                  />
                ))}
              </div>
            </div>
          )}

          {rejectedConsents.length > 0 && (
            <div>
              <h2 className="text-sm font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
                Declined ({rejectedConsents.length})
              </h2>
              <div className="space-y-3">
                {rejectedConsents.map((consent) => (
                  <ConsentCard key={consent.id} consent={consent} />
                ))}
              </div>
            </div>
          )}

          {revokedConsents.length > 0 && (
            <div>
              <h2 className="text-sm font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
                Revoked ({revokedConsents.length})
              </h2>
              <div className="space-y-3">
                {revokedConsents.map((consent) => (
                  <ConsentCard key={consent.id} consent={consent} />
                ))}
              </div>
            </div>
          )}
        </motion.div>
      )}
    </PageWrapper>
  )
}

function ConsentCard({ consent, onRevoke, revoking, revokeLabel }: {
  consent: ConsentGrant
  onRevoke?: () => void
  revoking?: boolean
  revokeLabel?: string
}) {
  const isActive = consent.status === 'active'
  const isPending = consent.status === 'pending'
  const isRejected = consent.status === 'rejected'

  const badgeVariant = isActive ? 'success' : isPending ? 'warning' : isRejected ? 'danger' : 'default' as const
  const iconBg = isActive
    ? 'bg-[var(--color-success-subtle)]'
    : isPending
    ? 'bg-[var(--color-warning-subtle,rgba(234,179,8,0.1))]'
    : 'bg-[var(--color-bg-elevated)]'

  return (
    <motion.div variants={staggerItem}>
      <Card className={`hover:border-[var(--color-border-hover)] transition-colors ${!isActive && !isPending ? 'opacity-60' : ''}`}>
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${iconBg}`}>
              {isActive
                ? <Shield className="w-5 h-5 text-[var(--color-success)]" />
                : isPending
                ? <Clock className="w-5 h-5 text-[var(--color-warning,#eab308)]" />
                : <ShieldOff className="w-5 h-5 text-[var(--color-text-muted)]" />
              }
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-medium text-[var(--color-text-primary)]">
                  {consent.providerName || `Provider ${consent.providerId.slice(0, 8)}…`}
                </h3>
                <Badge variant={badgeVariant}>
                  {isPending ? 'Awaiting approval' : isRejected ? 'Declined' : consent.status}
                </Badge>
              </div>
              <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                <span className="flex items-center gap-1">
                  <Calendar className="w-3 h-3" /> {isPending ? 'Requested' : 'Granted'}: {formatDate(consent.grantedAt)}
                </span>
                {consent.expiresAt && (
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" /> Expires: {formatRelative(consent.expiresAt)}
                  </span>
                )}
                {consent.purpose && <span>Purpose: {consent.purpose}</span>}
              </div>
              {consent.scope.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {consent.scope.map((s) => (
                    <Badge key={s} variant="accent">{s}</Badge>
                  ))}
                </div>
              )}
            </div>
          </div>

          {(isActive || isPending) && onRevoke && (
            <Button
              variant={isPending ? 'secondary' : 'danger'}
              size="sm"
              onClick={onRevoke}
              disabled={revoking}
            >
              <ShieldOff className="w-3.5 h-3.5" />
              {revoking ? (isPending ? 'Cancelling…' : 'Revoking…') : (revokeLabel || 'Revoke')}
            </Button>
          )}
        </div>
      </Card>
    </motion.div>
  )
}
