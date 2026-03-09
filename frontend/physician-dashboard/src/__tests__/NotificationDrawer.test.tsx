import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { NotificationDrawer } from '@/components/layout/NotificationDrawer'

let mockDrawerOpen = true
const mockToggleNotifications = vi.fn()

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, variants, initial, animate, exit, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
    aside: ({ children, variants, initial, animate, exit, transition, style, ...props }: any) => (
      <aside style={style} {...props}>{children}</aside>
    ),
  },
}))

vi.mock('@/store/uiStore', () => ({
  useUIStore: () => ({
    notificationDrawerOpen: mockDrawerOpen,
    toggleNotifications: mockToggleNotifications,
  }),
}))

vi.mock('@/components/ui/Drawer', () => ({
  Drawer: ({ open, onClose, title, children, width }: any) =>
    open ? (
      <div role="dialog" aria-label={title} data-testid="drawer">
        <h2>{title}</h2>
        <button onClick={onClose} aria-label="Close">Close</button>
        {children}
      </div>
    ) : null,
}))

describe('NotificationDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockDrawerOpen = true
  })

  it('renders the drawer with Notifications title when open', () => {
    render(<NotificationDrawer />)
    expect(screen.getByText('Notifications')).toBeInTheDocument()
  })

  it('renders empty state message', () => {
    render(<NotificationDrawer />)
    expect(screen.getByText('No notifications')).toBeInTheDocument()
    expect(screen.getByText(/Notifications about lab results/)).toBeInTheDocument()
  })

  it('calls toggleNotifications when close button is clicked', async () => {
    const user = userEvent.setup()
    render(<NotificationDrawer />)
    await user.click(screen.getByLabelText('Close'))
    expect(mockToggleNotifications).toHaveBeenCalledTimes(1)
  })

  it('does not render when drawer is closed', () => {
    mockDrawerOpen = false
    render(<NotificationDrawer />)
    expect(screen.queryByText('Notifications')).not.toBeInTheDocument()
  })
})
