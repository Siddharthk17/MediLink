import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

vi.mock('framer-motion', () => {
  const set = vi.fn()
  return {
    useMotionValue: (_init: number) => ({ set }),
    useSpring: (mv: any) => ({
      on: (_event: string, cb: (v: number) => void) => {
        // Immediately invoke with whatever value was last set
        const lastVal = mv.set.mock?.lastCall?.[0] ?? 0
        cb(lastVal)
        return () => {}
      },
    }),
    AnimatePresence: ({ children }: any) => <>{children}</>,
    motion: {
      div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    },
  }
})

import { StatBlock } from '@/components/dashboard/StatCard'

describe('StatBlock', () => {
  it('renders the label text', () => {
    render(<StatBlock label="Total Patients" value={42} />)
    expect(screen.getByText('Total Patients')).toBeInTheDocument()
  })

  it('renders the animated numeric value', () => {
    render(<StatBlock label="Visits" value={128} />)
    expect(screen.getByText('128')).toBeInTheDocument()
  })

  it('renders change text when provided', () => {
    render(<StatBlock label="Revenue" value={50} change="+12% this week" />)
    expect(screen.getByText('+12% this week')).toBeInTheDocument()
  })

  it('does not render change text when not provided', () => {
    const { container } = render(<StatBlock label="Score" value={99} />)
    const changeParagraphs = container.querySelectorAll('.text-xs.mt-2.font-medium')
    expect(changeParagraphs).toHaveLength(0)
  })

  it('applies positive change color style', () => {
    render(<StatBlock label="Growth" value={10} change="+5%" changeType="positive" />)
    const el = screen.getByText('+5%')
    expect(el).toHaveStyle({ color: 'var(--color-success)' })
  })

  it('applies negative change color style', () => {
    render(<StatBlock label="Decline" value={3} change="-2%" changeType="negative" />)
    const el = screen.getByText('-2%')
    expect(el).toHaveStyle({ color: 'var(--color-danger)' })
  })

  it('defaults to neutral change color', () => {
    render(<StatBlock label="Stable" value={7} change="0%" />)
    const el = screen.getByText('0%')
    expect(el).toHaveStyle({ color: 'var(--color-text-muted)' })
  })

  it('label has uppercase styling class', () => {
    render(<StatBlock label="Test Label" value={1} />)
    const label = screen.getByText('Test Label')
    expect(label.className).toContain('uppercase')
  })

  it('renders inside a glass-card container', () => {
    const { container } = render(<StatBlock label="Card" value={0} />)
    expect(container.querySelector('.glass-card')).toBeInTheDocument()
  })

  it('formats large numbers with toLocaleString', () => {
    render(<StatBlock label="Big" value={1000} />)
    // toLocaleString on 1000 may produce "1,000" or "1000" depending on locale
    const displayed = screen.getByText((_content, element) => {
      return element?.tagName === 'P' && /1.?000/.test(element.textContent || '')
    })
    expect(displayed).toBeInTheDocument()
  })
})
