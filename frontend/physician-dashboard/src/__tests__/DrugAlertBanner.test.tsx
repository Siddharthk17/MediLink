import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DrugAlertBanner } from '@/components/clinical/DrugAlertBanner'
import type { DrugCheckResult } from '@medilink/shared'

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: any) => <>{children}</>,
  motion: {
    div: ({ children, variants, initial, animate, exit, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

vi.mock('@medilink/shared', () => ({
  getSeverityDisplay: (severity: string) => {
    const map: Record<string, any> = {
      contraindicated: { label: 'Contraindicated', color: 'red', bgColor: '#fee', borderColor: '#fcc', icon: 'XCircle' },
      major: { label: 'Major', color: 'orange', bgColor: '#fef3c7', borderColor: '#fde68a', icon: 'AlertTriangle' },
      moderate: { label: 'Moderate', color: 'yellow', bgColor: '#fef9c3', borderColor: '#fef08a', icon: 'AlertCircle' },
      minor: { label: 'Minor', color: 'green', bgColor: '#d1fae5', borderColor: '#6ee7b7', icon: 'Info' },
      none: { label: 'None', color: 'gray', bgColor: '#f3f4f6', borderColor: '#d1d5db', icon: 'CheckCircle' },
    }
    return map[severity] || map.none
  },
}))

vi.mock('@/components/ui/Card', () => ({
  Card: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children, variant, size }: any) => (
    <span data-testid="badge" data-variant={variant}>{children}</span>
  ),
}))

vi.mock('@/components/ui/Button', () => ({
  Button: ({ children, disabled, loading, onClick, ...props }: any) => (
    <button disabled={disabled || loading} onClick={onClick} {...props}>{children}</button>
  ),
}))

vi.mock('@/lib/motion', () => ({
  alertPanelVariants: { hidden: {}, visible: {} },
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

function makeResult(overrides: Partial<DrugCheckResult> = {}): DrugCheckResult {
  return {
    highestSeverity: 'major' as any,
    newMedication: { name: 'Warfarin 5mg', rxnormCode: '855332' },
    interactions: [
      {
        drugA: { name: 'Warfarin 5mg', rxnormCode: '855332' },
        drugB: { name: 'Aspirin 75mg', rxnormCode: '198464' },
        severity: 'major' as any,
        description: 'Increased bleeding risk',
        mechanism: 'Additive anticoagulant effect',
        clinicalEffect: 'Hemorrhage',
        management: 'Monitor INR closely',
        source: 'drugbank',
        cached: false,
      },
    ],
    allergyConflicts: [],
    hasContraindication: false,
    checkComplete: true,
    ...overrides,
  } as DrugCheckResult
}

describe('DrugAlertBanner', () => {
  const mockAcknowledge = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    mockAcknowledge.mockResolvedValue(undefined)
  })

  it('renders nothing when severity is none', () => {
    const result = makeResult({ highestSeverity: 'none' as any })
    const { container } = render(
      <DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders interaction count header', () => {
    const result = makeResult()
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText(/1 drug interaction found/)).toBeInTheDocument()
  })

  it('pluralizes interaction count correctly', () => {
    const result = makeResult({
      interactions: [
        {
          drugA: { name: 'Warfarin 5mg', rxnormCode: '855332' },
          drugB: { name: 'Aspirin', rxnormCode: '198464' },
          severity: 'major' as any,
          description: 'Bleeding risk',
          source: 'drugbank',
          cached: false,
        } as any,
        {
          drugA: { name: 'Warfarin 5mg', rxnormCode: '855332' },
          drugB: { name: 'Ibuprofen', rxnormCode: '197806' },
          severity: 'moderate' as any,
          description: 'GI risk',
          source: 'drugbank',
          cached: false,
        } as any,
      ],
    })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText(/2 drug interactions found/)).toBeInTheDocument()
  })

  it('renders interaction medication names', () => {
    const result = makeResult()
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText(/Aspirin 75mg ↔ Warfarin 5mg/)).toBeInTheDocument()
  })

  it('renders interaction description', () => {
    const result = makeResult()
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText('Increased bleeding risk')).toBeInTheDocument()
  })

  it('expands interaction details on click', async () => {
    const user = userEvent.setup()
    const result = makeResult()
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    await user.click(screen.getByText(/Aspirin 75mg ↔ Warfarin 5mg/))
    expect(screen.getByText(/Additive anticoagulant effect/)).toBeInTheDocument()
    expect(screen.getByText(/Hemorrhage/)).toBeInTheDocument()
    expect(screen.getByText(/Monitor INR closely/)).toBeInTheDocument()
  })

  it('collapses interaction details on second click', async () => {
    const user = userEvent.setup()
    const result = makeResult()
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    const btn = screen.getByText(/Aspirin 75mg ↔ Warfarin 5mg/)
    await user.click(btn)
    expect(screen.getByText(/Additive anticoagulant effect/)).toBeInTheDocument()
    await user.click(btn)
    expect(screen.queryByText(/Additive anticoagulant effect/)).not.toBeInTheDocument()
  })

  it('renders allergy conflicts', () => {
    const result = makeResult({
      allergyConflicts: [{
        allergen: { name: 'Penicillin', rxnormCode: '723' },
        newMedication: { name: 'Warfarin 5mg', rxnormCode: '855332' },
        severity: 'contraindicated' as any,
        mechanism: 'Allergy cross-reactivity',
      }] as any,
    })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText(/Patient is allergic to Penicillin/)).toBeInTheDocument()
  })

  it('renders acknowledgment form for contraindicated severity', () => {
    const result = makeResult({ highestSeverity: 'contraindicated' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByPlaceholderText(/Document your clinical reasoning/)).toBeInTheDocument()
    expect(screen.getByText('Acknowledge Contraindication')).toBeInTheDocument()
  })

  it('disables acknowledge button when reason is less than 20 chars', () => {
    const result = makeResult({ highestSeverity: 'contraindicated' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText('Acknowledge Contraindication')).toBeDisabled()
  })

  it('enables acknowledge button when reason has 20+ characters', async () => {
    const user = userEvent.setup()
    const result = makeResult({ highestSeverity: 'contraindicated' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    const textarea = screen.getByPlaceholderText(/Document your clinical reasoning/)
    await user.type(textarea, 'Patient has no other options available here')
    expect(screen.getByText('Acknowledge Contraindication')).toBeEnabled()
  })

  it('calls onAcknowledge and shows success on submit', async () => {
    const user = userEvent.setup()
    const result = makeResult({ highestSeverity: 'contraindicated' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    const textarea = screen.getByPlaceholderText(/Document your clinical reasoning/)
    await user.type(textarea, 'Patient has no other options available here')
    await user.click(screen.getByText('Acknowledge Contraindication'))
    await waitFor(() => {
      expect(mockAcknowledge).toHaveBeenCalledWith('Patient has no other options available here')
    })
    await waitFor(() => {
      expect(screen.getByText(/Acknowledged/)).toBeInTheDocument()
    })
  })

  it('does not show acknowledgment form for non-contraindicated severity', () => {
    const result = makeResult({ highestSeverity: 'major' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.queryByPlaceholderText(/Document your clinical reasoning/)).not.toBeInTheDocument()
  })

  it('has aria-live polite for accessibility', () => {
    const result = makeResult()
    const { container } = render(
      <DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />
    )
    expect(container.querySelector('[aria-live="polite"]')).toBeInTheDocument()
  })

  it('renders character count indicator', () => {
    const result = makeResult({ highestSeverity: 'contraindicated' as any })
    render(<DrugAlertBanner result={result} onAcknowledge={mockAcknowledge} />)
    expect(screen.getByText('0/20 minimum')).toBeInTheDocument()
  })
})
