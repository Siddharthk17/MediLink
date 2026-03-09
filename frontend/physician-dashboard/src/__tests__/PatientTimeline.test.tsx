import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'

const mockGetTimeline = vi.fn()

vi.mock('@medilink/shared', () => ({
  fhirAPI: {
    getTimeline: (...args: any[]) => mockGetTimeline(...args),
  },
  getCodeDisplay: vi.fn((codeableConcept: any) => {
    if (!codeableConcept) return 'Unknown'
    return codeableConcept?.coding?.[0]?.display || codeableConcept?.text || 'Unknown'
  }),
  formatRelative: vi.fn((date: string) => 'recently'),
}))

import { PatientTimeline } from '@/components/patients/PatientTimeline'

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

const mockEntries = [
  {
    resource: {
      resourceType: 'Encounter',
      id: 'enc-1',
      class: { display: 'Outpatient' },
      period: { start: '2024-01-15T10:00:00Z' },
    },
  },
  {
    resource: {
      resourceType: 'Condition',
      id: 'cond-1',
      code: { coding: [{ display: 'Hypertension' }] },
      onsetDateTime: '2024-01-10T00:00:00Z',
    },
  },
]

describe('PatientTimeline', () => {
  beforeEach(() => {
    mockGetTimeline.mockReset()
  })

  it('renders all resource filter buttons', async () => {
    mockGetTimeline.mockResolvedValue({ data: { entry: [] } })
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    const expectedFilters = ['All', 'Encounters', 'Conditions', 'Medications', 'Labs', 'Reports', 'Allergies', 'Immunizations']
    for (const filter of expectedFilters) {
      expect(screen.getByText(filter)).toBeInTheDocument()
    }
  })

  it('shows loading skeletons while fetching', () => {
    mockGetTimeline.mockReturnValue(new Promise(() => {})) // never resolves
    const { container } = render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    const skeletons = container.querySelectorAll('.skeleton-shimmer')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it('shows empty state when no entries exist', async () => {
    mockGetTimeline.mockResolvedValue({ data: { entry: [] } })
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    expect(await screen.findByText('No records found for this patient')).toBeInTheDocument()
  })

  it('shows error state when query fails', async () => {
    mockGetTimeline.mockRejectedValue(new Error('Network error'))
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    expect(await screen.findByText('Failed to load timeline.')).toBeInTheDocument()
  })

  it('renders timeline entries when data is loaded', async () => {
    mockGetTimeline.mockResolvedValue({ data: { entry: mockEntries } })
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    expect(await screen.findByText('Encounter')).toBeInTheDocument()
    expect(screen.getByText('Condition')).toBeInTheDocument()
  })

  it('changes filter when clicking a filter button', async () => {
    mockGetTimeline.mockResolvedValue({ data: { entry: [] } })
    const user = userEvent.setup()
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    await screen.findByText('No records found for this patient')

    await user.click(screen.getByText('Encounters'))

    // The Encounters button should now have the active style
    const encountersBtn = screen.getByText('Encounters')
    expect(encountersBtn.className).toContain('text-[var(--color-accent)]')
  })

  it('All filter is active by default', () => {
    mockGetTimeline.mockReturnValue(new Promise(() => {}))
    render(<PatientTimeline patientId="p1" />, { wrapper: createWrapper() })
    const allBtn = screen.getByText('All')
    expect(allBtn.className).toContain('text-[var(--color-accent)]')
  })

  it('calls fhirAPI.getTimeline with the patient ID', async () => {
    mockGetTimeline.mockResolvedValue({ data: { entry: [] } })
    render(<PatientTimeline patientId="patient-abc" />, { wrapper: createWrapper() })
    await screen.findByText('No records found for this patient')
    expect(mockGetTimeline).toHaveBeenCalledWith('patient-abc', expect.any(Object))
  })
})
