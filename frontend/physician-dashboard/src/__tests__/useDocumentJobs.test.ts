import { vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import type { DocumentJob } from '@medilink/shared'

const mockGet = vi.fn()

vi.mock('@medilink/shared', () => ({
  apiClient: { get: (...args: unknown[]) => mockGet(...args) },
}))

import { useDocumentJobs } from '@/hooks/useDocumentJobs'

const makeJob = (overrides: Partial<DocumentJob> = {}): DocumentJob => ({
  jobId: 'job-1',
  status: 'processing',
  uploadedAt: '2024-01-01T00:00:00Z',
  ...overrides,
})

describe('useDocumentJobs', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGet.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('starts with empty jobs array', () => {
    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    expect(result.current.jobs).toEqual([])
  })

  it('exposes startPolling, stopPolling, setJobs', () => {
    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    expect(typeof result.current.startPolling).toBe('function')
    expect(typeof result.current.stopPolling).toBe('function')
    expect(typeof result.current.setJobs).toBe('function')
  })

  it('polls the API on the correct interval', async () => {
    const jobs = [makeJob()]
    mockGet.mockResolvedValue({ data: { jobs } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))

    act(() => result.current.startPolling())

    // Advance past first poll interval
    await act(async () => {
      vi.advanceTimersByTime(3000)
    })

    expect(mockGet).toHaveBeenCalledWith('/documents/jobs?patientId=patient-1')
  })

  it('updates jobs state after successful poll', async () => {
    const jobs = [makeJob()]
    mockGet.mockResolvedValue({ data: { jobs } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))

    act(() => result.current.startPolling())
    await act(async () => {
      vi.advanceTimersByTime(3000)
    })

    expect(result.current.jobs).toEqual(jobs)
  })

  it('stops polling when all jobs reach terminal status', async () => {
    const completedJobs = [
      makeJob({ jobId: 'j1', status: 'completed' }),
      makeJob({ jobId: 'j2', status: 'failed' }),
    ]
    mockGet.mockResolvedValue({ data: { jobs: completedJobs } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })

    mockGet.mockClear()

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })

    // No additional calls because polling stopped
    expect(mockGet).not.toHaveBeenCalled()
  })

  it('continues polling when jobs are not all terminal', async () => {
    mockGet
      .mockResolvedValueOnce({ data: { jobs: [makeJob({ status: 'processing' })] } })
      .mockResolvedValueOnce({ data: { jobs: [makeJob({ status: 'completed' })] } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(mockGet).toHaveBeenCalledTimes(1)

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(mockGet).toHaveBeenCalledTimes(2)
  })

  it('stopPolling prevents further API calls', async () => {
    mockGet.mockResolvedValue({ data: { jobs: [makeJob()] } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(mockGet).toHaveBeenCalledTimes(1)

    act(() => result.current.stopPolling())
    mockGet.mockClear()

    await act(async () => {
      vi.advanceTimersByTime(6000)
    })
    expect(mockGet).not.toHaveBeenCalled()
  })

  it('does not start duplicate polling intervals', async () => {
    mockGet.mockResolvedValue({ data: { jobs: [makeJob()] } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())
    act(() => result.current.startPolling()) // second call should be no-op

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('handles API errors silently and continues polling', async () => {
    mockGet
      .mockRejectedValueOnce(new Error('Network error'))
      .mockResolvedValueOnce({ data: { jobs: [makeJob({ status: 'completed' })] } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    // Jobs should still be empty after error
    expect(result.current.jobs).toEqual([])

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(result.current.jobs).toEqual([makeJob({ status: 'completed' })])
  })

  it('setJobs allows manual job state update', () => {
    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    const manualJobs = [makeJob({ jobId: 'manual-1' })]

    act(() => result.current.setJobs(manualJobs))
    expect(result.current.jobs).toEqual(manualJobs)
  })

  it('cleans up polling on unmount', async () => {
    mockGet.mockResolvedValue({ data: { jobs: [makeJob()] } })

    const { result, unmount } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    unmount()
    mockGet.mockClear()

    await act(async () => {
      vi.advanceTimersByTime(6000)
    })
    expect(mockGet).not.toHaveBeenCalled()
  })

  it('recognizes needs-manual-review as a terminal status', async () => {
    const jobs = [makeJob({ status: 'needs-manual-review' })]
    mockGet.mockResolvedValue({ data: { jobs } })

    const { result } = renderHook(() => useDocumentJobs('patient-1'))
    act(() => result.current.startPolling())

    await act(async () => {
      vi.advanceTimersByTime(3000)
    })

    mockGet.mockClear()
    await act(async () => {
      vi.advanceTimersByTime(3000)
    })
    expect(mockGet).not.toHaveBeenCalled()
  })
})
