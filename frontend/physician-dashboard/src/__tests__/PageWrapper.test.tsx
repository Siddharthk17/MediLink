import { render, screen } from '@testing-library/react'
import { PageWrapper } from '@/components/layout/PageWrapper'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, variants, initial, animate, exit, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

vi.mock('@/lib/motion', () => ({
  pageVariants: {
    initial: { opacity: 0 },
    animate: { opacity: 1 },
    exit: { opacity: 0 },
  },
}))

describe('PageWrapper', () => {
  it('renders children', () => {
    render(<PageWrapper><p>Hello world</p></PageWrapper>)
    expect(screen.getByText('Hello world')).toBeInTheDocument()
  })

  it('renders title when provided', () => {
    render(<PageWrapper title="Dashboard"><p>Content</p></PageWrapper>)
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Dashboard').tagName).toBe('H1')
  })

  it('renders subtitle when provided along with title', () => {
    render(
      <PageWrapper title="Dashboard" subtitle="Welcome back">
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByText('Welcome back')).toBeInTheDocument()
  })

  it('does not render subtitle without title', () => {
    render(
      <PageWrapper subtitle="Welcome back">
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.queryByText('Welcome back')).not.toBeInTheDocument()
  })

  it('renders actions when provided', () => {
    render(
      <PageWrapper actions={<button>Action</button>}>
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument()
  })

  it('does not render header section when no title or actions', () => {
    const { container } = render(<PageWrapper><p>Content</p></PageWrapper>)
    expect(container.querySelector('h1')).toBeNull()
  })

  it('renders title and actions together', () => {
    render(
      <PageWrapper title="Patients" actions={<button>Add Patient</button>}>
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByText('Patients')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add Patient' })).toBeInTheDocument()
  })
})
