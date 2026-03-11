'use client'

import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { FileText, Download, Clock, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { fhirAPI, formatDate, formatRelative } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function DocumentsPage() {
  const { user } = useAuthStore()
  const fhirId = user?.fhirPatientId

  const { data, isLoading } = useQuery({
    queryKey: ['fhir', 'DiagnosticReport'],
    queryFn: () => fhirAPI.searchResources('DiagnosticReport', { patient: `Patient/${fhirId}` }),
    enabled: !!fhirId,
    refetchInterval: 120_000,
  })

  const reports = data?.data?.entry ?? []

  return (
    <PageWrapper title="Documents" subtitle="Your diagnostic reports and medical documents">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-card" />
          ))}
        </div>
      ) : reports.length === 0 ? (
        <Card variant="glass" className="text-center py-16">
          <FileText className="w-10 h-10 mx-auto mb-3 text-[var(--color-text-muted)]" />
          <p className="text-[var(--color-text-muted)]">No documents on file</p>
        </Card>
      ) : (
        <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-4">
          {reports.map((entry, i) => {
            const report = entry.resource as Record<string, any>
            const title = report.code?.coding?.[0]?.display || report.code?.text || 'Diagnostic Report'
            const status = report.status
            const date = report.effectiveDateTime || report.issued
            const category = report.category?.[0]?.coding?.[0]?.display
            const conclusion = report.conclusion

            return (
              <motion.div key={report.id ?? `doc-${i}`} variants={staggerItem}>
                <Card className="hover:border-[var(--color-border-hover)] transition-colors">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-[var(--color-type-diagnostic-subtle)] flex items-center justify-center shrink-0">
                      <FileText className="w-5 h-5 text-[var(--color-type-diagnostic)]" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="font-medium text-[var(--color-text-primary)]">{title}</h3>
                        <Badge variant={
                          status === 'final' ? 'success' :
                          status === 'preliminary' ? 'warning' :
                          status === 'cancelled' ? 'danger' : 'default'
                        }>
                          {status === 'final' && <CheckCircle className="w-3 h-3 mr-1" />}
                          {status === 'cancelled' && <XCircle className="w-3 h-3 mr-1" />}
                          {status === 'preliminary' && <Loader2 className="w-3 h-3 mr-1" />}
                          {status}
                        </Badge>
                      </div>
                      {conclusion && (
                        <p className="mt-1 text-sm text-[var(--color-text-secondary)] line-clamp-2">
                          {conclusion}
                        </p>
                      )}
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--color-text-muted)]">
                        {category && <span>{category}</span>}
                        {date && (
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" /> {formatDate(date)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </Card>
              </motion.div>
            )
          })}
        </motion.div>
      )}
    </PageWrapper>
  )
}
