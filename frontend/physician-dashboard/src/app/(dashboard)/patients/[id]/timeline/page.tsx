'use client'

import { use } from 'react'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { PatientTimeline } from '@/components/patients/PatientTimeline'

export default function TimelinePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)

  return (
    <PageWrapper>
      <Link href={`/patients/${id}`} className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to patient
      </Link>
      <h1 className="font-display text-[28px] mb-6" style={{ color: 'var(--color-text-primary)' }}>
        Full Timeline
      </h1>
      <div className="max-w-[900px] mx-auto">
        <PatientTimeline patientId={id} />
      </div>
    </PageWrapper>
  )
}
