'use client'

import { useQuery } from '@tanstack/react-query'
import { fhirAPI, getCodeDisplay } from '@medilink/shared'
import type { CodeableConcept } from '@medilink/shared'
import { Badge } from '@/components/ui/Badge'
import { Shield } from 'lucide-react'

interface AllergyListProps {
  patientId: string
}

export function AllergyList({ patientId }: AllergyListProps) {
  const { data } = useQuery({
    queryKey: ['patient', patientId, 'allergies'],
    queryFn: async () => {
      const res = await fhirAPI.searchResources('AllergyIntolerance', { patient: `Patient/${patientId}` })
      return res.data
    },
    refetchInterval: 120_000,
  })

  const allergies = data?.entry?.map((e) => e.resource) || []

  if (allergies.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2">
      {allergies.map((allergy) => {
        const name = getCodeDisplay((allergy as { code?: CodeableConcept }).code)
        const criticality = (allergy as { criticality?: string }).criticality
        return (
          <Badge
            key={allergy.id}
            variant={criticality === 'high' ? 'danger' : 'warning'}
            size="md"
          >
            <Shield size={10} /> {name} ({criticality?.toUpperCase() || '—'})
          </Badge>
        )
      })}
    </div>
  )
}
