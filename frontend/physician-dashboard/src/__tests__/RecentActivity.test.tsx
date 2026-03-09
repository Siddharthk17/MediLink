import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@medilink/shared', () => ({
  formatRelative: vi.fn(() => 'some time ago'),
  consentAPI: {
    getMyPatients: vi.fn().mockResolvedValue({
      data: {
        patients: [
          {
            patient: { id: '1', fhirId: 'f1', fullName: 'Meera Sharma', gender: 'Female', birthDate: '1990-01-01' },
            consent: { id: 'c1', status: 'active', scope: ['*'], grantedAt: '2025-01-01T00:00:00Z' },
          },
          {
            patient: { id: '2', fhirId: 'f2', fullName: 'Rahul Kumar', gender: 'Male', birthDate: '1985-05-15' },
            consent: { id: 'c2', status: 'revoked', scope: ['read', 'write'], grantedAt: '2024-06-01T00:00:00Z' },
          },
        ],
        total: 2,
      },
    }),
  },
}))

import { RecentActivity } from '@/components/dashboard/RecentActivity'

function renderWithQuery(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

describe('RecentActivity', () => {
  it('renders the heading', () => {
    renderWithQuery(<RecentActivity />)
    expect(screen.getByText('Recent Consent Activity')).toBeInTheDocument()
  })

  it('renders consent activity items after loading', async () => {
    renderWithQuery(<RecentActivity />)
    expect(await screen.findByText(/Consent active for Meera Sharma/)).toBeInTheDocument()
    expect(screen.getByText(/Consent revoked for Rahul Kumar/)).toBeInTheDocument()
  })

  it('renders formatted timestamps for each item', async () => {
    renderWithQuery(<RecentActivity />)
    await screen.findByText(/Meera Sharma/)
    const timestamps = screen.getAllByText('some time ago')
    expect(timestamps).toHaveLength(2)
  })

  it('renders timeline dots for each item', async () => {
    const { container } = renderWithQuery(<RecentActivity />)
    await screen.findByText(/Meera Sharma/)
    const dots = container.querySelectorAll('.rounded-full.bg-\\[var\\(--color-accent\\)\\]')
    expect(dots).toHaveLength(2)
  })

  it('renders connector line for all but the last item', async () => {
    const { container } = renderWithQuery(<RecentActivity />)
    await screen.findByText(/Meera Sharma/)
    const connectors = container.querySelectorAll('.absolute.left-\\[3px\\]')
    expect(connectors).toHaveLength(1)
  })

  it('heading is an h2 element', () => {
    renderWithQuery(<RecentActivity />)
    const heading = screen.getByText('Recent Consent Activity')
    expect(heading.tagName).toBe('H2')
  })
})
