import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TopBar } from '@/components/layout/TopBar'

const mockToggleCommandPalette = vi.fn()
const mockToggleTheme = vi.fn()
const mockClearAuth = vi.fn()

let mockTheme = 'light' as 'light' | 'dark'
let mockUser: { fullName: string; role: string } | null = { fullName: 'Meera Patient', role: 'patient' }
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
    toggleCommandPalette: mockToggleCommandPalette,
    theme: mockTheme,
    toggleTheme: mockToggleTheme,
  }),
}))

vi.mock('@/store/authStore', () => ({
  useAuthStore: () => ({
    user: mockUser,
    clearAuth: mockClearAuth,
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
    mockUser = { fullName: 'Meera Patient', role: 'patient' }
    mockPathname = '/dashboard'
  })

  it('renders MediLink branding', () => {
    render(<TopBar />)
    expect(screen.getByText('MediLink')).toBeInTheDocument()
  })

  it('renders patient navigation items', () => {
    render(<TopBar />)
    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.getByText('Health')).toBeInTheDocument()
    expect(screen.getByText('Doctors')).toBeInTheDocument()
    expect(screen.getByText('Consents')).toBeInTheDocument()
    expect(screen.getByText('Documents')).toBeInTheDocument()
    expect(screen.getByText('Alerts')).toBeInTheDocument()
  })

  it('renders navigation links with correct hrefs', () => {
    render(<TopBar />)
    expect(screen.getByText('Overview').closest('a')).toHaveAttribute('href', '/dashboard')
    expect(screen.getByText('Health').closest('a')).toHaveAttribute('href', '/health')
    expect(screen.getByText('Doctors').closest('a')).toHaveAttribute('href', '/find-doctor')
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

  it('renders search button', () => {
    render(<TopBar />)
    expect(screen.getByLabelText('Search records')).toBeInTheDocument()
  })

  it('calls toggleCommandPalette on search button click', async () => {
    const user = userEvent.setup()
    render(<TopBar />)
    await user.click(screen.getByLabelText('Search records'))
    expect(mockToggleCommandPalette).toHaveBeenCalledTimes(1)
  })

  it('renders notifications link', () => {
    render(<TopBar />)
    expect(screen.getByLabelText('Notifications')).toBeInTheDocument()
  })

  it('notifications link points to /notifications', () => {
    render(<TopBar />)
    expect(screen.getByLabelText('Notifications').closest('a')).toHaveAttribute('href', '/notifications')
  })

  it('renders user initial from fullName', async () => {
    render(<TopBar />)
    expect(await screen.findByText('M')).toBeInTheDocument()
  })

  it('renders default initial P when user is null', async () => {
    mockUser = null
    render(<TopBar />)
    expect(await screen.findByText('P')).toBeInTheDocument()
  })

  it('renders user menu with sign out option', () => {
    render(<TopBar />)
    const menuBtn = screen.getByLabelText('User menu')
    fireEvent.click(menuBtn)
    expect(screen.getByText('Sign out')).toBeInTheDocument()
  })

  it('renders user menu with My Profile link', () => {
    render(<TopBar />)
    const menuBtn = screen.getByLabelText('User menu')
    fireEvent.click(menuBtn)
    expect(screen.getByText('My Profile')).toBeInTheDocument()
  })

  it('renders user name in dropdown', () => {
    render(<TopBar />)
    const menuBtn = screen.getByLabelText('User menu')
    fireEvent.click(menuBtn)
    expect(screen.getByText('Meera Patient')).toBeInTheDocument()
  })

  it('shows Patient Portal text in menu', () => {
    render(<TopBar />)
    fireEvent.click(screen.getByLabelText('User menu'))
    expect(screen.getByText('Patient Portal')).toBeInTheDocument()
  })

  it('renders as a header element', () => {
    render(<TopBar />)
    expect(document.querySelector('header')).toBeInTheDocument()
  })

  it('MediLink link points to /dashboard', () => {
    render(<TopBar />)
    const brandLink = screen.getByText('MediLink').closest('a')
    expect(brandLink).toHaveAttribute('href', '/dashboard')
  })
})
