'use client'

import { use } from 'react'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { DrugCheckPanel } from '@/components/clinical/DrugCheckPanel'
import { AllergyList } from '@/components/clinical/AllergyList'

export default function PrescribePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)

  return (
    <PageWrapper>
      <Link href={`/patients/${id}`} className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to patient
      </Link>
      <div className="mb-4">
        <AllergyList patientId={id} />
      </div>
      <DrugCheckPanel patientId={id} />
    </PageWrapper>
  )
}
