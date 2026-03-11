'use client'

import { useQuery } from '@tanstack/react-query'
import { fhirAPI } from '@medilink/shared'

export function useLabTrends(patientId: string, loincCode: string) {
  return useQuery({
    queryKey: ['patient', patientId, 'labs', loincCode],
    queryFn: async () => {
      const res = await fhirAPI.getLabTrends(patientId, loincCode)
      return res.data
    },
    enabled: !!loincCode,
    refetchInterval: 120_000,
  })
}
