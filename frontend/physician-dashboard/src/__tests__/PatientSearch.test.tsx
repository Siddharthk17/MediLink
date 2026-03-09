import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PatientSearch } from '@/components/patients/PatientSearch'

const defaultProps = {
  onSearch: vi.fn(),
  total: 42,
  filters: ['All', 'Active', 'Inactive'],
  activeFilter: 'All',
  onFilterChange: vi.fn(),
}

describe('PatientSearch', () => {
  beforeEach(() => {
    defaultProps.onSearch.mockClear()
    defaultProps.onFilterChange.mockClear()
  })

  it('renders search input with placeholder', () => {
    render(<PatientSearch {...defaultProps} />)
    expect(screen.getByPlaceholderText('Search patients...')).toBeInTheDocument()
  })

  it('displays total count', () => {
    render(<PatientSearch {...defaultProps} />)
    expect(screen.getByText('42 total')).toBeInTheDocument()
  })

  it('does not display total when not provided', () => {
    const { total, ...propsWithoutTotal } = defaultProps
    render(<PatientSearch {...propsWithoutTotal} />)
    expect(screen.queryByText(/total/)).not.toBeInTheDocument()
  })

  it('renders all filter buttons', () => {
    render(<PatientSearch {...defaultProps} />)
    expect(screen.getByText('All')).toBeInTheDocument()
    expect(screen.getByText('Active')).toBeInTheDocument()
    expect(screen.getByText('Inactive')).toBeInTheDocument()
  })

  it('calls onSearch when typing in the search input', async () => {
    const user = userEvent.setup()
    render(<PatientSearch {...defaultProps} />)
    const input = screen.getByPlaceholderText('Search patients...')
    await user.type(input, 'Patel')
    expect(defaultProps.onSearch).toHaveBeenCalledWith('P')
    expect(defaultProps.onSearch).toHaveBeenLastCalledWith('Patel')
  })

  it('calls onFilterChange when a filter button is clicked', async () => {
    const user = userEvent.setup()
    render(<PatientSearch {...defaultProps} />)
    await user.click(screen.getByText('Active'))
    expect(defaultProps.onFilterChange).toHaveBeenCalledWith('Active')
  })

  it('highlights the active filter button', () => {
    render(<PatientSearch {...defaultProps} activeFilter="Active" />)
    const activeBtn = screen.getByText('Active')
    expect(activeBtn.className).toContain('text-[var(--color-text-accent)]')
  })

  it('non-active filter buttons have muted styling', () => {
    render(<PatientSearch {...defaultProps} activeFilter="All" />)
    const inactiveBtn = screen.getByText('Inactive')
    expect(inactiveBtn.className).toContain('text-[var(--color-text-muted)]')
  })

  it('updates input value as user types', async () => {
    const user = userEvent.setup()
    render(<PatientSearch {...defaultProps} />)
    const input = screen.getByPlaceholderText('Search patients...')
    await user.type(input, 'Sharma')
    expect(input).toHaveValue('Sharma')
  })
})
