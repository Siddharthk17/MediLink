import { render, screen } from '@testing-library/react'
import { LabTrendChart } from '@/components/labs/LabTrendChart'
import type { Observation } from '@medilink/shared'

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  ComposedChart: ({ children, data }: any) => <div data-testid="composed-chart" data-count={data?.length}>{children}</div>,
  Area: (props: any) => <div data-testid="area" />,
  Line: (props: any) => <div data-testid="line" />,
  XAxis: (props: any) => <div data-testid="x-axis" />,
  YAxis: (props: any) => <div data-testid="y-axis" />,
  CartesianGrid: (props: any) => <div data-testid="cartesian-grid" />,
  Tooltip: (props: any) => <div data-testid="tooltip" />,
  ReferenceArea: (props: any) => <div data-testid="reference-area" data-y1={props.y1} data-y2={props.y2} />,
}))

vi.mock('@medilink/shared', () => ({
  getObservationValue: (obs: any) => obs.valueQuantity?.value,
  formatDate: (d: string) => d ? new Date(d).toLocaleDateString() : null,
  getObservationStatus: (obs: any) => 'normal',
}))

function makeObservation(overrides: Partial<Observation> = {}): Observation {
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

describe('LabTrendChart', () => {
  it('renders empty state when no observations', () => {
    render(<LabTrendChart observations={[]} loincCode="2339-0" />)
    expect(screen.getByText('No data for this test type')).toBeInTheDocument()
  })

  it('renders empty state when observations have no valueQuantity', () => {
    const obs = makeObservation({ valueQuantity: undefined })
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.getByText('No data for this test type')).toBeInTheDocument()
  })

  it('renders chart container with valid data', () => {
    const obs = makeObservation()
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.getByTestId('responsive-container')).toBeInTheDocument()
    expect(screen.getByTestId('composed-chart')).toBeInTheDocument()
  })

  it('renders chart axes', () => {
    const obs = makeObservation()
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.getByTestId('x-axis')).toBeInTheDocument()
    expect(screen.getByTestId('y-axis')).toBeInTheDocument()
  })

  it('renders Area and Line components', () => {
    const obs = makeObservation()
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.getByTestId('area')).toBeInTheDocument()
    expect(screen.getByTestId('line')).toBeInTheDocument()
  })

  it('renders reference area when reference ranges exist', () => {
    const obs = makeObservation({
      referenceRange: [{ low: { value: 70 }, high: { value: 100 } }],
    })
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    const refArea = screen.getByTestId('reference-area')
    expect(refArea).toHaveAttribute('data-y1', '70')
    expect(refArea).toHaveAttribute('data-y2', '100')
  })

  it('does not render reference area when no ranges', () => {
    const obs = makeObservation({ referenceRange: undefined })
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.queryByTestId('reference-area')).not.toBeInTheDocument()
  })

  it('renders CartesianGrid', () => {
    const obs = makeObservation()
    render(<LabTrendChart observations={[obs]} loincCode="2339-0" />)
    expect(screen.getByTestId('cartesian-grid')).toBeInTheDocument()
  })

  it('passes correct data count to chart', () => {
    const observations = [
      makeObservation({ id: 'o1', effectiveDateTime: '2024-01-01T00:00:00Z', valueQuantity: { value: 90, unit: 'mg/dL' } }),
      makeObservation({ id: 'o2', effectiveDateTime: '2024-02-01T00:00:00Z', valueQuantity: { value: 95, unit: 'mg/dL' } }),
      makeObservation({ id: 'o3', effectiveDateTime: '2024-03-01T00:00:00Z', valueQuantity: { value: 110, unit: 'mg/dL' } }),
    ]
    render(<LabTrendChart observations={observations} loincCode="2339-0" />)
    expect(screen.getByTestId('composed-chart')).toHaveAttribute('data-count', '3')
  })

  it('sorts observations by date', () => {
    const observations = [
      makeObservation({ id: 'o2', effectiveDateTime: '2024-03-01T00:00:00Z', valueQuantity: { value: 95, unit: 'mg/dL' } }),
      makeObservation({ id: 'o1', effectiveDateTime: '2024-01-01T00:00:00Z', valueQuantity: { value: 90, unit: 'mg/dL' } }),
    ]
    render(<LabTrendChart observations={observations} loincCode="2339-0" />)
    expect(screen.getByTestId('composed-chart')).toHaveAttribute('data-count', '2')
  })
})
