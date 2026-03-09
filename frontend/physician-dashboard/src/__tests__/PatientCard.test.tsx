import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}))

vi.mock('@medilink/shared', () => ({
  getInitials: vi.fn((name: string) => {
    return name.split(' ').map((w: string) => w[0]).join('').toUpperCase()
  }),
  getConsentDisplay: vi.fn((status: string) => {
    const map: Record<string, { label: string }> = {
      active: { label: 'Active' },
      revoked: { label: 'Revoked' },
      expired: { label: 'Expired' },
    }
    return map[status] || { label: status }
  }),
}))

import { PatientCard } from '@/components/patients/PatientCard'

const defaultPatient = {
  id: '1',
  fhirId: 'fhir-123',
  fullName: 'Ravi Patel',
  gender: 'male',
  birthDate: '1985-06-15',
}

const defaultConsent = {
  status: 'active' as const,
  expiresAt: '2026-01-01T00:00:00Z',
}

describe('PatientCard', () => {
  it('renders the patient full name', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    expect(screen.getByText('Ravi Patel')).toBeInTheDocument()
  })

  it('renders patient initials', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    expect(screen.getByText('RP')).toBeInTheDocument()
  })

  it('renders computed age from birthDate', () => {
    const currentYear = new Date().getFullYear()
    const expectedAge = currentYear - 1985
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    expect(screen.getByText(new RegExp(`${expectedAge} years`))).toBeInTheDocument()
  })

  it('renders capitalized gender', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    expect(screen.getByText(/Male/)).toBeInTheDocument()
  })

  it('renders dash when gender is not provided', () => {
    const patient = { ...defaultPatient, gender: undefined }
    render(<PatientCard patient={patient} consent={defaultConsent} />)
    expect(screen.getByText(/—/)).toBeInTheDocument()
  })

  it('does not render age when birthDate is not provided', () => {
    const patient = { ...defaultPatient, birthDate: undefined }
    render(<PatientCard patient={patient} consent={defaultConsent} />)
    expect(screen.queryByText(/years/)).not.toBeInTheDocument()
  })

  it('renders View button with correct link', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    const viewBtn = screen.getByText('View')
    const link = viewBtn.closest('a')
    expect(link).toHaveAttribute('href', '/patients/fhir-123')
  })

  it('renders Interactions button with correct link', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    const interactionsBtn = screen.getByText('Interactions')
    const link = interactionsBtn.closest('a')
    expect(link).toHaveAttribute('href', '/patients/fhir-123/prescribe')
  })

  it('renders consent status badge', () => {
    render(<PatientCard patient={defaultPatient} consent={defaultConsent} />)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders revoked consent badge when status is revoked', () => {
    const consent = { status: 'revoked' as const }
    render(<PatientCard patient={defaultPatient} consent={consent} />)
    expect(screen.getByText('Revoked')).toBeInTheDocument()
  })
})
