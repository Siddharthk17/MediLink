'use client'

import { useQuery } from '@tanstack/react-query'
import { fhirAPI } from '@medilink/shared'

export function useTimeline(patientId: string, resourceType?: string) {
  const params: Record<string, string> = {}
  if (resourceType) params._type = resourceType

  return useQuery({
    queryKey: ['patient', patientId, 'timeline', resourceType || 'all'],
    queryFn: async () => {
      const res = await fhirAPI.getTimeline(patientId, params)
      return res.data
    },
  })
}
