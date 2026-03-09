import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { PrescriptionForm } from '@/components/clinical/PrescriptionForm'

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
)

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}))

vi.mock('@/hooks/useDrugCheck', () => ({
  useDrugCheck: () => ({
    result: null,
    isChecking: false,
    check: vi.fn(),
    acknowledge: vi.fn(),
    isAcknowledging: false,
    reset: vi.fn(),
  }),
}))

vi.mock('@/components/ui/Card', () => ({
  Card: ({ children, className, padding }: any) => <div>{children}</div>,
}))

vi.mock('@/components/ui/Button', () => ({
  Button: ({ children, disabled, loading, onClick, ...props }: any) => (
    <button disabled={disabled || loading} onClick={onClick} {...props}>{children}</button>
  ),
}))

vi.mock('@/components/ui/Input', () => ({
  Input: ({ label, value, onChange, onFocus, placeholder }: any) => (
    <div>
      {label && <label htmlFor="med-input">{label}</label>}
      <input id="med-input" value={value} onChange={onChange} onFocus={onFocus} placeholder={placeholder} />
    </div>
  ),
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children }: any) => <span>{children}</span>,
}))

vi.mock('@/components/ui/Skeleton', () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

vi.mock('@medilink/shared', () => ({
  fhirAPI: { createResource: vi.fn() },
  getSeverityDisplay: () => ({ label: 'None', color: 'gray', bgColor: '#eee', borderColor: '#ccc', icon: 'CheckCircle' }),
}))

vi.mock('react-hot-toast', () => ({
  default: { success: vi.fn(), error: vi.fn() },
}))

describe('PrescriptionForm', () => {
  it('renders — it is a re-export of DrugCheckPanel', () => {
    render(<PrescriptionForm patientId="p1" />, { wrapper })
    expect(screen.getByText('New Prescription')).toBeInTheDocument()
  })

  it('renders search input', () => {
    render(<PrescriptionForm patientId="p1" />, { wrapper })
    expect(screen.getByPlaceholderText(/Search medication/)).toBeInTheDocument()
  })

  it('renders Check Interactions button', () => {
    render(<PrescriptionForm patientId="p1" />, { wrapper })
    expect(screen.getByText('Check Interactions')).toBeInTheDocument()
  })

  it('renders Prescribe button', () => {
    render(<PrescriptionForm patientId="p1" />, { wrapper })
    expect(screen.getByText('Prescribe')).toBeInTheDocument()
  })
})
