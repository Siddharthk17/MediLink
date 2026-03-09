'use client'

import { useState, useRef, useCallback, useEffect } from 'react'
import { apiClient } from '@medilink/shared'
import type { DocumentJob } from '@medilink/shared'

const TERMINAL_STATUSES = ['completed', 'failed', 'needs-manual-review']
const POLL_INTERVAL_MS = 3000

export function useDocumentJobs(patientFhirId: string) {
  const [jobs, setJobs] = useState<DocumentJob[]>([])
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const startPolling = useCallback(() => {
    if (intervalRef.current) return
    intervalRef.current = setInterval(async () => {
      try {
        const res = await apiClient.get<{ jobs: DocumentJob[]; total: number }>(`/documents/jobs?patientId=${patientFhirId}`)
        const fetched = res.data.jobs || []
        setJobs(fetched)
        const allDone = fetched.every((j) => TERMINAL_STATUSES.includes(j.status))
        if (allDone && intervalRef.current) {
          clearInterval(intervalRef.current)
          intervalRef.current = null
        }
      } catch {
        // silently continue polling
      }
    }, POLL_INTERVAL_MS)
  }, [patientFhirId])

  const stopPolling = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  useEffect(() => () => stopPolling(), [stopPolling])

  return { jobs, startPolling, stopPolling, setJobs }
}
