import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TopBar } from '@/components/layout/TopBar'

const mockToggleNotifications = vi.fn()
const mockToggleCommandPalette = vi.fn()
const mockToggleTheme = vi.fn()

let mockTheme = 'light' as 'light' | 'dark'
let mockUser: { fullName: string; role: string } | null = { fullName: 'Dr. Smith', role: 'physician' }
let mockIsAdmin = false
let mockPathname = '/dashboard'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, variants, initial, animate, transition, ...props }: any) => (
      <div {...props}>{children}</div>
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
  useUIStore: () => ({
    toggleNotifications: mockToggleNotifications,
    toggleCommandPalette: mockToggleCommandPalette,
    theme: mockTheme,
    toggleTheme: mockToggleTheme,
  }),
}))

vi.mock('@/store/authStore', () => ({
  useAuthStore: () => ({
    user: mockUser,
    hasRole: (role: string) => role === 'admin' && mockIsAdmin,
    clearAuth: vi.fn(),
  }),
}))

vi.mock('@/components/aura/Interactions', () => ({
  Magnetic: ({ children }: any) => <div>{children}</div>,
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('TopBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockTheme = 'light'
    mockUser = { fullName: 'Dr. Smith', role: 'physician' }
    mockIsAdmin = false
    mockPathname = '/dashboard'
  })

  it('renders MediLink branding', () => {
    render(<TopBar />)
    expect(screen.getByText('MediLink')).toBeInTheDocument()
  })

  it('renders all navigation items', () => {
    render(<TopBar />)
    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.getByText('Patients')).toBeInTheDocument()
    expect(screen.getByText('Consents')).toBeInTheDocument()
    expect(screen.getByText('Search')).toBeInTheDocument()
    expect(screen.getByText('Notifications')).toBeInTheDocument()
  })

  it('renders navigation links with correct hrefs', () => {
    render(<TopBar />)
    expect(screen.getByText('Overview').closest('a')).toHaveAttribute('href', '/dashboard')
    expect(screen.getByText('Patients').closest('a')).toHaveAttribute('href', '/patients')
    expect(screen.getByText('Consents').closest('a')).toHaveAttribute('href', '/consents')
  })

  it('renders theme toggle button with correct label for light mode', () => {
    mockTheme = 'light'
    render(<TopBar />)
    expect(screen.getByLabelText('Switch to dark theme')).toBeInTheDocument()
  })

  it('renders theme toggle button with correct label for dark mode', () => {
    mockTheme = 'dark'
    render(<TopBar />)
    expect(screen.getByLabelText('Switch to light theme')).toBeInTheDocument()
  })

  it('calls toggleTheme when theme button is clicked', async () => {
    const user = userEvent.setup()
    render(<TopBar />)
    await user.click(screen.getByLabelText('Switch to dark theme'))
    expect(mockToggleTheme).toHaveBeenCalledTimes(1)
  })

  it('renders search command palette button', () => {
    render(<TopBar />)
    expect(screen.getByLabelText('Open search command palette')).toBeInTheDocument()
  })

  it('calls toggleCommandPalette on search button click', async () => {
    const user = userEvent.setup()
    render(<TopBar />)
    await user.click(screen.getByLabelText('Open search command palette'))
    expect(mockToggleCommandPalette).toHaveBeenCalledTimes(1)
  })

  it('renders notification button', () => {
    render(<TopBar />)
    expect(screen.getByLabelText('Open notifications')).toBeInTheDocument()
  })

  it('calls toggleNotifications on notification button click', async () => {
    const user = userEvent.setup()
    render(<TopBar />)
    await user.click(screen.getByLabelText('Open notifications'))
    expect(mockToggleNotifications).toHaveBeenCalledTimes(1)
  })

  it('renders user initial from fullName', async () => {
    render(<TopBar />)
    // After mount effect, user initial shows
    expect(await screen.findByText('D')).toBeInTheDocument()
  })

  it('renders default initial when user is null', async () => {
    mockUser = null
    render(<TopBar />)
    expect(await screen.findByText('U')).toBeInTheDocument()
  })

  it('renders user menu with logout', () => {
    mockIsAdmin = true
    render(<TopBar />)
    const menuBtn = screen.getByLabelText('User menu')
    fireEvent.click(menuBtn)
    expect(screen.getByText('Sign out')).toBeInTheDocument()
    expect(screen.getByText('Admin Panel')).toBeInTheDocument()
  })

  it('renders as a header element', () => {
    render(<TopBar />)
    expect(document.querySelector('header')).toBeInTheDocument()
  })
})
