import { render, screen } from '@testing-library/react'
import { ObservationTable } from '@/components/labs/ObservationTable'
import type { Observation } from '@medilink/shared'

vi.mock('@/components/ui/Table', () => ({
  Table: ({ children }: any) => <table>{children}</table>,
  TableHeader: ({ children }: any) => <thead><tr>{children}</tr></thead>,
  TableHead: ({ children }: any) => <th>{children}</th>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableRow: ({ children, className }: any) => <tr className={className}>{children}</tr>,
  TableCell: ({ children, className }: any) => <td className={className}>{children}</td>,
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children, variant, size }: any) => (
    <span data-testid="badge" data-variant={variant}>{children}</span>
  ),
}))

vi.mock('@medilink/shared', () => ({
  getObservationValue: (obs: any) => obs.valueQuantity?.value ?? '—',
  getCodeDisplay: (code: any) => code?.coding?.[0]?.display || 'Unknown',
  formatDate: (d: string) => d ? '2024-01-15' : null,
  getObservationStatus: (obs: any) => {
    if (!obs.referenceRange?.[0]) return 'unknown'
    const val = obs.valueQuantity?.value
    const high = obs.referenceRange[0].high?.value
    const low = obs.referenceRange[0].low?.value
    if (high !== undefined && val > high) return 'high'
    if (low !== undefined && val < low) return 'low'
    return 'normal'
  },
}))

function makeObs(overrides: Partial<Observation> = {}): Observation {
  return {
    id: 'obs-1',
    resourceType: 'Observation',
    status: 'final',
    code: { coding: [{ system: 'http://loinc.org', code: '2339-0', display: 'Glucose' }] },
    effectiveDateTime: '2024-01-15T10:00:00Z',
    valueQuantity: { value: 95, unit: 'mg/dL' },
    referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    ...overrides,
  } as Observation
}

describe('ObservationTable', () => {
  it('renders table headers', () => {
    render(<ObservationTable observations={[]} />)
    expect(screen.getByText('Date')).toBeInTheDocument()
    expect(screen.getByText('Value')).toBeInTheDocument()
    expect(screen.getByText('Unit')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
    expect(screen.getByText('Interpretation')).toBeInTheDocument()
  })

  it('renders observation data', () => {
    render(<ObservationTable observations={[makeObs()]} />)
    expect(screen.getByText('2024-01-15')).toBeInTheDocument()
    expect(screen.getByText('95')).toBeInTheDocument()
    expect(screen.getByText('mg/dL')).toBeInTheDocument()
  })

  it('renders status badge', () => {
    render(<ObservationTable observations={[makeObs()]} />)
    expect(screen.getByText('final')).toBeInTheDocument()
  })

  it('renders NORMAL badge for values in range', () => {
    render(<ObservationTable observations={[makeObs()]} />)
    expect(screen.getByText('NORMAL')).toBeInTheDocument()
  })

  it('renders HIGH badge for values above range', () => {
    const obs = makeObs({
      id: 'obs-high',
      valueQuantity: { value: 150, unit: 'mg/dL' },
      referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    })
    render(<ObservationTable observations={[obs]} />)
    expect(screen.getByText('HIGH')).toBeInTheDocument()
  })

  it('renders LOW badge for values below range', () => {
    const obs = makeObs({
      id: 'obs-low',
      valueQuantity: { value: 50, unit: 'mg/dL' },
      referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    })
    render(<ObservationTable observations={[obs]} />)
    expect(screen.getByText('LOW')).toBeInTheDocument()
  })

  it('renders dash for unknown interpretation', () => {
    const obs = makeObs({
      id: 'obs-unknown',
      referenceRange: undefined,
    })
    render(<ObservationTable observations={[obs]} />)
    expect(screen.getByText('—', { selector: 'span' })).toBeInTheDocument()
  })

  it('renders dash when valueQuantity is missing', () => {
    const obs = makeObs({
      id: 'obs-no-val',
      valueQuantity: undefined,
      referenceRange: undefined,
    })
    render(<ObservationTable observations={[obs]} />)
    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(1)
  })

  it('renders multiple observations', () => {
    const observations = [
      makeObs({ id: 'o1', valueQuantity: { value: 95, unit: 'mg/dL' } }),
      makeObs({ id: 'o2', valueQuantity: { value: 120, unit: 'mg/dL' } }),
      makeObs({ id: 'o3', valueQuantity: { value: 60, unit: 'mg/dL' } }),
    ]
    render(<ObservationTable observations={observations} />)
    expect(screen.getByText('95')).toBeInTheDocument()
    expect(screen.getByText('120')).toBeInTheDocument()
    expect(screen.getByText('60')).toBeInTheDocument()
  })

  it('uses danger badge variant for HIGH', () => {
    const obs = makeObs({
      id: 'obs-high',
      valueQuantity: { value: 150, unit: 'mg/dL' },
      referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    })
    render(<ObservationTable observations={[obs]} />)
    const highBadge = screen.getByText('HIGH').closest('[data-testid="badge"]')
    expect(highBadge).toHaveAttribute('data-variant', 'danger')
  })

  it('uses warning badge variant for LOW', () => {
    const obs = makeObs({
      id: 'obs-low',
      valueQuantity: { value: 50, unit: 'mg/dL' },
      referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    })
    render(<ObservationTable observations={[obs]} />)
    const lowBadge = screen.getByText('LOW').closest('[data-testid="badge"]')
    expect(lowBadge).toHaveAttribute('data-variant', 'warning')
  })

  it('uses success badge variant for NORMAL', () => {
    render(<ObservationTable observations={[makeObs()]} />)
    const normalBadge = screen.getByText('NORMAL').closest('[data-testid="badge"]')
    expect(normalBadge).toHaveAttribute('data-variant', 'success')
  })
})
