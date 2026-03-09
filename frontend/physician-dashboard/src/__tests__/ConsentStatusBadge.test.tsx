import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

vi.mock('@medilink/shared', () => ({
  getConsentDisplay: vi.fn((status: string) => {
    const map: Record<string, { label: string }> = {
      active: { label: 'Active' },
      revoked: { label: 'Revoked' },
      expired: { label: 'Expired' },
    }
    return map[status] || { label: status }
  }),
}))

import { ConsentStatusBadge } from '@/components/patients/ConsentStatusBadge'

describe('ConsentStatusBadge', () => {
  it('renders active consent label', () => {
    render(<ConsentStatusBadge status="active" />)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders revoked consent label', () => {
    render(<ConsentStatusBadge status="revoked" />)
    expect(screen.getByText('Revoked')).toBeInTheDocument()
  })

  it('renders expired consent label', () => {
    render(<ConsentStatusBadge status="expired" />)
    expect(screen.getByText('Expired')).toBeInTheDocument()
  })

  it('shows dot indicator only for active status', () => {
    const { container: activeContainer } = render(<ConsentStatusBadge status="active" />)
    const activeDot = activeContainer.querySelector('.pulse-soft')
    expect(activeDot).toBeInTheDocument()

    const { container: revokedContainer } = render(<ConsentStatusBadge status="revoked" />)
    const revokedDot = revokedContainer.querySelector('.pulse-soft')
    expect(revokedDot).not.toBeInTheDocument()
  })

  it('applies success variant class for active', () => {
    const { container } = render(<ConsentStatusBadge status="active" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('text-[var(--color-success)]')
  })

  it('applies danger variant class for revoked', () => {
    const { container } = render(<ConsentStatusBadge status="revoked" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('text-[var(--color-danger)]')
  })

  it('applies muted variant class for expired', () => {
    const { container } = render(<ConsentStatusBadge status="expired" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('text-[var(--color-text-muted)]')
  })

  it('accepts optional expiresAt prop without error', () => {
    expect(() => {
      render(<ConsentStatusBadge status="active" expiresAt="2025-12-31T00:00:00Z" />)
    }).not.toThrow()
  })
})
