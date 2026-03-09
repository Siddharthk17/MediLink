'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'
import { apiClient, formatDateTime } from '@medilink/shared'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/authStore'

interface AuditLog {
  id: string
  userId: string
  userRole: string
  userEmail: string
  action: string
  resourceType: string
  resourceId: string
  purpose: string
  success: boolean
  statusCode: number
  ipAddress: string
  createdAt: string
}

export default function AuditLogsPage() {
  const [page, setPage] = useState(1)
  const { user } = useAuthStore()
  const router = useRouter()

  const PAGE_SIZE = 25
  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin', 'audit-logs', page],
    queryFn: async () => {
      const res = await apiClient.get<{ entries: AuditLog[]; total: number }>('/admin/audit-logs', {
        params: { _count: PAGE_SIZE, _offset: (page - 1) * PAGE_SIZE },
      })
      return res.data
    },
    enabled: user?.role === 'admin',
  })

  const logs = data?.entries || []
  const total = data?.total || 0

  const getActionColor = (action: string): 'success' | 'danger' | 'warning' | 'info' | 'muted' => {
    if (action.startsWith('create')) return 'success'
    if (action.startsWith('delete')) return 'danger'
    if (action.startsWith('update')) return 'warning'
    if (action.startsWith('read') || action.startsWith('search')) return 'info'
    return 'muted'
  }

  if (user && user.role !== 'admin') {
    router.replace('/dashboard')
    return null
  }

  return (
    <PageWrapper title="Audit Logs" subtitle={`${total} total entries`}>
      <Link href="/admin" className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to admin
      </Link>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 10 }).map((_, i) => <Skeleton key={i} className="h-12" />)}
        </div>
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load audit logs. Please try again later.
          </p>
        </div>
      ) : logs.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            No audit log entries found.
          </p>
        </div>
      ) : (
        <>
          <Card padding="sm">
            <Table>
              <TableHeader>
                <TableHead>Timestamp</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource</TableHead>
                <TableHead>IP Address</TableHead>
              </TableHeader>
              <TableBody>
                {logs.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="font-mono text-xs whitespace-nowrap">
                      {formatDateTime(log.createdAt)}
                    </TableCell>
                    <TableCell className="text-xs">
                      {log.userEmail || log.userRole || '—'}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getActionColor(log.action)} size="sm">
                        {log.action}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span className="text-xs font-mono">{log.resourceType}/{log.resourceId}</span>
                    </TableCell>
                    <TableCell className="text-xs font-mono">{log.ipAddress || '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>

          <div className="flex items-center justify-between mt-4">
            <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              Page {page} of {Math.ceil(total / 25) || 1}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1.5 text-xs font-medium rounded border border-[var(--color-border)] disabled:opacity-40 hover:bg-[var(--color-bg-elevated)] transition-colors"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= Math.ceil(total / 25)}
                className="px-3 py-1.5 text-xs font-medium rounded border border-[var(--color-border)] disabled:opacity-40 hover:bg-[var(--color-bg-elevated)] transition-colors"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                Next
              </button>
            </div>
          </div>
        </>
      )}
    </PageWrapper>
  )
}
