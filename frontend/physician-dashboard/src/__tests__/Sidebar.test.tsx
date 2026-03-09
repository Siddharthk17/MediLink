import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Sidebar } from '@/components/layout/Sidebar'

let mockPathname = '/dashboard'
let mockSidebarExpanded = false
let mockSidebarPinned = false
const mockSetSidebarPinned = vi.fn()
const mockClearAuth = vi.fn()
let mockIsAdmin = false
let mockUser: { fullName: string; role: string } | null = { fullName: 'Dr. Alice', role: 'physician' }

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, variants, initial, animate, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
    nav: ({ children, variants, initial, animate, onMouseEnter, onMouseLeave, ...props }: any) => (
      <nav onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave} {...props}>{children}</nav>
    ),
    span: ({ children, variants, initial, animate, transition, ...props }: any) => (
      <span {...props}>{children}</span>
    ),
  },
}))

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}))

vi.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
}))

vi.mock('@/store/uiStore', () => ({
  useUIStore: Object.assign(
    () => ({
      sidebarExpanded: mockSidebarExpanded,
      sidebarPinned: mockSidebarPinned,
      setSidebarPinned: mockSetSidebarPinned,
    }),
    { setState: vi.fn() }
  ),
}))

vi.mock('@/store/authStore', () => ({
  useAuthStore: () => ({
    user: mockUser,
    clearAuth: mockClearAuth,
    hasRole: (role: string) => mockIsAdmin && role === 'admin',
  }),
}))

vi.mock('@/lib/motion', () => ({
  sidebarVariants: { collapsed: { width: 64 }, expanded: { width: 240 } },
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

vi.mock('@/components/ui/Tooltip', () => ({
  Tooltip: ({ children, content }: any) => <div title={content}>{children}</div>,
}))

describe('Sidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockPathname = '/dashboard'
    mockSidebarExpanded = true
    mockSidebarPinned = false
    mockIsAdmin = false
    mockUser = { fullName: 'Dr. Alice', role: 'physician' }
  })

  it('renders main navigation items', () => {
    render(<Sidebar />)
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Patients')).toBeInTheDocument()
    expect(screen.getByText('Consents')).toBeInTheDocument()
    expect(screen.getByText('Search')).toBeInTheDocument()
  })

  it('renders MediLink branding when expanded', () => {
    mockSidebarExpanded = true
    render(<Sidebar />)
    expect(screen.getByText('MediLink')).toBeInTheDocument()
  })

  it('renders "M" logo always', () => {
    render(<Sidebar />)
    expect(screen.getByText('M')).toBeInTheDocument()
  })

  it('renders nav links with correct hrefs', () => {
    render(<Sidebar />)
    const dashboardLink = screen.getByText('Dashboard').closest('a')
    expect(dashboardLink).toHaveAttribute('href', '/dashboard')
    const patientsLink = screen.getByText('Patients').closest('a')
    expect(patientsLink).toHaveAttribute('href', '/patients')
  })

  it('does not render admin nav for non-admin users', () => {
    mockIsAdmin = false
    render(<Sidebar />)
    expect(screen.queryByText('Admin')).not.toBeInTheDocument()
    expect(screen.queryByText('Audit Logs')).not.toBeInTheDocument()
    expect(screen.queryByText('Users')).not.toBeInTheDocument()
  })

  it('renders admin nav items for admin users', () => {
    mockIsAdmin = true
    render(<Sidebar />)
    expect(screen.getByText('Admin')).toBeInTheDocument()
    expect(screen.getByText('Audit Logs')).toBeInTheDocument()
    expect(screen.getByText('Users')).toBeInTheDocument()
  })

  it('renders notifications link', () => {
    render(<Sidebar />)
    expect(screen.getByText('Notifications')).toBeInTheDocument()
  })

  it('renders pin sidebar button when expanded', () => {
    mockSidebarExpanded = true
    render(<Sidebar />)
    expect(screen.getByLabelText('Pin sidebar')).toBeInTheDocument()
  })

  it('renders unpin label when sidebar is pinned', () => {
    mockSidebarExpanded = true
    mockSidebarPinned = true
    render(<Sidebar />)
    expect(screen.getByLabelText('Unpin sidebar')).toBeInTheDocument()
    expect(screen.getByText('Unpin')).toBeInTheDocument()
  })

  it('calls setSidebarPinned on pin button click', async () => {
    mockSidebarExpanded = true
    const user = userEvent.setup()
    render(<Sidebar />)
    await user.click(screen.getByLabelText('Pin sidebar'))
    expect(mockSetSidebarPinned).toHaveBeenCalledWith(true)
  })

  it('renders user initial from fullName', () => {
    render(<Sidebar />)
    expect(screen.getByText('D')).toBeInTheDocument()
  })

  it('renders user fullName when expanded', () => {
    mockSidebarExpanded = true
    render(<Sidebar />)
    expect(screen.getByText('Dr. Alice')).toBeInTheDocument()
  })

  it('renders Sign out button when expanded', () => {
    mockSidebarExpanded = true
    render(<Sidebar />)
    expect(screen.getByText('Sign out')).toBeInTheDocument()
  })

  it('renders with aria-label for main navigation', () => {
    render(<Sidebar />)
    expect(screen.getByLabelText('Main navigation')).toBeInTheDocument()
  })

  it('shows default name when user is null', () => {
    mockUser = null
    mockSidebarExpanded = true
    render(<Sidebar />)
    expect(screen.getByText('Physician')).toBeInTheDocument()
  })
})
