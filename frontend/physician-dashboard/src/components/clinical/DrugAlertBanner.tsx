'use client'

import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { AlertTriangle, XCircle, AlertCircle, Info, CheckCircle, HelpCircle, ChevronDown, ChevronUp } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { getSeverityDisplay } from '@medilink/shared'
import type { DrugCheckResult } from '@medilink/shared'
import { alertPanelVariants } from '@/lib/motion'
import { cn } from '@/lib/utils'
import React from 'react'
import type { InteractionSeverity } from '@medilink/shared'

function severityToBadgeVariant(severity: InteractionSeverity): 'success' | 'warning' | 'danger' | 'info' | 'muted' {
  switch (severity) {
    case 'contraindicated': return 'danger'
    case 'major': return 'info'
    case 'moderate': return 'warning'
    case 'minor': return 'success'
    case 'none': return 'success'
    default: return 'muted'
  }
}

interface DrugAlertBannerProps {
  result: DrugCheckResult
  onAcknowledge: (reason: string) => Promise<void>
  onDismiss?: () => void
}

const severityIcons: Record<string, React.ElementType> = {
  XCircle, AlertTriangle, AlertCircle, Info, CheckCircle, HelpCircle,
}

export const DrugAlertBanner = React.memo(function DrugAlertBanner({ result, onAcknowledge }: DrugAlertBannerProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null)
  const [ackReason, setAckReason] = useState('')
  const [isAcknowledging, setIsAcknowledging] = useState(false)
  const [acknowledged, setAcknowledged] = useState(false)

  const severity = getSeverityDisplay(result.highestSeverity)
  const SeverityIcon = severityIcons[severity.icon] || AlertCircle
  const totalInteractions = result.interactions.length + result.allergyConflicts.length

  if (result.highestSeverity === 'none') return null

  const handleAcknowledge = async () => {
    setIsAcknowledging(true)
    try {
      await onAcknowledge(ackReason)
      setAcknowledged(true)
    } finally {
      setIsAcknowledging(false)
    }
  }

  return (
    <motion.div
      variants={alertPanelVariants}
      initial="hidden"
      animate="visible"
      aria-live="polite"
    >
      {/* Header */}
      <div
        className="p-4 rounded-t-card border"
        style={{ background: severity.bgColor, borderColor: severity.borderColor }}
      >
        <div className="flex items-center gap-3">
          <SeverityIcon size={24} style={{ color: severity.color }} />
          <div className="flex-1">
            <p className="font-semibold text-sm" style={{ color: severity.color }}>
              {totalInteractions} drug interaction{totalInteractions !== 1 ? 's' : ''} found
            </p>
          </div>
          <Badge variant={severityToBadgeVariant(result.highestSeverity)} size="md">{severity.label}</Badge>
        </div>
      </div>

      {/* Interaction cards */}
      <div className="border-x border-[var(--color-border)] divide-y divide-[var(--color-border-subtle)]">
        {result.interactions.map((interaction, i) => {
          const intSev = getSeverityDisplay(interaction.severity)
          const isExpanded = expandedIndex === i
          return (
            <div key={i} className="p-4 bg-[var(--color-bg-card)]">
              <button
                onClick={() => setExpandedIndex(isExpanded ? null : i)}
                className="flex items-center gap-2 w-full text-left"
              >
                <AlertTriangle size={16} style={{ color: intSev.color }} />
                <span className="flex-1 text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                  {interaction.drugB.name} ↔ {result.newMedication.name}
                </span>
                <Badge variant={severityToBadgeVariant(interaction.severity)} size="sm">{intSev.label}</Badge>
                {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
              </button>
              <p className="text-xs mt-1 ml-6" style={{ color: 'var(--color-text-muted)' }}>
                {interaction.description}
              </p>
              <AnimatePresence>
                {isExpanded && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    className="overflow-hidden ml-6 mt-2 space-y-1"
                  >
                    {interaction.mechanism && (
                      <p className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                        <strong>Mechanism:</strong> {interaction.mechanism}
                      </p>
                    )}
                    {interaction.clinicalEffect && (
                      <p className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                        <strong>Clinical effect:</strong> {interaction.clinicalEffect}
                      </p>
                    )}
                    {interaction.management && (
                      <p className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                        <strong>Management:</strong> {interaction.management}
                      </p>
                    )}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          )
        })}

        {/* Allergy conflicts */}
        {result.allergyConflicts.map((allergy, i) => (
          <div key={`allergy-${i}`} className="p-4 bg-[var(--color-bg-card)]" style={{ borderLeft: '3px solid var(--color-danger)' }}>
            <div className="flex items-center gap-2">
              <XCircle size={16} style={{ color: 'var(--color-danger)' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                Patient is allergic to {allergy.allergen.name}
              </span>
              <Badge variant="danger" size="sm">ALLERGY</Badge>
            </div>
          </div>
        ))}
      </div>

      {/* Acknowledgment form for contraindicated */}
      {result.highestSeverity === 'contraindicated' && !acknowledged && (
        <div className="p-4 border border-t-0 border-[var(--color-border)] rounded-b-card bg-[var(--color-bg-card)]">
          <p className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-primary)' }}>
            Clinical reason for overriding this contraindication:
          </p>
          <textarea
            value={ackReason}
            onChange={(e) => setAckReason(e.target.value)}
            className="w-full h-20 p-3 text-sm rounded-button resize-none bg-[var(--color-bg-surface)] border border-[var(--color-border)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-border-focus)] focus:outline-none"
            placeholder="Document your clinical reasoning (minimum 20 characters)..."
          />
          <div className="flex items-center justify-between mt-2">
            <span className="text-xs" style={{ color: ackReason.length >= 20 ? 'var(--color-success)' : 'var(--color-text-muted)' }}>
              {ackReason.length}/20 minimum
            </span>
            <Button
              variant="danger"
              size="sm"
              disabled={ackReason.length < 20}
              loading={isAcknowledging}
              onClick={handleAcknowledge}
            >
              Acknowledge Contraindication
            </Button>
          </div>
        </div>
      )}

      {acknowledged && (
        <div className="p-3 border border-t-0 border-[var(--color-border)] rounded-b-card bg-[var(--color-success-subtle)] text-center">
          <p className="text-sm font-medium" style={{ color: 'var(--color-success)' }}>
            ✓ Acknowledged — prescribe enabled for 30 minutes
          </p>
        </div>
      )}
    </motion.div>
  )
})
