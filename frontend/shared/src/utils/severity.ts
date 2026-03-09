import type { InteractionSeverity } from '../types/fhir'
import type { DocumentJob, ConsentedPatient } from '../types/api'

export interface SeverityDisplay {
  label: string
  color: string
  bgColor: string
  borderColor: string
  icon: string
}

export const SEVERITY_MAP: Record<InteractionSeverity, SeverityDisplay> = {
  contraindicated: {
    label: 'Contraindicated',
    color: '#F43F5E',
    bgColor: 'var(--color-danger-subtle)',
    borderColor: 'rgba(244, 63, 94, 0.3)',
    icon: '⛔',
  },
  major: {
    label: 'Major',
    color: '#8B5CF6',
    bgColor: 'var(--color-info-subtle)',
    borderColor: 'rgba(139, 92, 246, 0.3)',
    icon: '⚠️',
  },
  moderate: {
    label: 'Moderate',
    color: '#F59E0B',
    bgColor: 'var(--color-warning-subtle)',
    borderColor: 'rgba(245, 158, 11, 0.3)',
    icon: '⚡',
  },
  minor: {
    label: 'Minor',
    color: '#10B981',
    bgColor: 'var(--color-success-subtle)',
    borderColor: 'rgba(16, 185, 129, 0.3)',
    icon: 'ℹ️',
  },
  unknown: {
    label: 'Unknown',
    color: '#71717A',
    bgColor: 'rgba(113, 113, 122, 0.08)',
    borderColor: 'rgba(113, 113, 122, 0.2)',
    icon: '❓',
  },
  none: {
    label: 'No Interactions',
    color: '#10B981',
    bgColor: 'var(--color-success-subtle)',
    borderColor: 'rgba(16, 185, 129, 0.3)',
    icon: '✅',
  },
}

export function getSeverityDisplay(severity: InteractionSeverity): SeverityDisplay {
  return SEVERITY_MAP[severity] ?? SEVERITY_MAP.unknown
}

const SEVERITY_ORDER: InteractionSeverity[] = [
  'none', 'minor', 'unknown', 'moderate', 'major', 'contraindicated',
]

export function getOverallSeverity(severities: InteractionSeverity[]): InteractionSeverity {
  if (severities.length === 0) return 'none'
  let highest: InteractionSeverity = 'none'
  for (const s of severities) {
    if (SEVERITY_ORDER.indexOf(s) > SEVERITY_ORDER.indexOf(highest)) {
      highest = s
    }
  }
  return highest
}

export interface ConsentDisplay {
  label: string
  color: string
  bgColor: string
}

export function getConsentDisplay(status: ConsentedPatient['consent']['status']): ConsentDisplay {
  switch (status) {
    case 'active':
      return { label: 'Active', color: '#10B981', bgColor: 'var(--color-success-subtle)' }
    case 'revoked':
      return { label: 'Revoked', color: '#F43F5E', bgColor: 'var(--color-danger-subtle)' }
    case 'expired':
      return { label: 'Expired', color: '#71717A', bgColor: 'rgba(113, 113, 122, 0.08)' }
    default:
      return { label: 'Unknown', color: '#71717A', bgColor: 'rgba(113, 113, 122, 0.08)' }
  }
}

export function getJobStatusDisplay(status: DocumentJob['status']): {
  label: string
  color: string
  bgColor: string
} {
  switch (status) {
    case 'pending':
      return { label: 'Pending', color: '#F59E0B', bgColor: 'var(--color-warning-subtle)' }
    case 'processing':
      return { label: 'Processing', color: '#3B82F6', bgColor: 'var(--color-info-subtle)' }
    case 'completed':
      return { label: 'Completed', color: '#10B981', bgColor: 'var(--color-success-subtle)' }
    case 'failed':
      return { label: 'Failed', color: '#F43F5E', bgColor: 'var(--color-danger-subtle)' }
    case 'needs-manual-review':
      return { label: 'Needs Review', color: '#8B5CF6', bgColor: 'var(--color-info-subtle)' }
    default:
      return { label: 'Unknown', color: '#71717A', bgColor: 'rgba(113, 113, 122, 0.08)' }
  }
}
