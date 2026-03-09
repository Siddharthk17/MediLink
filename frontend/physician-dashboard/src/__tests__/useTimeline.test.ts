import { vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

const mockGetTimeline = vi.fn()

vi.mock('@medilink/shared', () => ({
  fhirAPI: {
    getTimeline: (...args: unknown[]) => mockGetTimeline(...args),
  },
}))

import { useTimeline } from '@/hooks/useTimeline'

const mockTimelineData = {
  resourceType: 'Bundle' as const,
  type: 'searchset' as const,
  total: 1,
  entry: [
    {
      resource: {
        resourceType: 'Encounter',
        id: 'enc-1',
        meta: { versionId: '1', lastUpdated: '2024-01-10T00:00:00Z' },
        status: 'finished',
        class: { code: 'AMB', display: 'ambulatory' },
        subject: { reference: 'Patient/patient-1' },
        period: { start: '2024-01-10T09:00:00Z', end: '2024-01-10T10:00:00Z' },
      },
    },
  ],
}

function createWrapper(qc?: QueryClient) {
  const client =
    qc ?? new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client }, children)
  }
  return Wrapper
}

describe('useTimeline', () => {
  beforeEach(() => {
    mockGetTimeline.mockReset()
  })

  it('fetches timeline for a patient', async () => {
    mockGetTimeline.mockResolvedValue({ data: mockTimelineData })

    const { result } = renderHook(() => useTimeline('patient-1'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetTimeline).toHaveBeenCalledWith('patient-1', {})
    expect(result.current.data).toEqual(mockTimelineData)
  })

  it('passes resourceType as _type param when provided', async () => {
    mockGetTimeline.mockResolvedValue({ data: mockTimelineData })

    const { result } = renderHook(() => useTimeline('patient-1', 'Encounter'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetTimeline).toHaveBeenCalledWith('patient-1', { _type: 'Encounter' })
  })

  it('starts in loading state', () => {
    mockGetTimeline.mockReturnValue(new Promise(() => {}))

    const { result } = renderHook(() => useTimeline('patient-1'), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
    expect(result.current.data).toBeUndefined()
  })

  it('handles API errors', async () => {
    mockGetTimeline.mockRejectedValue(new Error('Server error'))

    const { result } = renderHook(() => useTimeline('patient-1'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBeDefined()
  })

  it('uses query key with "all" when no resourceType provided', async () => {
    mockGetTimeline.mockResolvedValue({ data: mockTimelineData })

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const wrapper = createWrapper(qc)

    renderHook(() => useTimeline('patient-1'), { wrapper })

    await waitFor(() => {
      const keys = qc.getQueryCache().findAll().map((q) => q.queryKey)
      expect(keys).toContainEqual(['patient', 'patient-1', 'timeline', 'all'])
    })
  })

  it('uses query key with resourceType when provided', async () => {
    mockGetTimeline.mockResolvedValue({ data: mockTimelineData })

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const wrapper = createWrapper(qc)

    renderHook(() => useTimeline('patient-1', 'Condition'), { wrapper })

    await waitFor(() => {
      const keys = qc.getQueryCache().findAll().map((q) => q.queryKey)
      expect(keys).toContainEqual(['patient', 'patient-1', 'timeline', 'Condition'])
    })
  })

  it('does not pass _type param when resourceType is undefined', async () => {
    mockGetTimeline.mockResolvedValue({ data: mockTimelineData })

    renderHook(() => useTimeline('patient-1'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(mockGetTimeline).toHaveBeenCalled()
    })

    const params = mockGetTimeline.mock.calls[0][1]
    expect(params).not.toHaveProperty('_type')
  })
})
