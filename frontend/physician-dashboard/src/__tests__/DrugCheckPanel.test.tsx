import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { DrugCheckPanel } from '@/components/clinical/DrugCheckPanel'

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
)

const mockCheck = vi.fn()
const mockAcknowledge = vi.fn()
const mockReset = vi.fn()
let mockResult: any = null
let mockIsChecking = false

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, variants, initial, animate, exit, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

vi.mock('@/hooks/useDrugCheck', () => ({
  useDrugCheck: () => ({
    result: mockResult,
    isChecking: mockIsChecking,
    check: mockCheck,
    acknowledge: mockAcknowledge,
    isAcknowledging: false,
    reset: mockReset,
  }),
}))

vi.mock('@/components/ui/Card', () => ({
  Card: ({ children, className, padding }: any) => (
    <div data-testid="card" className={className}>{children}</div>
  ),
}))

vi.mock('@/components/ui/Button', () => ({
  Button: ({ children, disabled, loading, onClick, ...props }: any) => (
    <button disabled={disabled || loading} onClick={onClick} {...props}>{children}</button>
  ),
}))

vi.mock('@/components/ui/Input', () => ({
  Input: ({ label, value, onChange, onFocus, placeholder, icon }: any) => (
    <div>
      {label && <label htmlFor="med-input">{label}</label>}
      <input
        id="med-input"
        value={value}
        onChange={onChange}
        onFocus={onFocus}
        placeholder={placeholder}
      />
    </div>
  ),
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children, variant }: any) => <span data-variant={variant}>{children}</span>,
}))

vi.mock('@/components/ui/Skeleton', () => ({
  Skeleton: ({ className }: any) => <div data-testid="skeleton" className={className} />,
}))

vi.mock('./DrugAlertBanner', () => ({
  DrugAlertBanner: ({ result, onAcknowledge }: any) => (
    <div data-testid="drug-alert-banner">Alert: {result.highestSeverity}</div>
  ),
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

vi.mock('@medilink/shared', () => ({
  fhirAPI: { createResource: vi.fn() },
  getSeverityDisplay: (s: string) => ({ label: s, color: 'red', bgColor: '#fee', borderColor: '#fcc', icon: 'AlertCircle' }),
}))

vi.mock('react-hot-toast', () => ({
  default: { success: vi.fn(), error: vi.fn() },
}))

describe('DrugCheckPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockResult = null
    mockIsChecking = false
  })

  it('renders the New Prescription heading', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('New Prescription')).toBeInTheDocument()
  })

  it('renders medication search input', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByPlaceholderText(/Search medication/)).toBeInTheDocument()
  })

  it('renders Check Interactions button (disabled without selection)', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    const btn = screen.getByText('Check Interactions')
    expect(btn).toBeInTheDocument()
    expect(btn).toBeDisabled()
  })

  it('shows medication suggestions when typing 2+ characters', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    expect(screen.getByText('Metformin 500mg')).toBeInTheDocument()
  })

  it('selects a medication from suggestions', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    await user.click(screen.getByText('Metformin 500mg'))
    expect(screen.getByPlaceholderText(/Search medication/)).toHaveValue('Metformin 500mg')
  })

  it('enables Check Interactions after selecting a medication', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    await user.click(screen.getByText('Metformin 500mg'))
    expect(screen.getByText('Check Interactions')).toBeEnabled()
  })

  it('calls check with rxnormCode when Check Interactions is clicked', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    await user.click(screen.getByText('Metformin 500mg'))
    await user.click(screen.getByText('Check Interactions'))
    expect(mockCheck).toHaveBeenCalledWith('861007')
  })

  it('shows no interactions message when severity is none', () => {
    mockResult = { highestSeverity: 'none', interactions: [], allergyConflicts: [], hasContraindication: false, checkComplete: true }
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('No known interactions found')).toBeInTheDocument()
    expect(screen.getByText('Safe to prescribe')).toBeInTheDocument()
  })

  it('shows placeholder when no result and not checking', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('Select a medication to check for interactions')).toBeInTheDocument()
  })

  it('shows skeletons while checking', () => {
    mockIsChecking = true
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getAllByTestId('skeleton').length).toBeGreaterThanOrEqual(1)
  })

  it('renders Prescribe button', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('Prescribe')).toBeInTheDocument()
  })

  it('disables Prescribe when no result', () => {
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('Prescribe')).toBeDisabled()
  })

  it('renders review checkbox for non-none non-contraindicated results', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'War')
    await user.click(screen.getByText('Warfarin 5mg'))
    mockResult = {
      highestSeverity: 'major',
      newMedication: { name: 'Warfarin 5mg', rxnormCode: '855332' },
      interactions: [{ drugA: { name: 'Warfarin 5mg', rxnormCode: '855332' }, drugB: { name: 'Aspirin', rxnormCode: '198464' }, severity: 'major', description: 'Risk', source: 'drugbank', cached: false }],
      allergyConflicts: [],
      hasContraindication: false,
      checkComplete: true,
    }
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getAllByText(/I have reviewed the interaction warnings/).length).toBeGreaterThanOrEqual(1)
  })

  it('shows drug class and form in suggestions', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    expect(screen.getByText(/Biguanide · Tablet/)).toBeInTheDocument()
  })

  it('shows rxnormCode in suggestions', async () => {
    const user = userEvent.setup()
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    await user.type(screen.getByPlaceholderText(/Search medication/), 'Met')
    expect(screen.getByText('861007')).toBeInTheDocument()
  })

  it('renders blocked text for contraindicated results', () => {
    mockResult = {
      highestSeverity: 'contraindicated',
      newMedication: { name: 'Drug', rxnormCode: '123' },
      interactions: [{ drugA: { name: 'Drug', rxnormCode: '123' }, drugB: { name: 'Other', rxnormCode: '456' }, severity: 'contraindicated', description: 'No', source: 'drugbank', cached: false }],
      allergyConflicts: [],
      hasContraindication: true,
      checkComplete: true,
    }
    render(<DrugCheckPanel patientId="p1" />, { wrapper })
    expect(screen.getByText('Blocked — Acknowledge Required')).toBeInTheDocument()
  })
})
