'use client'

import { useQuery } from '@tanstack/react-query'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { apiClient } from '@medilink/shared'
import type { AdminStats } from '@medilink/shared'
import { ChevronRight } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/authStore'

export default function AdminPage() {
  const { user } = useAuthStore()
  const router = useRouter()
  const { data: stats, isLoading, isError } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: async () => {
      const res = await apiClient.get<AdminStats>('/admin/stats')
      return res.data
    },
    enabled: user?.role === 'admin',
    refetchInterval: 120_000,
  })

  if (user && user.role !== 'admin') {
    router.replace('/dashboard')
    return null
  }

  const statCards = [
    { label: 'Total Users', value: stats?.users.total ?? 0 },
    { label: 'Physicians', value: stats?.users.physicians ?? 0 },
    { label: 'Patients', value: stats?.users.patients ?? 0 },
    { label: 'FHIR Resources', value: stats?.fhirResources.total ?? 0 },
    { label: 'Documents', value: stats?.documents.totalUploaded ?? 0 },
    { label: 'Active Consents', value: stats?.consents.activeGrants ?? 0 },
  ]

  const adminLinks = [
    { href: '/admin/users', label: 'User Management', description: 'Create, edit, and manage user accounts' },
    { href: '/admin/audit-logs', label: 'Audit Logs', description: 'View system audit trail' },
  ]

  return (
      <PageWrapper title="Admin Panel" subtitle="System administration and monitoring">
        {/* Stats grid */}
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
          {isLoading
            ? Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-24" />)
            : isError
              ? <div className="col-span-full text-center py-8">
                  <p className="text-sm" style={{ color: 'var(--color-danger)' }}>Failed to load admin stats.</p>
                </div>
              : statCards.map((stat) => {
              return (
                <Card key={stat.label} padding="md">
                  <div>
                    <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-[var(--color-text-muted)]">{stat.label}</p>
                    <p className="font-display text-3xl mt-2 text-[var(--color-text-primary)]">
                      {(stat.value as number).toLocaleString()}
                    </p>
                  </div>
                </Card>
              )
            })}
        </div>

        {/* Admin navigation */}
        <h2 className="font-display text-2xl mb-4" style={{ color: 'var(--color-text-primary)' }}>Quick Links</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {adminLinks.map((link) => {
            return (
              <Link key={link.href} href={link.href}>
                <Card padding="md" hover>
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>{link.label}</p>
                      <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{link.description}</p>
                    </div>
                    <ChevronRight size={16} className="text-[var(--color-text-muted)]" />
                  </div>
                </Card>
              </Link>
            )
        })}
      </div>
    </PageWrapper>
  )
}
