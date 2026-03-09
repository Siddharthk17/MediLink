import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}))

import { QuickActions } from '@/components/dashboard/QuickActions'

describe('QuickActions', () => {
  it('renders all three action links', () => {
    render(<QuickActions />)
    expect(screen.getByText('Patients')).toBeInTheDocument()
    expect(screen.getByText('Search records')).toBeInTheDocument()
    expect(screen.getByText('Consents')).toBeInTheDocument()
  })

  it('renders correct hrefs for each action', () => {
    render(<QuickActions />)
    const links = screen.getAllByRole('link')
    expect(links).toHaveLength(3)
    expect(links[0]).toHaveAttribute('href', '/patients')
    expect(links[1]).toHaveAttribute('href', '/search')
    expect(links[2]).toHaveAttribute('href', '/consents')
  })

  it('displays keyboard shortcut keys', () => {
    render(<QuickActions />)
    expect(screen.getByText('P')).toBeInTheDocument()
    expect(screen.getByText('S')).toBeInTheDocument()
    expect(screen.getByText('C')).toBeInTheDocument()
  })

  it('renders keyboard shortcuts inside kbd elements', () => {
    const { container } = render(<QuickActions />)
    const kbds = container.querySelectorAll('kbd')
    expect(kbds).toHaveLength(3)
    expect(kbds[0].textContent).toBe('P')
    expect(kbds[1].textContent).toBe('S')
    expect(kbds[2].textContent).toBe('C')
  })

  it('each link has the rounded-full class', () => {
    render(<QuickActions />)
    const links = screen.getAllByRole('link')
    links.forEach((link) => {
      expect(link.className).toContain('rounded-full')
    })
  })
})
