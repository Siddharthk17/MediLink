import { render, screen } from '@testing-library/react'
import { Card } from '@/components/ui/Card'

describe('Card', () => {
  it('renders children', () => {
    render(<Card>Card content</Card>)
    expect(screen.getByText('Card content')).toBeInTheDocument()
  })

  it('renders as a div', () => {
    render(<Card>Test</Card>)
    expect(screen.getByText('Test').tagName).toBe('DIV')
  })

  describe('padding', () => {
    it('defaults to md padding (p-5)', () => {
      render(<Card>Default pad</Card>)
      expect(screen.getByText('Default pad')).toHaveClass('p-5')
    })

    it('applies sm padding (p-4)', () => {
      render(<Card padding="sm">Small pad</Card>)
      expect(screen.getByText('Small pad')).toHaveClass('p-4')
    })

    it('applies lg padding (p-7)', () => {
      render(<Card padding="lg">Large pad</Card>)
      expect(screen.getByText('Large pad')).toHaveClass('p-7')
    })
  })

  describe('hover', () => {
    it('does not apply hover classes by default', () => {
      render(<Card>No hover</Card>)
      expect(screen.getByText('No hover')).not.toHaveClass('cursor-pointer')
    })

    it('applies hover classes when hover prop is true', () => {
      render(<Card hover>Hoverable</Card>)
      expect(screen.getByText('Hoverable')).toHaveClass('cursor-pointer')
    })
  })

  describe('className merging', () => {
    it('merges custom className', () => {
      render(<Card className="extra-style">Merged</Card>)
      const el = screen.getByText('Merged')
      expect(el).toHaveClass('extra-style')
      expect(el).toHaveClass('glass-card')
    })
  })

  it('renders nested elements', () => {
    render(
      <Card>
        <h2>Title</h2>
        <p>Description</p>
      </Card>
    )
    expect(screen.getByText('Title')).toBeInTheDocument()
    expect(screen.getByText('Description')).toBeInTheDocument()
  })
})
