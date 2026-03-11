import { render, screen } from '@testing-library/react'
import { Card } from '@/components/ui/Card'

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('Card', () => {
  it('renders children', () => {
    render(<Card>Card Content</Card>)
    expect(screen.getByText('Card Content')).toBeInTheDocument()
  })

  it('renders a div element', () => {
    const { container } = render(<Card>Test</Card>)
    expect(container.firstChild?.nodeName).toBe('DIV')
  })

  describe('variant classes', () => {
    it('applies default variant class', () => {
      const { container } = render(<Card>Default</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('shadow-card')
    })

    it('applies glass variant class', () => {
      const { container } = render(<Card variant="glass">Glass</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('glass-card')
    })

    it('applies elevated variant class', () => {
      const { container } = render(<Card variant="elevated">Elevated</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('shadow-elevated')
    })
  })

  describe('padding classes', () => {
    it('applies md padding by default', () => {
      const { container } = render(<Card>Padded</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('p-5')
    })

    it('applies sm padding', () => {
      const { container } = render(<Card padding="sm">Small</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('p-4')
    })

    it('applies lg padding', () => {
      const { container } = render(<Card padding="lg">Large</Card>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('p-7')
    })
  })

  it('always has rounded-card class', () => {
    const { container } = render(<Card>Rounded</Card>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('rounded-card')
  })

  it('merges custom className', () => {
    const { container } = render(<Card className="custom-class">Merged</Card>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('custom-class')
  })

  it('passes through extra props', () => {
    render(<Card data-testid="my-card">Props</Card>)
    expect(screen.getByTestId('my-card')).toBeInTheDocument()
  })

  it('has displayName set to Card', () => {
    expect(Card.displayName).toBe('Card')
  })

  it('forwards ref', () => {
    const ref = vi.fn()
    render(<Card ref={ref}>Ref</Card>)
    expect(ref).toHaveBeenCalledWith(expect.any(HTMLDivElement))
  })
})
