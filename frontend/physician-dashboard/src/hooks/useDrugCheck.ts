'use client'

import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { clinicalAPI } from '@medilink/shared'
import type { DrugCheckResult } from '@medilink/shared'
import toast from 'react-hot-toast'

export function useDrugCheck(patientId: string) {
  const [result, setResult] = useState<DrugCheckResult | null>(null)

  const checkMutation = useMutation({
    mutationFn: async (rxnormCode: string) => {
      const res = await clinicalAPI.checkDrugInteractions(patientId, rxnormCode)
      return res.data
    },
    onSuccess: setResult,
    onError: () => {
      toast.error('Drug check failed. You may proceed with caution.')
    },
  })

  const acknowledgeMutation = useMutation({
    mutationFn: async (reason: string) => {
      if (!result?.newMedication?.rxnormCode) {
        throw new Error('No drug check result to acknowledge')
      }
      const conflicting = result.interactions.map((i) => i.drugB.rxnormCode)
      await clinicalAPI.acknowledgeDrugInteraction(
        patientId,
        result.newMedication.rxnormCode,
        conflicting,
        reason
      )
    },
  })

  return {
    result,
    isChecking: checkMutation.isPending,
    check: checkMutation.mutate,
    acknowledge: acknowledgeMutation.mutateAsync,
    isAcknowledging: acknowledgeMutation.isPending,
    reset: () => setResult(null),
  }
}
