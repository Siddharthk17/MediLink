'use client'

import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { User, Mail, Lock, Save, Sparkles, Shield } from 'lucide-react'
import toast from 'react-hot-toast'
import { authAPI } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { Badge } from '@/components/ui/Badge'

export default function ProfilePage() {
  const { user } = useAuthStore()

  const { data: meData, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => authAPI.getMe(),
  })

  const me = meData?.data

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

  return (
    <PageWrapper title="My Profile" subtitle="View and manage your account">
      <div className="space-y-6">
        <Card padding="lg">
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-16 w-16 rounded-full" />
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-4 w-64" />
            </div>
          ) : (
            <div className="flex items-start gap-5">
              <div className="w-16 h-16 rounded-full bg-[var(--color-accent)] flex items-center justify-center text-white text-2xl font-semibold shrink-0">
                {me?.fullName?.[0]?.toUpperCase() || 'D'}
              </div>
              <div className="flex-1">
                <h2 className="text-xl font-semibold text-[var(--color-text-primary)]">{me?.fullName || 'Physician'}</h2>
                <div className="mt-3 space-y-2">
                  <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                    <Mail className="w-4 h-4" />
                    <span>{me?.email || 'No email'}</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
                    <User className="w-4 h-4" />
                    <span className="capitalize">{me?.role || 'physician'}</span>
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Badge variant="accent">
                    {me?.role === 'admin' ? 'Administrator' : 'Physician'}
                  </Badge>
                  <Badge variant={me?.status === 'active' ? 'success' : 'warning'}>
                    {me?.status || 'unknown'}
                  </Badge>
                  {me?.totpEnabled && <Badge variant="info">MFA Enabled</Badge>}
                </div>
              </div>
            </div>
          )}
        </Card>

        <Card padding="lg">
          <div className="flex items-center gap-3 mb-5">
            <Lock className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <h2 className="font-display text-xl" style={{ color: 'var(--color-text-primary)' }}>Change Password</h2>
          </div>
          <form onSubmit={handleChangePassword} className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="currentPwd" className="block text-sm font-medium text-[var(--color-text-secondary)]">Current Password</label>
              <input
                id="currentPwd"
                type="password"
                placeholder="Enter current password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, currentPassword: e.target.value }))}
                required
                autoComplete="current-password"
                className="w-full h-11 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] transition-colors duration-200 hover:border-[var(--color-border-hover)] focus:outline-none focus:border-[var(--color-border-focus)] focus:ring-2 focus:ring-[var(--color-accent-subtle)]"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="newPwd" className="block text-sm font-medium text-[var(--color-text-secondary)]">New Password</label>
              <input
                id="newPwd"
                type="password"
                placeholder="Min 8 characters"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, newPassword: e.target.value }))}
                required
                autoComplete="new-password"
                className="w-full h-11 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] transition-colors duration-200 hover:border-[var(--color-border-hover)] focus:outline-none focus:border-[var(--color-border-focus)] focus:ring-2 focus:ring-[var(--color-accent-subtle)]"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="confirmPwd" className="block text-sm font-medium text-[var(--color-text-secondary)]">Confirm New Password</label>
              <input
                id="confirmPwd"
                type="password"
                placeholder="Repeat new password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm((p) => ({ ...p, confirmPassword: e.target.value }))}
                required
                autoComplete="new-password"
                className="w-full h-11 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] transition-colors duration-200 hover:border-[var(--color-border-hover)] focus:outline-none focus:border-[var(--color-border-focus)] focus:ring-2 focus:ring-[var(--color-accent-subtle)]"
              />
            </div>
            {passwordError && (
              <p className="text-sm text-[var(--color-danger)]">{passwordError}</p>
            )}
            <button
              type="submit"
              disabled={changePasswordMutation.isPending}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {changePasswordMutation.isPending ? 'Changing…' : 'Change Password'}
            </button>
          </form>
        </Card>

        <Card padding="lg">
          <div className="flex items-center gap-3 mb-3">
            <Sparkles className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <h2 className="font-display text-xl" style={{ color: 'var(--color-text-primary)' }}>About MediLink</h2>
          </div>
          <p className="text-sm text-[var(--color-text-muted)]">
            MediLink is a FHIR R4-compliant health platform designed for physicians to manage patient records,
            prescriptions, and lab results with end-to-end encryption and consent-based access control.
          </p>
          <p className="text-xs text-[var(--color-text-muted)] mt-3">
            Physician Dashboard v0.1.0
          </p>
        </Card>
      </div>
    </PageWrapper>
  )
}
