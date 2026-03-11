'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Bell, Mail, Smartphone } from 'lucide-react'
import toast from 'react-hot-toast'
import { notificationsAPI } from '@medilink/shared'
import type { NotificationPreferences } from '@medilink/shared'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { Card } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Skeleton'
import { cn } from '@/lib/utils'

interface PreferenceField {
  key: keyof NotificationPreferences
  label: string
  description: string
  locked: boolean
}

const EMAIL_PREFS: PreferenceField[] = [
  { key: 'emailBreakGlass', label: 'Emergency access alerts', description: 'Email when a provider uses break-glass access on your records', locked: true },
  { key: 'emailAccountLocked', label: 'Account locked alerts', description: 'Email when your account is locked due to failed login attempts', locked: true },
  { key: 'emailDocumentComplete', label: 'Document processing complete', description: 'Email when uploaded documents finish processing', locked: false },
  { key: 'emailDocumentFailed', label: 'Document processing failed', description: 'Email when document processing encounters errors', locked: false },
  { key: 'emailConsentGranted', label: 'Consent granted', description: 'Email when a provider is granted access to your records', locked: false },
  { key: 'emailConsentRevoked', label: 'Consent revoked', description: 'Email when access to your records is revoked', locked: false },
]

const PUSH_PREFS: PreferenceField[] = [
  { key: 'pushEnabled', label: 'Enable push notifications', description: 'Master toggle for all push notifications', locked: false },
  { key: 'pushNewPrescription', label: 'New prescriptions', description: 'Push notification when a new medication is prescribed', locked: false },
  { key: 'pushLabResultReady', label: 'Lab results ready', description: 'Push notification when lab results become available', locked: false },
  { key: 'pushConsentRequest', label: 'Consent requests', description: 'Push notification when a provider requests access to your records', locked: false },
  { key: 'pushCriticalLab', label: 'Critical lab alerts', description: 'Urgent alerts for critical lab values', locked: false },
]

export default function NotificationsPage() {
  const queryClient = useQueryClient()

  const { data: preferences, isLoading } = useQuery({
    queryKey: ['notification-preferences'],
    queryFn: async () => {
      const res = await notificationsAPI.getPreferences()
      return res.data
    },
  })

  const updateMutation = useMutation({
    mutationFn: (prefs: Partial<NotificationPreferences>) =>
      notificationsAPI.updatePreferences(prefs),
    onMutate: async (newPrefs) => {
      await queryClient.cancelQueries({ queryKey: ['notification-preferences'] })
      const previous = queryClient.getQueryData<NotificationPreferences>(['notification-preferences'])
      if (previous) {
        queryClient.setQueryData<NotificationPreferences>(
          ['notification-preferences'],
          { ...previous, ...newPrefs }
        )
      }
      return { previous }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-preferences'] })
      toast.success('Preferences updated')
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['notification-preferences'], context.previous)
      }
      toast.error('Failed to update preferences')
    },
  })

  const togglePreference = (key: keyof NotificationPreferences) => {
    if (!preferences) return
    const current = preferences[key]
    updateMutation.mutate({ [key]: !current })
  }

  const renderToggle = (field: PreferenceField) => {
    const enabled = preferences ? Boolean(preferences[field.key]) : false
    return (
      <div key={field.key} className="flex items-center justify-between py-3">
        <div>
          <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
            {field.label}
          </p>
          <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
            {field.description}
            {field.locked && (
              <span className="ml-1 text-[10px] font-medium" style={{ color: 'var(--color-warning)' }}>
                (required)
              </span>
            )}
          </p>
        </div>
        <button
          onClick={() => togglePreference(field.key)}
          disabled={field.locked || updateMutation.isPending}
          className={cn(
            'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
            enabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border)]',
            (field.locked || updateMutation.isPending) && 'opacity-60 cursor-not-allowed'
          )}
          role="switch"
          aria-checked={enabled}
          aria-label={`Toggle ${field.label}`}
        >
          <span
            className={cn(
              'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
              enabled ? 'translate-x-6' : 'translate-x-1'
            )}
          />
        </button>
      </div>
    )
  }

  return (
    <PageWrapper title="Notifications" subtitle="Manage your notification preferences">
      <div className="glass-card rounded-card border border-[var(--color-border)] shadow-card overflow-hidden mb-8">
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Bell className="w-10 h-10 text-[var(--color-text-muted)] mb-4" />
          <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>No notifications</p>
          <p className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
            Notifications about lab results, consents, and prescriptions will appear here
          </p>
        </div>
      </div>

      <Card padding="lg" className="mb-6">
        <h2 className="font-display text-xl mb-1" style={{ color: 'var(--color-text-primary)' }}>
          Email Preferences
        </h2>
        <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>
          Manage which notifications are sent to your email
        </p>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-12" />)}
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border-subtle)]">
            {EMAIL_PREFS.map(renderToggle)}
          </div>
        )}
      </Card>

      <Card padding="lg">
        <h2 className="font-display text-xl mb-1" style={{ color: 'var(--color-text-primary)' }}>
          Push Notifications
        </h2>
        <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>
          Configure push notification delivery
        </p>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-12" />)}
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border-subtle)]">
            {PUSH_PREFS.map(renderToggle)}
          </div>
        )}
      </Card>
    </PageWrapper>
  )
}
