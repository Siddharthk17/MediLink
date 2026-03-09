'use client'

import { Badge } from '@/components/ui/Badge'
import { getConsentDisplay } from '@medilink/shared'

interface ConsentStatusBadgeProps {
  status: 'active' | 'revoked' | 'expired'
  expiresAt?: string
}

export function ConsentStatusBadge({ status, expiresAt }: ConsentStatusBadgeProps) {
  const display = getConsentDisplay(status)

  const badgeVariant: 'success' | 'warning' | 'danger' | 'info' | 'muted' = (() => {
    switch (status) {
      case 'active': return 'success'
      case 'revoked': return 'danger'
      case 'expired': return 'muted'
      default: return 'muted'
    }
  })()

  return (
    <Badge variant={badgeVariant} dot={status === 'active'} size="sm">
      {display.label}
    </Badge>
  )
}
