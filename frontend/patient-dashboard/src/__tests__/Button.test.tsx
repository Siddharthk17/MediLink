import { render, screen, fireEvent } from '@testing-library/react'
import { Button } from '@/components/ui/Button'

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('Button', () => {
  it('renders children text', () => {
    render(<Button>Click me</Button>)
    expect(screen.getByText('Click me')).toBeInTheDocument()
  })

  it('renders as a button element', () => {
    render(<Button>Btn</Button>)
    expect(screen.getByRole('button', { name: 'Btn' })).toBeInTheDocument()
  })

  it('fires onClick when clicked', () => {
    const handleClick = vi.fn()
    render(<Button onClick={handleClick}>Click</Button>)
    fireEvent.click(screen.getByText('Click'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('does not fire onClick when disabled', () => {
    const handleClick = vi.fn()
    render(<Button disabled onClick={handleClick}>Disabled</Button>)
    fireEvent.click(screen.getByText('Disabled'))
    expect(handleClick).not.toHaveBeenCalled()
  })

  describe('variant classes', () => {
    it('applies primary variant by default', () => {
      const { container } = render(<Button>Primary</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-accent)]')
      expect(el.className).toContain('text-white')
    })

    it('applies secondary variant', () => {
      const { container } = render(<Button variant="secondary">Secondary</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-bg-elevated)]')
    })

    it('applies ghost variant', () => {
      const { container } = render(<Button variant="ghost">Ghost</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('text-[var(--color-text-secondary)]')
    })

    it('applies danger variant', () => {
      const { container } = render(<Button variant="danger">Danger</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('bg-[var(--color-danger)]')
    })
  })

  describe('size classes', () => {
    it('applies md size by default', () => {
      const { container } = render(<Button>Medium</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('h-10')
      expect(el.className).toContain('px-4')
    })

    it('applies sm size', () => {
      const { container } = render(<Button size="sm">Small</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('h-8')
      expect(el.className).toContain('px-3')
    })

    it('applies lg size', () => {
      const { container } = render(<Button size="lg">Large</Button>)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('h-12')
      expect(el.className).toContain('px-6')
    })
  })

  it('has base styling classes', () => {
    const { container } = render(<Button>Base</Button>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('inline-flex')
    expect(el.className).toContain('items-center')
    expect(el.className).toContain('justify-center')
    expect(el.className).toContain('font-medium')
  })

  it('merges custom className', () => {
    const { container } = render(<Button className="custom">Custom</Button>)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('custom')
  })

  it('has displayName set to Button', () => {
    expect(Button.displayName).toBe('Button')
  })

  it('forwards ref', () => {
    const ref = vi.fn()
    render(<Button ref={ref}>Ref</Button>)
    expect(ref).toHaveBeenCalledWith(expect.any(HTMLButtonElement))
  })

  it('passes through type attribute', () => {
    render(<Button type="submit">Submit</Button>)
    expect(screen.getByRole('button')).toHaveAttribute('type', 'submit')
  })

  it('is disabled when disabled prop is true', () => {
    render(<Button disabled>Disabled</Button>)
    expect(screen.getByRole('button')).toBeDisabled()
  })
})
