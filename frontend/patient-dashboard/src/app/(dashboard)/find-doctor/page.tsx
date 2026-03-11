'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { Search, UserPlus, Stethoscope, Clock, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { doctorsAPI, consentAPI } from '@medilink/shared'
import type { DoctorSummary } from '@medilink/shared'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { Spinner } from '@/components/ui/Spinner'
import { PageWrapper } from '@/components/layout/PageWrapper'
import toast from 'react-hot-toast'

type RequestStatus = 'none' | 'pending' | 'active' | 'rejected'

export default function FindDoctorPage() {
  const queryClient = useQueryClient()
  const [searchTerm, setSearchTerm] = useState('')

  const { data: doctorsData, isLoading: loadingDoctors } = useQuery({
    queryKey: ['doctors'],
    queryFn: () => doctorsAPI.list(),
    refetchInterval: 60_000,
  })

  const { data: grantsData } = useQuery({
    queryKey: ['my-grants'],
    queryFn: () => consentAPI.getMyGrants(),
    refetchInterval: 30_000,
  })

  const requestMutation = useMutation({
    mutationFn: (doctorId: string) =>
      consentAPI.grantConsent({ providerId: doctorId, scope: ['*'], purpose: 'treatment' }),
    onSuccess: () => {
      toast.success('Consultation request sent!')
      queryClient.invalidateQueries({ queryKey: ['my-grants'] })
      queryClient.invalidateQueries({ queryKey: ['consents'] })
    },
    onError: () => {
      toast.error('Failed to send request. You may already have a pending request.')
    },
  })

  const doctors = doctorsData?.data?.doctors ?? []
  const grants = grantsData?.data?.consents ?? []

  const getRequestStatus = (doctorId: string): RequestStatus => {
    const grant = grants.find((g) => g.providerId === doctorId)
    if (!grant) return 'none'
    const s = grant.status
    if (s === 'pending' || s === 'active' || s === 'rejected') return s
    return 'none'
  }

  const filtered = doctors.filter((d: DoctorSummary) => {
    const term = searchTerm.toLowerCase()
    return (
      d.fullName.toLowerCase().includes(term) ||
      (d.specialization || '').toLowerCase().includes(term)
    )
  })

  const statusConfig: Record<RequestStatus, { label: string; color: string; icon: React.ReactNode }> = {
    none: { label: '', color: '', icon: null },
    pending: { label: 'Request Pending', color: 'warning', icon: <Clock className="w-3.5 h-3.5" /> },
    active: { label: 'Connected', color: 'success', icon: <CheckCircle className="w-3.5 h-3.5" /> },
    rejected: { label: 'Declined', color: 'danger', icon: <XCircle className="w-3.5 h-3.5" /> },
  }

  return (
    <PageWrapper title="Find a Doctor" subtitle="Browse available physicians and request a consultation">
      <div className="mb-8 relative max-w-lg">
        <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" />
        <input
          type="text"
          placeholder="Search by name or specialization…"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full pl-11 pr-4 py-3 rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-elevated)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] transition-all"
        />
      </div>

      {loadingDoctors ? (
        <div className="flex justify-center py-20">
          <Spinner size="lg" />
        </div>
      ) : filtered.length === 0 ? (
        <Card className="p-12 text-center">
          <Stethoscope className="w-12 h-12 mx-auto mb-4 text-[var(--color-text-muted)] opacity-40" />
          <p className="text-[var(--color-text-muted)]">
            {searchTerm ? 'No doctors match your search' : 'No doctors available at the moment'}
          </p>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <AnimatePresence mode="popLayout">
            {filtered.map((doctor: DoctorSummary, i: number) => {
              const status = getRequestStatus(doctor.id)
              const cfg = statusConfig[status]

              return (
                <motion.div
                  key={doctor.id}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                  transition={{ delay: i * 0.04, duration: 0.3 }}
                >
                  <Card className="p-6 flex flex-col h-full">
                    <div className="flex items-start gap-4 mb-4">
                      <div className="w-12 h-12 rounded-2xl flex items-center justify-center shrink-0" style={{ background: 'var(--color-accent-subtle)' }}>
                        <Stethoscope className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
                      </div>
                      <div className="min-w-0">
                        <h3 className="font-semibold text-[var(--color-text-primary)] truncate">
                          {doctor.fullName}
                        </h3>
                        <p className="text-sm text-[var(--color-text-muted)]">
                          {doctor.specialization || 'General Practice'}
                        </p>
                      </div>
                    </div>

                    {doctor.mciNumber && (
                      <p className="text-xs text-[var(--color-text-muted)] mb-4 font-mono">
                        MCI: {doctor.mciNumber}
                      </p>
                    )}

                    <div className="mt-auto pt-4 border-t border-[var(--color-border-subtle)]">
                      {status === 'none' ? (
                        <Button
                          onClick={() => requestMutation.mutate(doctor.id)}
                          disabled={requestMutation.isPending}
                          className="w-full"
                          size="sm"
                        >
                          {requestMutation.isPending ? (
                            <Loader2 className="w-4 h-4 animate-spin mr-1" />
                          ) : (
                            <UserPlus className="w-4 h-4 mr-1" />
                          )}
                          Request Consultation
                        </Button>
                      ) : (
                        <div className="flex items-center justify-center gap-2 py-1.5">
                          {cfg.icon}
                          <Badge variant={cfg.color as 'warning' | 'success' | 'danger'}>
                            {cfg.label}
                          </Badge>
                        </div>
                      )}
                    </div>
                  </Card>
                </motion.div>
              )
            })}
          </AnimatePresence>
        </div>
      )}
    </PageWrapper>
  )
}
