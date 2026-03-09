import { render, screen } from '@testing-library/react'
import { Badge } from '@/components/ui/Badge'

describe('Badge', () => {
  it('renders children text', () => {
    render(<Badge variant="success">Active</Badge>)
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('renders as a span element', () => {
    render(<Badge variant="info">Status</Badge>)
    const el = screen.getByText('Status')
    expect(el.tagName).toBe('SPAN')
  })

  describe('variants', () => {
    const variants = ['success', 'warning', 'danger', 'info', 'muted', 'accent'] as const

    variants.forEach((variant) => {
      it(`renders ${variant} variant`, () => {
        render(<Badge variant={variant}>{variant}</Badge>)
        expect(screen.getByText(variant)).toBeInTheDocument()
      })
    })
  })

  describe('sizes', () => {
    it('defaults to sm size', () => {
      render(<Badge variant="success">Small</Badge>)
      expect(screen.getByText('Small').className).toContain('text-[10px]')
    })

    it('applies md size classes', () => {
      render(<Badge variant="success" size="md">Medium</Badge>)
      expect(screen.getByText('Medium').className).toContain('text-[11px]')
    })
  })

  describe('dot indicator', () => {
    it('does not render dot by default', () => {
      const { container } = render(<Badge variant="success">No Dot</Badge>)
      const dots = container.querySelectorAll('.pulse-soft')
      expect(dots).toHaveLength(0)
    })

    it('renders dot when dot prop is true', () => {
      const { container } = render(<Badge variant="success" dot>With Dot</Badge>)
      const badge = screen.getByText('With Dot')
      const dot = badge.querySelector('.pulse-soft')
      expect(dot).toBeInTheDocument()
      expect(dot).toHaveClass('h-1.5', 'w-1.5', 'rounded-full')
    })
  })

  describe('className merging', () => {
    it('merges custom className', () => {
      render(<Badge variant="info" className="custom-class">Merged</Badge>)
      expect(screen.getByText('Merged')).toHaveClass('custom-class')
    })

    it('preserves base classes alongside custom ones', () => {
      render(<Badge variant="success" className="my-extra">Test</Badge>)
      const el = screen.getByText('Test')
      expect(el).toHaveClass('inline-flex')
      expect(el).toHaveClass('my-extra')
    })
  })

  it('renders with complex children', () => {
    render(
      <Badge variant="accent">
        <span data-testid="inner">Inner</span>
      </Badge>
    )
    expect(screen.getByTestId('inner')).toBeInTheDocument()
  })
})
