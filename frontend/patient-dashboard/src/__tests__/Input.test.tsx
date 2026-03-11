import { render, screen } from '@testing-library/react'
import { Input } from '@/components/ui/Input'

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('Input', () => {
  it('renders an input element', () => {
    render(<Input placeholder="Enter text" />)
    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument()
  })

  it('renders label when provided', () => {
    render(<Input label="Email" id="email" />)
    expect(screen.getByText('Email')).toBeInTheDocument()
  })

  it('label is associated with input via htmlFor', () => {
    render(<Input label="Email" id="email-input" />)
    const label = screen.getByText('Email')
    expect(label).toHaveAttribute('for', 'email-input')
  })

  it('does not render label when not provided', () => {
    const { container } = render(<Input />)
    expect(container.querySelector('label')).not.toBeInTheDocument()
  })

  it('renders error message when error prop is set', () => {
    render(<Input error="Required field" />)
    expect(screen.getByText('Required field')).toBeInTheDocument()
  })

  it('error message has danger text color', () => {
    render(<Input error="Bad input" />)
    const errorEl = screen.getByText('Bad input')
    expect(errorEl.className).toContain('text-[var(--color-danger)]')
  })

  it('does not render error element when no error', () => {
    const { container } = render(<Input />)
    const errorP = container.querySelector('.text-\\[var\\(--color-danger\\)\\]')
    expect(errorP).not.toBeInTheDocument()
  })

  it('renders icon when provided', () => {
    render(<Input icon={<span data-testid="icon">🔍</span>} />)
    expect(screen.getByTestId('icon')).toBeInTheDocument()
  })

  it('adds pl-10 class when icon is present', () => {
    const { container } = render(<Input icon={<span>I</span>} />)
    const input = container.querySelector('input') as HTMLElement
    expect(input.className).toContain('pl-10')
  })

  it('has displayName set to Input', () => {
    expect(Input.displayName).toBe('Input')
  })

  it('forwards ref', () => {
    const ref = vi.fn()
    render(<Input ref={ref} />)
    expect(ref).toHaveBeenCalledWith(expect.any(HTMLInputElement))
  })

  it('passes through HTML input attributes', () => {
    const { container } = render(<Input type="password" name="pwd" autoComplete="current-password" />)
    const input = container.querySelector('input')!
    expect(input).toHaveAttribute('type', 'password')
    expect(input).toHaveAttribute('name', 'pwd')
    expect(input).toHaveAttribute('autocomplete', 'current-password')
  })

  it('merges custom className on input', () => {
    const { container } = render(<Input className="my-input" />)
    const input = container.querySelector('input') as HTMLElement
    expect(input.className).toContain('my-input')
  })

  it('applies error border class when error is set', () => {
    const { container } = render(<Input error="fail" />)
    const input = container.querySelector('input') as HTMLElement
    expect(input.className).toContain('border-[var(--color-danger)]')
  })
})
