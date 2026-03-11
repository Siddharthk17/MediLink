import { render, screen } from '@testing-library/react'
import { Badge } from '@/components/ui/Badge'

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('Badge', () => {
  it('renders children text', () => {
    render(<Badge>Active</Badge>)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders as a span element', () => {
    const { container } = render(<Badge>Test</Badge>)
    expect(container.firstChild?.nodeName).toBe('SPAN')
  })

  it('has base styling classes', () => {
    const { container } = render(<Badge>Base</Badge>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('inline-flex')
    expect(el.className).toContain('rounded-full')
    expect(el.className).toContain('text-xs')
    expect(el.className).toContain('font-medium')
  })

  describe('variants', () => {
    it('applies default variant', () => {
      const { container } = render(<Badge>Default</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-bg-elevated)]')
    })

    it('applies success variant', () => {
      const { container } = render(<Badge variant="success">Success</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-success-subtle)]')
      expect(el.className).toContain('text-[var(--color-success)]')
    })

    it('applies warning variant', () => {
      const { container } = render(<Badge variant="warning">Warning</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-warning-subtle)]')
    })

    it('applies danger variant', () => {
      const { container } = render(<Badge variant="danger">Danger</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-danger-subtle)]')
    })

    it('applies info variant', () => {
      const { container } = render(<Badge variant="info">Info</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-info-subtle)]')
    })

    it('applies accent variant', () => {
      const { container } = render(<Badge variant="accent">Accent</Badge>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-accent-subtle)]')
    })
  })

  it('merges custom className', () => {
    const { container } = render(<Badge className="extra">Merged</Badge>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('extra')
  })

  it('renders complex children', () => {
    const { container } = render(<Badge><span data-testid="icon">●</span> Active</Badge>)
    expect(screen.getByTestId('icon')).toBeInTheDocument()
    expect(container.firstChild?.textContent).toBe('● Active')
  })
})
