import { render } from '@testing-library/react'
import { Skeleton } from '@/components/ui/Skeleton'

describe('Skeleton', () => {
  it('renders a div element', () => {
    const { container } = render(<Skeleton />)
    expect(container.firstChild).toBeInstanceOf(HTMLDivElement)
  })

  it('applies shimmer base class', () => {
    const { container } = render(<Skeleton />)
    expect(container.firstChild).toHaveClass('skeleton-shimmer')
  })

  describe('variants', () => {
    it('defaults to rectangular variant', () => {
      const { container } = render(<Skeleton />)
      expect(container.firstChild).toHaveClass('rounded-card')
    })

    it('applies text variant classes', () => {
      const { container } = render(<Skeleton variant="text" />)
      const el = container.firstChild as HTMLElement
      expect(el).toHaveClass('h-4')
      expect(el).toHaveClass('rounded')
    })

    it('applies circular variant class', () => {
      const { container } = render(<Skeleton variant="circular" />)
      expect(container.firstChild).toHaveClass('rounded-full')
    })

    it('applies rectangular variant class', () => {
      const { container } = render(<Skeleton variant="rectangular" />)
      expect(container.firstChild).toHaveClass('rounded-card')
    })
  })

  describe('className merging', () => {
    it('merges custom className', () => {
      const { container } = render(<Skeleton className="w-32 h-8" />)
      const el = container.firstChild as HTMLElement
      expect(el).toHaveClass('w-32')
      expect(el).toHaveClass('h-8')
      expect(el).toHaveClass('skeleton-shimmer')
    })
  })
})
