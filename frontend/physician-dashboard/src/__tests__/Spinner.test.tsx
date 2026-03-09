import { render } from '@testing-library/react'
import { Spinner } from '@/components/ui/Spinner'

describe('Spinner', () => {
  it('renders an svg element', () => {
    const { container } = render(<Spinner />)
    const svg = container.querySelector('svg')
    expect(svg).toBeInTheDocument()
  })

  it('has animate-spin class', () => {
    const { container } = render(<Spinner />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveClass('animate-spin')
  })

  it('has correct viewBox', () => {
    const { container } = render(<Spinner />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('viewBox', '0 0 24 24')
  })

  describe('sizes', () => {
    it('defaults to md size (h-6 w-6)', () => {
      const { container } = render(<Spinner />)
      const svg = container.querySelector('svg')
      expect(svg).toHaveClass('h-6', 'w-6')
    })

    it('applies sm size (h-4 w-4)', () => {
      const { container } = render(<Spinner size="sm" />)
      const svg = container.querySelector('svg')
      expect(svg).toHaveClass('h-4', 'w-4')
    })

    it('applies lg size (h-8 w-8)', () => {
      const { container } = render(<Spinner size="lg" />)
      const svg = container.querySelector('svg')
      expect(svg).toHaveClass('h-8', 'w-8')
    })
  })

  describe('className merging', () => {
    it('merges custom className', () => {
      const { container } = render(<Spinner className="text-red-500" />)
      const svg = container.querySelector('svg')
      expect(svg).toHaveClass('text-red-500')
      expect(svg).toHaveClass('animate-spin')
    })
  })

  it('contains a circle and path child', () => {
    const { container } = render(<Spinner />)
    expect(container.querySelector('circle')).toBeInTheDocument()
    expect(container.querySelector('path')).toBeInTheDocument()
  })
})
