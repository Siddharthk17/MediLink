'use client'

import { use } from 'react'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { PageWrapper } from '@/components/layout/PageWrapper'
import { DocumentUpload } from '@/components/documents/DocumentUpload'

export default function DocumentsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)

  return (
    <PageWrapper title="Documents & Reports">
      <Link href={`/patients/${id}`} className="inline-flex items-center gap-1 text-sm mb-4 hover:underline" style={{ color: 'var(--color-text-muted)' }}>
        <ArrowLeft size={14} /> Back to patient
      </Link>
      <DocumentUpload patientId={id} />
    </PageWrapper>
  )
}
