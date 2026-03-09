import { vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import type { DrugCheckResult } from '@medilink/shared'

const mockCheckDrugInteractions = vi.fn()
const mockAcknowledgeDrugInteraction = vi.fn()
const mockToastError = vi.fn()

vi.mock('@medilink/shared', () => ({
  clinicalAPI: {
    checkDrugInteractions: (...args: unknown[]) => mockCheckDrugInteractions(...args),
    acknowledgeDrugInteraction: (...args: unknown[]) => mockAcknowledgeDrugInteraction(...args),
  },
}))

vi.mock('react-hot-toast', () => ({
  default: { error: (...args: unknown[]) => mockToastError(...args) },
}))

import { useDrugCheck } from '@/hooks/useDrugCheck'

const mockResult: DrugCheckResult = {
  newMedication: { rxnormCode: 'RX123', name: 'Aspirin' },
  interactions: [
    {
      drugA: { rxnormCode: 'RX123', name: 'Aspirin' },
      drugB: { rxnormCode: 'RX456', name: 'Warfarin' },
      severity: 'major',
      description: 'Increased bleeding risk',
      source: 'drugbank',
      cached: false,
    },
  ],
  allergyConflicts: [],
  highestSeverity: 'major',
  hasContraindication: false,
  checkComplete: true,
}

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: qc }, children)
  }
  return Wrapper
}

describe('useDrugCheck', () => {
  beforeEach(() => {
    mockCheckDrugInteractions.mockReset()
    mockAcknowledgeDrugInteraction.mockReset()
    mockToastError.mockReset()
  })

  it('starts with null result and not checking', () => {
    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })
    expect(result.current.result).toBeNull()
    expect(result.current.isChecking).toBe(false)
    expect(result.current.isAcknowledging).toBe(false)
  })

  it('calls clinicalAPI.checkDrugInteractions with correct args', async () => {
    mockCheckDrugInteractions.mockResolvedValue({ data: mockResult })

    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    act(() => result.current.check('RX123'))

    await waitFor(() => {
      expect(result.current.isChecking).toBe(false)
    })

    expect(mockCheckDrugInteractions).toHaveBeenCalledWith('patient-1', 'RX123')
  })

  it('sets result on successful check', async () => {
    mockCheckDrugInteractions.mockResolvedValue({ data: mockResult })

    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    act(() => result.current.check('RX123'))

    await waitFor(() => {
      expect(result.current.result).toEqual(mockResult)
    })
  })

  it('shows toast error on check failure', async () => {
    mockCheckDrugInteractions.mockRejectedValue(new Error('API error'))

    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    act(() => result.current.check('RX123'))

    await waitFor(() => {
      expect(mockToastError).toHaveBeenCalledWith(
        'Drug check failed. You may proceed with caution.'
      )
    })
  })

  it('calls acknowledgeDrugInteraction with correct args', async () => {
    mockCheckDrugInteractions.mockResolvedValue({ data: mockResult })
    mockAcknowledgeDrugInteraction.mockResolvedValue({ data: {} })

    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    // First check to set the result
    act(() => result.current.check('RX123'))
    await waitFor(() => {
      expect(result.current.result).not.toBeNull()
    })

    // Then acknowledge
    await act(async () => {
      await result.current.acknowledge('Clinical necessity')
    })

    expect(mockAcknowledgeDrugInteraction).toHaveBeenCalledWith(
      'patient-1',
      'RX123',
      ['RX456'],
      'Clinical necessity'
    )
  })

  it('acknowledge throws when result is null', async () => {
    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    await expect(
      act(async () => {
        await result.current.acknowledge('Override reason')
      })
    ).rejects.toThrow('No drug check result to acknowledge')

    expect(mockAcknowledgeDrugInteraction).not.toHaveBeenCalled()
  })

  it('reset clears the result', async () => {
    mockCheckDrugInteractions.mockResolvedValue({ data: mockResult })

    const { result } = renderHook(() => useDrugCheck('patient-1'), {
      wrapper: createWrapper(),
    })

    act(() => result.current.check('RX123'))
    await waitFor(() => {
      expect(result.current.result).toEqual(mockResult)
    })

    act(() => result.current.reset())
    expect(result.current.result).toBeNull()
  })
})
