'use client'

import { useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useDropzone } from 'react-dropzone'
import { Upload, FileText, RefreshCw, ExternalLink } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { apiClient, getJobStatusDisplay, formatRelative } from '@medilink/shared'
import type { DocumentJob } from '@medilink/shared'
import { useDocumentJobs } from '@/hooks/useDocumentJobs'
import { cn } from '@/lib/utils'
import toast from 'react-hot-toast'
import Link from 'next/link'

function statusToBadgeVariant(status: string): 'success' | 'warning' | 'danger' | 'info' | 'muted' | 'accent' {
  switch (status) {
    case 'completed': return 'success'
    case 'processing': return 'warning'
    case 'failed': return 'danger'
    case 'queued': case 'uploaded': return 'info'
    default: return 'muted'
  }
}

interface DocumentUploadProps {
  patientId: string
}

const ACCEPTED_TYPES = {
  'application/pdf': ['.pdf'],
  'image/jpeg': ['.jpg', '.jpeg'],
  'image/png': ['.png'],
  'image/webp': ['.webp'],
}
const MAX_SIZE = 20 * 1024 * 1024

export function DocumentUpload({ patientId }: DocumentUploadProps) {
  const queryClient = useQueryClient()
  const { jobs, startPolling, setJobs } = useDocumentJobs(patientId)

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    for (const file of acceptedFiles) {
      const formData = new FormData()
      formData.append('file', file)
      const toastId = toast.loading(`Uploading ${file.name}...`)
      try {
        const res = await apiClient.post<{ jobId: string; status: string; estimatedProcessingTime: string; message: string }>(
          `/documents/upload?patientId=${patientId}`, formData,
          { headers: { 'Content-Type': 'multipart/form-data' } }
        )
        toast.dismiss(toastId)
        toast.success(`${file.name} uploaded successfully`)
        queryClient.invalidateQueries({ queryKey: ['dashboard', 'document-jobs'] })
        queryClient.invalidateQueries({ queryKey: ['patient', patientId] })
        const newJob: DocumentJob = {
          jobId: res.data.jobId,
          status: res.data.status as DocumentJob['status'],
          uploadedAt: new Date().toISOString(),
          estimatedProcessingTime: res.data.estimatedProcessingTime,
        }
        setJobs((prev) => [newJob, ...prev])
        startPolling()
      } catch {
        toast.dismiss(toastId)
        toast.error(`Failed to upload ${file.name}`)
      }
    }
  }, [patientId, queryClient, setJobs, startPolling])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: ACCEPTED_TYPES,
    maxSize: MAX_SIZE,
    onDropRejected: (rejections) => {
      for (const r of rejections) {
        if (r.errors[0]?.code === 'file-too-large') {
          toast.error(`${r.file.name} exceeds 20MB limit`)
        } else {
          toast.error(`${r.file.name}: ${r.errors[0]?.message || 'Invalid file type'}`)
        }
      }
    },
  })

  return (
    <div className="space-y-6">
      {/* Dropzone */}
      <div
        {...getRootProps()}
        className={cn(
          'border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-all',
          isDragActive
            ? 'border-[var(--color-accent)] bg-[var(--color-accent-subtle)]'
            : 'border-[var(--color-border)] hover:border-[var(--color-accent)] hover:bg-[var(--color-accent-subtle)]'
        )}
      >
        <input {...getInputProps()} />
        <Upload size={32} className="mx-auto mb-3" style={{ color: isDragActive ? 'var(--color-accent)' : 'var(--color-text-muted)' }} />
        <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
          {isDragActive ? 'Drop files here' : 'Drop lab report here'}
        </p>
        <p className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
          Supports PDF, JPG, PNG, WEBP — max 20MB
        </p>
      </div>

      {/* Job tracker */}
      {jobs.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-sm font-semibold" style={{ color: 'var(--color-text-primary)' }}>
            Uploaded Reports
          </h3>
          {jobs.map((job) => {
            const statusDisplay = getJobStatusDisplay(job.status)
            return (
              <Card key={job.jobId} padding="sm">
                <div className="flex items-start gap-3">
                  <FileText size={20} style={{ color: 'var(--color-accent)' }} className="mt-0.5 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-sm font-medium truncate" style={{ color: 'var(--color-text-primary)' }}>
                        Report {job.jobId.slice(0, 8)}
                      </p>
                      <Badge
                        variant={statusToBadgeVariant(job.status)}
                        dot={job.status === 'processing'}
                        size="sm"
                      >
                        {statusDisplay.label}
                      </Badge>
                    </div>
                    {job.status === 'completed' && (
                      <p className="text-xs mt-1" style={{ color: 'var(--color-success)' }}>
                        ✓ {job.observationsCreated || 0} observations extracted, {job.loincMapped || 0} LOINC codes mapped
                      </p>
                    )}
                    {job.status === 'failed' && (
                      <p className="text-xs mt-1" style={{ color: 'var(--color-danger)' }}>
                        {job.errorMessage || 'Processing failed'}
                      </p>
                    )}
                    <p className="text-[10px] mt-1" style={{ color: 'var(--color-text-muted)' }}>
                      Uploaded {formatRelative(job.uploadedAt)}
                    </p>
                  </div>
                </div>
                {job.status === 'completed' && job.fhirReportId && (
                  <div className="mt-2 ml-8">
                    <Link href={`/patients/${patientId}/labs`}>
                      <Button variant="ghost" size="sm">
                        <ExternalLink size={12} /> View DiagnosticReport
                      </Button>
                    </Link>
                  </div>
                )}
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
