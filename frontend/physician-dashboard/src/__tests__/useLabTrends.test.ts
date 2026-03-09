import { vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

const mockGetLabTrends = vi.fn()

vi.mock('@medilink/shared', () => ({
  fhirAPI: {
    getLabTrends: (...args: unknown[]) => mockGetLabTrends(...args),
  },
}))

import { useLabTrends } from '@/hooks/useLabTrends'

const mockLabData = {
  resourceType: 'Bundle' as const,
  type: 'searchset' as const,
  total: 2,
  entry: [
    {
      resource: {
        resourceType: 'Observation',
        id: 'obs-1',
        meta: { versionId: '1', lastUpdated: '2024-01-15T00:00:00Z' },
        status: 'final',
        code: { coding: [{ system: 'http://loinc.org', code: '2339-0', display: 'Glucose' }] },
        subject: { reference: 'Patient/patient-1' },
        effectiveDateTime: '2024-01-15',
        valueQuantity: { value: 95, unit: 'mg/dL' },
      },
    },
    {
      resource: {
        resourceType: 'Observation',
        id: 'obs-2',
        meta: { versionId: '1', lastUpdated: '2024-02-15T00:00:00Z' },
        status: 'final',
        code: { coding: [{ system: 'http://loinc.org', code: '2339-0', display: 'Glucose' }] },
        subject: { reference: 'Patient/patient-1' },
        effectiveDateTime: '2024-02-15',
        valueQuantity: { value: 102, unit: 'mg/dL' },
      },
    },
  ],
}

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: qc }, children)
  }
  return Wrapper
}

describe('useLabTrends', () => {
  beforeEach(() => {
    mockGetLabTrends.mockReset()
  })

  it('fetches lab trends for given patient and LOINC code', async () => {
    mockGetLabTrends.mockResolvedValue({ data: mockLabData })

    const { result } = renderHook(() => useLabTrends('patient-1', '2339-0'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetLabTrends).toHaveBeenCalledWith('patient-1', '2339-0')
    expect(result.current.data).toEqual(mockLabData)
  })

  it('starts in loading state', () => {
    mockGetLabTrends.mockReturnValue(new Promise(() => {})) // never resolves

    const { result } = renderHook(() => useLabTrends('patient-1', '2339-0'), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
    expect(result.current.data).toBeUndefined()
  })

  it('handles API errors', async () => {
    mockGetLabTrends.mockRejectedValue(new Error('Network error'))

    const { result } = renderHook(() => useLabTrends('patient-1', '2339-0'), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBeDefined()
  })

  it('is disabled when loincCode is empty', () => {
    const { result } = renderHook(() => useLabTrends('patient-1', ''), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe('idle')
    expect(mockGetLabTrends).not.toHaveBeenCalled()
  })

  it('uses correct query key', async () => {
    mockGetLabTrends.mockResolvedValue({ data: mockLabData })

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: qc }, children)

    renderHook(() => useLabTrends('patient-1', '2339-0'), { wrapper })

    await waitFor(() => {
      const cache = qc.getQueryCache().findAll()
      const keys = cache.map((q) => q.queryKey)
      expect(keys).toContainEqual(['patient', 'patient-1', 'labs', '2339-0'])
    })
  })
})
