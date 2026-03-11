'use client'

import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { User, Mail, Phone, Calendar, Lock, Save, Heart } from 'lucide-react'
import toast from 'react-hot-toast'
import { authAPI, fhirAPI, getPatientName, formatDate } from '@medilink/shared'
import type { Patient } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { staggerContainer, staggerItem } from '@/lib/motion'

export default function ProfilePage() {
  const { user } = useAuthStore()
  const queryClient = useQueryClient()
  const fhirId = user?.fhirPatientId

  const { data: meData, isLoading: loadingMe } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => authAPI.getMe(),
  })

  const { data: patientData, isLoading: loadingPatient } = useQuery({
    queryKey: ['fhir', 'Patient', fhirId],
    queryFn: () => fhirAPI.getResource('Patient', fhirId!),
    enabled: !!fhirId,
  })

  const me = meData?.data
  const patient = patientData?.data as Patient | undefined

  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  const [passwordError, setPasswordError] = useState('')

  const changePasswordMutation = useMutation({
    mutationFn: (data: { currentPassword: string; newPassword: string }) =>
      authAPI.changePassword(data),
    onSuccess: () => {
      toast.success('Password changed successfully')
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
    },
    onError: () => {
      toast.error('Failed to change password')
    },
  })

  const handleChangePassword = (e: React.FormEvent) => {
    e.preventDefault()
    setPasswordError('')

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordError('Passwords do not match')
      return
    }
    if (passwordForm.newPassword.length < 8) {
      setPasswordError('Password must be at least 8 characters')
      return
    }

    changePasswordMutation.mutate({
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword,
    })
  }

  const patientName = patient ? getPatientName(patient) : me?.fullName || 'Patient'
  const birthDate = patient?.birthDate
  const gender = patient?.gender

  return (
    <PageWrapper title="My Profile" subtitle="View and manage your account">
      <motion.div variants={staggerContainer} initial="initial" animate="animate" className="space-y-6">
        <motion.div variants={staggerItem}>
          <Card>
            {(loadingMe || loadingPatient) ? (
              <div className="space-y-4">
                <Skeleton className="h-16 w-16 rounded-full" />
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-4 w-64" />
              </div>
            ) : (
              <div className="flex items-start gap-5">
                <div className="w-16 h-16 rounded-full bg-[var(--color-accent)] flex items-center justify-center text-white text-2xl font-semibold shrink-0">
                  {patientName[0]?.toUpperCase() || 'P'}
                </div>
                <div className="flex-1">
                  <h2 className="text-xl font-semibold text-[var(--color-text-primary)]">{patientName}</h2>
                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                      <Mail className="w-4 h-4" />
                      <span>{me?.email || 'No email'}</span>
                    </div>
                    {me?.phone && (
                      <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                        <Phone className="w-4 h-4" />
                        <span>{me.phone}</span>
                      </div>
                    )}
                    {birthDate && (
                      <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                        <Calendar className="w-4 h-4" />
                        <span>{formatDate(birthDate)}</span>
                      </div>
                    )}
                    {gender && (
                      <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                        <User className="w-4 h-4" />
                        <span className="capitalize">{gender}</span>
                      </div>
                    )}
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Badge variant="accent">Patient</Badge>
                    <Badge variant={me?.status === 'active' ? 'success' : 'warning'}>
                      {me?.status || 'unknown'}
                    </Badge>
                    {me?.totpEnabled && <Badge variant="info">MFA Enabled</Badge>}
                  </div>
                </div>
              </div>
            )}
          </Card>
        </motion.div>

        <motion.div variants={staggerItem}>
          <Card>
            <div className="flex items-center gap-3 mb-5">
              <Lock className="w-5 h-5 text-[var(--color-accent)]" />
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Change Password</h2>
            </div>
            <form onSubmit={handleChangePassword} className="space-y-4">
              <Input
                id="currentPassword"
                type="password"
                label="Current Password"
                placeholder="Enter current password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, currentPassword: e.target.value }))}
                required
                autoComplete="current-password"
              />
              <Input
                id="newPassword"
                type="password"
                label="New Password"
                placeholder="Min 8 characters"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, newPassword: e.target.value }))}
                required
                autoComplete="new-password"
              />
              <Input
                id="confirmNewPassword"
                type="password"
                label="Confirm New Password"
                placeholder="Repeat new password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, confirmPassword: e.target.value }))}
                required
                autoComplete="new-password"
              />
              {passwordError && (
                <p className="text-sm text-[var(--color-danger)]">{passwordError}</p>
              )}
              <Button type="submit" disabled={changePasswordMutation.isPending}>
                <Save className="w-4 h-4" />
                {changePasswordMutation.isPending ? 'Changing…' : 'Change Password'}
              </Button>
            </form>
          </Card>
        </motion.div>

        <motion.div variants={staggerItem}>
          <Card variant="glass">
            <div className="flex items-center gap-3 mb-3">
              <Heart className="w-5 h-5 text-[var(--color-accent)]" />
              <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">About MediLink</h2>
            </div>
            <p className="text-sm text-[var(--color-text-muted)]">
              MediLink is a FHIR R4-compliant health platform that puts you in control of your medical data.
              Your records are encrypted and access is governed by your consent grants.
            </p>
            <p className="text-xs text-[var(--color-text-muted)] mt-3">
              Patient Dashboard v0.1.0
            </p>
          </Card>
        </motion.div>
      </motion.div>
    </PageWrapper>
  )
}
