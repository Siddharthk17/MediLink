import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TOTPInput } from '@/components/auth/TOTPInput'

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, variants, animate, ...props }: any) => <div {...props}>{children}</div>,
  },
}))

describe('TOTPInput', () => {
  const mockOnComplete = vi.fn()

  beforeEach(() => {
    mockOnComplete.mockClear()
  })

  it('renders 6 input boxes', () => {
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    expect(inputs).toHaveLength(6)
  })

  it('auto-advances focus on digit entry', async () => {
    const user = userEvent.setup()
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    await user.type(inputs[0], '1')
    expect(inputs[1]).toHaveFocus()
  })

  it('calls onComplete when all 6 digits entered', async () => {
    const user = userEvent.setup()
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    await user.type(inputs[0], '123456')
    expect(mockOnComplete).toHaveBeenCalledWith('123456')
  })

  it('only accepts numeric input', async () => {
    const user = userEvent.setup()
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    await user.type(inputs[0], 'abc')
    expect(inputs[0]).toHaveValue('')
  })

  it('handles paste of 6-digit code', async () => {
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    fireEvent.paste(inputs[0], {
      clipboardData: { getData: () => '654321' },
    })
    expect(mockOnComplete).toHaveBeenCalledWith('654321')
  })

  it('handles backspace to go to previous input', async () => {
    const user = userEvent.setup()
    render(<TOTPInput onComplete={mockOnComplete} />)
    const inputs = screen.getAllByRole('textbox')
    await user.type(inputs[0], '1')
    expect(inputs[1]).toHaveFocus()
    await user.keyboard('{Backspace}')
    expect(inputs[0]).toHaveFocus()
  })

  it('renders with disabled state', () => {
    render(<TOTPInput onComplete={mockOnComplete} disabled />)
    const inputs = screen.getAllByRole('textbox')
    inputs.forEach((input) => {
      expect(input).toBeDisabled()
    })
  })

  it('has correct aria labels', () => {
    render(<TOTPInput onComplete={mockOnComplete} />)
    for (let i = 1; i <= 6; i++) {
      expect(screen.getByLabelText(`Digit ${i}`)).toBeInTheDocument()
    }
  })
})
