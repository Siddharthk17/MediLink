'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Skeleton } from '@/components/ui/Skeleton'
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'
import { apiClient, formatDate } from '@medilink/shared'
import { Search, ArrowLeft } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/authStore'
import toast from 'react-hot-toast'

interface UserRecord {
  id: string
  role: string
  status: string
  fullName: string
  emailHash: string
  createdAt: string
  lastLoginAt?: string
  fhirPatientId?: string
  totpEnabled: boolean
}

export default function AdminUsersPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const { user } = useAuthStore()
  const router = useRouter()
  const queryClient = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin', 'users'],
    queryFn: async () => {
      const res = await apiClient.get<{ users: UserRecord[]; total: number }>('/admin/users')
      return res.data.users ?? []
    },
    enabled: user?.role === 'admin',
  })

  const approveMutation = useMutation({
    mutationFn: async (userId: string) => {
      await apiClient.post(`/admin/physicians/${userId}/approve`)
    },
    onSuccess: () => {
      toast.success('User approved successfully')
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] })
    },
    onError: () => {
      toast.error('Failed to approve user')
    },
  })

  if (user && user.role !== 'admin') {
    router.replace('/dashboard')
    return null
  }

  const users = data || []
  const filtered = searchQuery
    ? users.filter(
        (u) =>
          (u.fullName || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
          (u.role || '').toLowerCase().includes(searchQuery.toLowerCase())
      )
    : users

  return (
    <PageWrapper title="User Management" subtitle="Manage system users and roles">
      <Link href="/admin" className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to admin
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <div className="flex-1">
          <Input
            placeholder="Search users..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            icon={<Search size={16} />}
          />
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-16" />)}
        </div>
      ) : isError ? (
        <div className="text-center py-16">
          <p className="text-sm" style={{ color: 'var(--color-danger)' }}>
            Failed to load users. Please try again later.
          </p>
        </div>
      ) : (
        <Card padding="sm">
          <Table>
            <TableHeader>
              <TableHead>Name</TableHead>
              <TableHead>Email Hash</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Last Login</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableHeader>
            <TableBody>
              {filtered.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium text-[var(--color-text-primary)]">
                    {u.fullName || '—'}
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {u.emailHash.slice(0, 12)}…
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={u.role === 'admin' ? 'accent' : u.role === 'physician' ? 'info' : 'muted'}
                      size="sm"
                    >
                      {u.role}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.status === 'active' ? 'success' : u.status === 'pending' ? 'warning' : 'danger'} size="sm" dot>
                      {u.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{formatDate(u.createdAt)}</TableCell>
                  <TableCell>{u.lastLoginAt ? formatDate(u.lastLoginAt) : '—'}</TableCell>
                  <TableCell className="text-right">
                    {u.status === 'pending' && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => approveMutation.mutate(u.id)}
                        disabled={approveMutation.isPending}
                      >
                        Approve
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {filtered.length === 0 && (
                <TableRow>
                  <TableCell className="text-center py-8">
                    <p className="text-sm text-[var(--color-text-muted)]">No users found</p>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </Card>
      )}
    </PageWrapper>
  )
}
