import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}))

vi.mock('@medilink/shared', () => ({
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
            consent: { id: 'c2', status: 'expired', scope: ['read'], grantedAt: '2024-06-01T00:00:00Z' },
          },
        ],
        total: 2,
      },
    }),
  },
}))

import { TodaysPatients } from '@/components/dashboard/TodaysPatients'

function renderWithQuery(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

describe('TodaysPatients', () => {
  it('renders the heading', () => {
    renderWithQuery(<TodaysPatients />)
    expect(screen.getByText('My Patients')).toBeInTheDocument()
  })

  it('renders the View all link with correct href', () => {
    renderWithQuery(<TodaysPatients />)
    const viewAll = screen.getByText('View all')
    expect(viewAll.closest('a')).toHaveAttribute('href', '/patients')
  })

  it('renders patient names after loading', async () => {
    renderWithQuery(<TodaysPatients />)
    expect(await screen.findByText('Meera Sharma')).toBeInTheDocument()
    expect(screen.getByText('Rahul Kumar')).toBeInTheDocument()
  })

  it('renders patient links to correct patient pages', async () => {
    renderWithQuery(<TodaysPatients />)
    await screen.findByText('Meera Sharma')
    const links = screen.getAllByRole('link')
    const patientLinks = links.filter((l) => l.getAttribute('href')?.startsWith('/patients/f'))
    expect(patientLinks).toHaveLength(2)
    expect(patientLinks[0]).toHaveAttribute('href', '/patients/f1')
    expect(patientLinks[1]).toHaveAttribute('href', '/patients/f2')
  })

  it('renders consent status dots', async () => {
    const { container } = renderWithQuery(<TodaysPatients />)
    await screen.findByText('Meera Sharma')
    const dots = container.querySelectorAll('.h-2.w-2.rounded-full')
    expect(dots).toHaveLength(2)
  })

  it('heading is an h2 element', () => {
    renderWithQuery(<TodaysPatients />)
    const heading = screen.getByText('My Patients')
    expect(heading.tagName).toBe('H2')
  })
})
