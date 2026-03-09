import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginForm } from '@/components/auth/LoginForm'

const mockLoginPhysician = vi.fn()
const mockVerifyTOTP = vi.fn()

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, variants, initial, animate, ...props }: any) => <div {...props}>{children}</div>,
    form: ({ children, variants, initial, animate, onSubmit, ...props }: any) => (
      <form onSubmit={onSubmit} {...props}>{children}</form>
    ),
    h1: ({ children, variants, ...props }: any) => <h1 {...props}>{children}</h1>,
    p: ({ children, variants, ...props }: any) => <p {...props}>{children}</p>,
  },
}))

vi.mock('@medilink/shared', async () => {
  const actual = await vi.importActual('@medilink/shared')
  return {
    ...actual,
    authAPI: {
      loginPhysician: (...args: any[]) => mockLoginPhysician(...args),
      verifyTOTP: (...args: any[]) => mockVerifyTOTP(...args),
    },
    parseAPIError: vi.fn(() => 'Invalid credentials'),
  }
})

vi.mock('react-hot-toast', () => ({
  default: { error: vi.fn(), success: vi.fn() },
}))

describe('LoginForm', () => {
  beforeEach(() => {
    mockLoginPhysician.mockReset()
    mockVerifyTOTP.mockReset()
  })

  it('renders email and password fields', () => {
    render(<LoginForm />)
    expect(screen.getByPlaceholderText(/email/i)).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/password/i)).toBeInTheDocument()
  })

  it('renders sign in button', () => {
    render(<LoginForm />)
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('form inputs have required attribute', () => {
    render(<LoginForm />)
    expect(screen.getByPlaceholderText(/email/i)).toBeRequired()
    expect(screen.getByPlaceholderText(/password/i)).toBeRequired()
  })

  it('button is enabled by default', () => {
    render(<LoginForm />)
    expect(screen.getByRole('button', { name: /sign in/i })).toBeEnabled()
  })

  it('shows error on invalid credentials', async () => {
    mockLoginPhysician.mockRejectedValue(new Error('Unauthorized'))
    const user = userEvent.setup()
    render(<LoginForm />)
    await user.type(screen.getByPlaceholderText(/email/i), 'bad@test.com')
    await user.type(screen.getByPlaceholderText(/password/i), 'wrong')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    await waitFor(() => {
      expect(screen.getByText(/invalid/i)).toBeInTheDocument()
    })
  })

  it('shows TOTP form when TOTP is required', async () => {
    mockLoginPhysician.mockResolvedValue({
      data: { accessToken: 'partial-token', requiresTOTP: true, expiresIn: 300, role: 'physician' },
    })
    const user = userEvent.setup()
    render(<LoginForm />)
    await user.type(screen.getByPlaceholderText(/email/i), 'totp@medilink.in')
    await user.type(screen.getByPlaceholderText(/password/i), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    await waitFor(() => {
      expect(screen.getByText(/two-factor/i)).toBeInTheDocument()
    })
  })

  it('has accessible form elements', () => {
    render(<LoginForm />)
    const emailInput = screen.getByPlaceholderText(/email/i)
    const passwordInput = screen.getByPlaceholderText(/password/i)
    expect(emailInput).toHaveAttribute('type', 'email')
    expect(passwordInput).toHaveAttribute('type', 'password')
  })

  it('renders MediLink branding', () => {
    render(<LoginForm />)
    expect(screen.getAllByText(/medilink/i).length).toBeGreaterThanOrEqual(1)
  })
})
