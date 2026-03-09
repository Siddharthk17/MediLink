'use client'

import { Calendar, Heart, Pill, Activity, FileText, Shield, Syringe } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { getCodeDisplay, formatRelative } from '@medilink/shared'
import { cn } from '@/lib/utils'
import React from 'react'

interface TimelineEventProps {
  resource: {
    resourceType: string
    id: string
    [key: string]: unknown
  }
  isLast?: boolean
}

const typeConfig: Record<string, { icon: React.ElementType; color: string; bg: string }> = {
  Encounter:           { icon: Calendar, color: 'var(--color-type-encounter)', bg: 'var(--color-type-encounter-subtle)' },
  Condition:           { icon: Heart, color: 'var(--color-type-condition)', bg: 'var(--color-type-condition-subtle)' },
  MedicationRequest:   { icon: Pill, color: 'var(--color-type-medication)', bg: 'var(--color-type-medication-subtle)' },
  Observation:         { icon: Activity, color: 'var(--color-type-observation)', bg: 'var(--color-type-observation-subtle)' },
  DiagnosticReport:    { icon: FileText, color: 'var(--color-type-diagnostic)', bg: 'var(--color-type-diagnostic-subtle)' },
  AllergyIntolerance:  { icon: Shield, color: 'var(--color-type-allergy)', bg: 'var(--color-type-allergy-subtle)' },
  Immunization:        { icon: Syringe, color: 'var(--color-type-medication)', bg: 'var(--color-type-medication-subtle)' },
}

function getEventDate(resource: Record<string, unknown>): string | undefined {
  return (resource.effectiveDateTime || resource.authoredOn || resource.onsetDateTime ||
    resource.occurrenceDateTime || resource.recordedDate ||
    (resource.period as { start?: string })?.start ||
    (resource.meta as { lastUpdated?: string })?.lastUpdated) as string | undefined
}

function getEventTitle(resource: Record<string, unknown>): string {
  const type = resource.resourceType as string
  if (type === 'Encounter') return `${(resource.class as { display?: string })?.display || 'Visit'} encounter`
  if (type === 'Condition') return getCodeDisplay(resource.code as any)
  if (type === 'MedicationRequest') return getCodeDisplay(resource.medicationCodeableConcept as any)
  if (type === 'Observation') return getCodeDisplay(resource.code as any)
  if (type === 'DiagnosticReport') return getCodeDisplay(resource.code as any)
  if (type === 'AllergyIntolerance') return getCodeDisplay(resource.code as any)
  if (type === 'Immunization') return getCodeDisplay(resource.vaccineCode as any)
  return type
}

export const TimelineEvent = React.memo(function TimelineEvent({ resource, isLast }: TimelineEventProps) {
  const config = typeConfig[resource.resourceType] || typeConfig.Encounter
  const Icon = config.icon
  const date = getEventDate(resource as Record<string, unknown>)
  const title = getEventTitle(resource as Record<string, unknown>)

  return (
    <div className="flex gap-4 relative">
      {/* Connector line */}
      {!isLast && (
        <div
          className="absolute left-[19px] top-[40px] bottom-0 w-[2px]"
          style={{ background: 'var(--color-border-subtle)' }}
        />
      )}

      {/* Icon */}
      <div
        className="w-10 h-10 rounded-full flex items-center justify-center shrink-0 z-10"
        style={{ background: config.bg, color: config.color }}
      >
        <Icon size={16} />
      </div>

      {/* Card */}
      <Card padding="sm" className="flex-1 mb-4">
        <div className="flex items-start justify-between gap-2">
          <Badge variant="muted" size="sm">{resource.resourceType}</Badge>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {date ? formatRelative(date) : '—'}
          </span>
        </div>
        <p className="font-display text-[17px] mt-1" style={{ color: 'var(--color-text-primary)' }}>
          {title}
        </p>
      </Card>
    </div>
  )
})
