import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useUIStore } from '@/store/uiStore'

// Mock next/navigation
const mockPush = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}))

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, initial, animate, exit, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Search: (props: any) => <svg data-testid="icon-search" {...props} />,
  User: (props: any) => <svg data-testid="icon-user" {...props} />,
  FileText: (props: any) => <svg data-testid="icon-filetext" {...props} />,
  Pill: (props: any) => <svg data-testid="icon-pill" {...props} />,
  FlaskConical: (props: any) => <svg data-testid="icon-flask" {...props} />,
  Settings: (props: any) => <svg data-testid="icon-settings" {...props} />,
}))

// Import after mocks
import { CommandPalette } from '@/components/ui/CommandPalette'

function openPalette() {
  useUIStore.getState().toggleCommandPalette()
}

describe('CommandPalette', () => {
  beforeEach(() => {
    mockPush.mockClear()
    // Reset store state so palette is closed
    useUIStore.setState({ commandPaletteOpen: false })
  })

  it('renders nothing when closed', () => {
    render(<CommandPalette />)
    expect(screen.queryByPlaceholderText(/command/i)).not.toBeInTheDocument()
  })

  it('renders search input when open', () => {
    openPalette()
    render(<CommandPalette />)
    expect(screen.getByPlaceholderText(/command/i)).toBeInTheDocument()
  })

  it('shows all commands when query is empty', () => {
    openPalette()
    render(<CommandPalette />)
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Patient List')).toBeInTheDocument()
    expect(screen.getByText('Search Records')).toBeInTheDocument()
    expect(screen.getByText('Consent Management')).toBeInTheDocument()
    expect(screen.getByText('Notifications')).toBeInTheDocument()
    expect(screen.getByText('Admin Panel')).toBeInTheDocument()
  })

  it('filters commands by label', async () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    await userEvent.setup().type(input, 'Patient')
    expect(screen.getByText('Patient List')).toBeInTheDocument()
    expect(screen.queryByText('Admin Panel')).not.toBeInTheDocument()
  })

  it('filters commands by keyword', async () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    await userEvent.setup().type(input, 'consent')
    expect(screen.getByText('Consent Management')).toBeInTheDocument()
    expect(screen.queryByText('Dashboard')).not.toBeInTheDocument()
  })

  it('shows no results message when nothing matches', async () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    await userEvent.setup().type(input, 'xyznonexistent')
    expect(screen.getByText('No results found')).toBeInTheDocument()
  })

  it('executes command on click and navigates', async () => {
    openPalette()
    render(<CommandPalette />)
    fireEvent.click(screen.getByText('Dashboard'))
    expect(mockPush).toHaveBeenCalledWith('/dashboard')
  })

  it('closes palette after command execution', () => {
    openPalette()
    render(<CommandPalette />)
    fireEvent.click(screen.getByText('Dashboard'))
    expect(useUIStore.getState().commandPaletteOpen).toBe(false)
  })

  it('closes palette on backdrop click', () => {
    openPalette()
    const { container } = render(<CommandPalette />)
    // Backdrop is the first div with fixed class
    const backdrop = container.querySelector('[class*="backdrop-blur"]') as HTMLElement
    fireEvent.click(backdrop)
    expect(useUIStore.getState().commandPaletteOpen).toBe(false)
  })

  it('toggles palette with Ctrl+K', () => {
    render(<CommandPalette />)
    // Open
    fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
    expect(useUIStore.getState().commandPaletteOpen).toBe(true)
    // Close
    fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
    expect(useUIStore.getState().commandPaletteOpen).toBe(false)
  })

  it('closes palette on Escape key inside input', async () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(useUIStore.getState().commandPaletteOpen).toBe(false)
  })

  it('navigates commands with ArrowDown/ArrowUp keys', async () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)

    // First item (Dashboard) is selected by default - press Down to select second
    fireEvent.keyDown(input, { key: 'ArrowDown' })
    // Press Enter to execute second command (Patient List)
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(mockPush).toHaveBeenCalledWith('/patients')
  })

  it('ArrowUp does not go below 0', () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    fireEvent.keyDown(input, { key: 'ArrowUp' })
    fireEvent.keyDown(input, { key: 'Enter' })
    // Should still be first item (Dashboard)
    expect(mockPush).toHaveBeenCalledWith('/dashboard')
  })

  it('executes selected command on Enter', () => {
    openPalette()
    render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    // Default is first item (Dashboard)
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(mockPush).toHaveBeenCalledWith('/dashboard')
  })

  it('displays ESC keyboard hint', () => {
    openPalette()
    render(<CommandPalette />)
    expect(screen.getByText('ESC')).toBeInTheDocument()
  })

  it('displays command descriptions', () => {
    openPalette()
    render(<CommandPalette />)
    expect(screen.getByText('Go to home')).toBeInTheDocument()
    expect(screen.getByText('Browse consented patients')).toBeInTheDocument()
  })

  it('resets query and selection when reopened', () => {
    openPalette()
    const { unmount } = render(<CommandPalette />)
    const input = screen.getByPlaceholderText(/command/i)
    fireEvent.change(input, { target: { value: 'Patient' } })
    // Close and reopen
    useUIStore.getState().toggleCommandPalette()
    useUIStore.getState().toggleCommandPalette()
    unmount()
    render(<CommandPalette />)
    expect(screen.getByPlaceholderText(/command/i)).toHaveValue('')
  })
})
