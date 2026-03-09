import { render, screen, fireEvent } from '@testing-library/react'
import { ErrorBoundary } from '@/components/ui/ErrorBoundary'

function ThrowingComponent({ message }: { message?: string }): React.ReactNode {
  throw new Error(message || 'Test error')
}

function GoodComponent() {
  return <div>All good</div>
}

describe('ErrorBoundary', () => {
  // Suppress console.error for expected errors
  const originalConsoleError = console.error
  beforeEach(() => {
    console.error = vi.fn()
  })
  afterEach(() => {
    console.error = originalConsoleError
  })

  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <GoodComponent />
      </ErrorBoundary>
    )
    expect(screen.getByText('All good')).toBeInTheDocument()
  })

  it('renders error UI when child throws', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('displays the error message', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent message="Network failure" />
      </ErrorBoundary>
    )
    expect(screen.getByText('Network failure')).toBeInTheDocument()
  })

  it('displays fallback message for errors without message', () => {
    const BareBoneThrow = () => { throw new Error() }
    render(
      <ErrorBoundary>
        <BareBoneThrow />
      </ErrorBoundary>
    )
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument()
  })

  it('renders a Try again button', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument()
  })

  it('resets error state when Try again is clicked', () => {
    let shouldThrow = true
    function ConditionalThrow() {
      if (shouldThrow) throw new Error('Boom')
      return <div>Recovered</div>
    }

    render(
      <ErrorBoundary>
        <ConditionalThrow />
      </ErrorBoundary>
    )
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    shouldThrow = false
    fireEvent.click(screen.getByRole('button', { name: /try again/i }))
    expect(screen.getByText('Recovered')).toBeInTheDocument()
  })

  it('shows the warning emoji', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>
    )
    expect(screen.getByText('⚠️')).toBeInTheDocument()
  })

  it('logs error to console', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent message="Logged error" />
      </ErrorBoundary>
    )
    expect(console.error).toHaveBeenCalled()
  })
})
