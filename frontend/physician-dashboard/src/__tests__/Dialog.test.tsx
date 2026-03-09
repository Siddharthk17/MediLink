import { render, screen, fireEvent } from '@testing-library/react'
import { Dialog } from '@/components/ui/Dialog'

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, initial, animate, exit, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

describe('Dialog', () => {
  const onClose = vi.fn()

  beforeEach(() => {
    onClose.mockClear()
  })

  it('renders nothing when closed', () => {
    render(<Dialog open={false} onClose={onClose}>Content</Dialog>)
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })

  it('renders children when open', () => {
    render(<Dialog open={true} onClose={onClose}>Content</Dialog>)
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('renders title when provided', () => {
    render(<Dialog open={true} onClose={onClose} title="My Dialog">Content</Dialog>)
    expect(screen.getByText('My Dialog')).toBeInTheDocument()
  })

  it('does not render title element when title is not provided', () => {
    const { container } = render(<Dialog open={true} onClose={onClose}>Content</Dialog>)
    expect(container.querySelector('h2')).not.toBeInTheDocument()
  })

  it('has role="dialog" and aria-modal', () => {
    render(<Dialog open={true} onClose={onClose} title="Test">Content</Dialog>)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-label', 'Test')
  })

  it('calls onClose when backdrop is clicked', () => {
    const { container } = render(<Dialog open={true} onClose={onClose}>Content</Dialog>)
    const backdrop = container.querySelector('.fixed > div:first-child') as HTMLElement
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when Escape key is pressed', () => {
    render(<Dialog open={true} onClose={onClose}>Content</Dialog>)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not call onClose on Escape when closed', () => {
    render(<Dialog open={false} onClose={onClose}>Content</Dialog>)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('does not call onClose on non-Escape keys', () => {
    render(<Dialog open={true} onClose={onClose}>Content</Dialog>)
    fireEvent.keyDown(document, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('renders complex children', () => {
    render(
      <Dialog open={true} onClose={onClose}>
        <p>Paragraph</p>
        <button>OK</button>
      </Dialog>
    )
    expect(screen.getByText('Paragraph')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'OK' })).toBeInTheDocument()
  })
})
