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
    render(<PageWrapper title="My Records"><p>Content</p></PageWrapper>)
    expect(screen.getByText('My Records')).toBeInTheDocument()
    expect(screen.getByText('My Records').tagName).toBe('H1')
  })

  it('renders subtitle when provided along with title', () => {
    render(
      <PageWrapper title="Health" subtitle="Overview of your health">
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByText('Overview of your health')).toBeInTheDocument()
  })

  it('does not render subtitle without title or actions', () => {
    render(
      <PageWrapper subtitle="Orphaned subtitle">
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.queryByText('Orphaned subtitle')).not.toBeInTheDocument()
  })

  it('renders actions when provided', () => {
    render(
      <PageWrapper actions={<button>Export</button>}>
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByRole('button', { name: 'Export' })).toBeInTheDocument()
  })

  it('does not render header section when no title or actions', () => {
    const { container } = render(<PageWrapper><p>Content</p></PageWrapper>)
    expect(container.querySelector('h1')).toBeNull()
  })

  it('renders title and actions together', () => {
    render(
      <PageWrapper title="Documents" actions={<button>Upload</button>}>
        <p>Content</p>
      </PageWrapper>
    )
    expect(screen.getByText('Documents')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Upload' })).toBeInTheDocument()
  })

  it('has max-w-7xl container class', () => {
    const { container } = render(<PageWrapper><p>C</p></PageWrapper>)
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper.className).toContain('max-w-7xl')
  })
})
