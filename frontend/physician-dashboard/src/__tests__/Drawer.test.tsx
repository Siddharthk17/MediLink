import { render, screen, fireEvent } from '@testing-library/react'
import { Drawer } from '@/components/ui/Drawer'

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, initial, animate, exit, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
    aside: ({ children, initial, animate, exit, transition, ...props }: any) => (
      <aside {...props}>{children}</aside>
    ),
  },
}))

describe('Drawer', () => {
  const onClose = vi.fn()

  beforeEach(() => {
    onClose.mockClear()
  })

  it('renders nothing when closed', () => {
    render(<Drawer open={false} onClose={onClose}>Content</Drawer>)
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })

  it('renders children when open', () => {
    render(<Drawer open={true} onClose={onClose}>Content</Drawer>)
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('renders title when provided', () => {
    render(<Drawer open={true} onClose={onClose} title="Notifications">Items</Drawer>)
    expect(screen.getByText('Notifications')).toBeInTheDocument()
  })

  it('does not render title header when title is absent', () => {
    const { container } = render(<Drawer open={true} onClose={onClose}>Items</Drawer>)
    expect(container.querySelector('h2')).not.toBeInTheDocument()
  })

  it('has role="dialog" and aria-modal', () => {
    render(<Drawer open={true} onClose={onClose} title="Drawer">Content</Drawer>)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-label', 'Drawer')
  })

  it('sets custom width via style', () => {
    render(<Drawer open={true} onClose={onClose} width={500}>Content</Drawer>)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveStyle({ width: '500px' })
  })

  it('uses default width of 380', () => {
    render(<Drawer open={true} onClose={onClose}>Content</Drawer>)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveStyle({ width: '380px' })
  })

  it('calls onClose when backdrop is clicked', () => {
    const { container } = render(<Drawer open={true} onClose={onClose}>Content</Drawer>)
    const backdrop = container.querySelector('.fixed > div:first-child') as HTMLElement
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when Escape key is pressed', () => {
    render(<Drawer open={true} onClose={onClose}>Content</Drawer>)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not call onClose on Escape when closed', () => {
    render(<Drawer open={false} onClose={onClose}>Content</Drawer>)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('renders close button when title is provided', () => {
    render(<Drawer open={true} onClose={onClose} title="Panel">Content</Drawer>)
    const closeBtn = screen.getByRole('button', { name: /close/i })
    expect(closeBtn).toBeInTheDocument()
  })

  it('calls onClose when close button is clicked', () => {
    render(<Drawer open={true} onClose={onClose} title="Panel">Content</Drawer>)
    fireEvent.click(screen.getByRole('button', { name: /close/i }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
