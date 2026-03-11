'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { Eye, Clock } from 'lucide-react'
import { consentAPI, formatDateTime } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function AccessLogPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['consents', 'access-log'],
    queryFn: () => consentAPI.getAccessLog(),
    refetchInterval: 60_000,
  })

  const entries = data?.data?.entries ?? []

  return (
    <PageWrapper title="Access Log" subtitle="See who has accessed your health records">
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-card" />
          ))}
        </div>
      ) : entries.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <Eye className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No access records yet</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            When a provider accesses your records, it will appear here
          </p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-3">
          {entries.map((entry, i) => (
            <motion.div key={entry.id ?? `al-${i}`} variants={staggerItem}>
              <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-[var(--color-info-subtle)] flex items-center justify-center">
                      <Eye className="w-4 h-4 text-[var(--color-info)]" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-[var(--color-text-primary)]">
                        {entry.accessedByName || entry.accessedBy}
                      </p>
                      <p className="text-xs text-[var(--color-text-muted)]">
                        {entry.action} · {entry.resourceType}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="info">{entry.resourceType}</Badge>
                    <span className="text-xs text-[var(--color-text-muted)] flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {formatDateTime(entry.accessedAt)}
                    </span>
                  </div>
                </div>
              </Card>
            </motion.div>
          ))}
        </motion.div>
      )}
    </PageWrapper>
  )
}
