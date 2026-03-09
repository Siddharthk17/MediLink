import { render, screen } from '@testing-library/react'
import { AllergyList } from '@/components/clinical/AllergyList'

const mockData = {
  entry: [
    {
      resource: {
        id: 'a1',
        resourceType: 'AllergyIntolerance',
        code: { coding: [{ display: 'Penicillin' }] },
        criticality: 'high',
      },
    },
    {
      resource: {
        id: 'a2',
        resourceType: 'AllergyIntolerance',
        code: { coding: [{ display: 'Aspirin' }] },
        criticality: 'low',
      },
    },
  ],
}

let queryReturnData: any = mockData

vi.mock('@tanstack/react-query', () => ({
  useQuery: (opts: any) => ({
    data: queryReturnData,
    isLoading: false,
    error: null,
  }),
}))

vi.mock('@medilink/shared', () => ({
  fhirAPI: {
    searchResources: vi.fn(),
  },
  getCodeDisplay: (code: any) => code?.coding?.[0]?.display || 'Unknown',
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children, variant, size }: any) => (
    <span data-testid="badge" data-variant={variant}>{children}</span>
  ),
}))

describe('AllergyList', () => {
  beforeEach(() => {
    queryReturnData = mockData
  })

  it('renders allergy names from FHIR data', () => {
    render(<AllergyList patientId="p1" />)
    expect(screen.getByText(/Penicillin/)).toBeInTheDocument()
    expect(screen.getByText(/Aspirin/)).toBeInTheDocument()
  })

  it('renders criticality labels', () => {
    render(<AllergyList patientId="p1" />)
    expect(screen.getByText(/HIGH/)).toBeInTheDocument()
    expect(screen.getByText(/LOW/)).toBeInTheDocument()
  })

  it('uses danger variant badge for high criticality', () => {
    render(<AllergyList patientId="p1" />)
    const badges = screen.getAllByTestId('badge')
    expect(badges[0]).toHaveAttribute('data-variant', 'danger')
  })

  it('uses warning variant badge for non-high criticality', () => {
    render(<AllergyList patientId="p1" />)
    const badges = screen.getAllByTestId('badge')
    expect(badges[1]).toHaveAttribute('data-variant', 'warning')
  })

  it('renders nothing when no allergies are present', () => {
    queryReturnData = { entry: [] }
    const { container } = render(<AllergyList patientId="p1" />)
    expect(container.innerHTML).toBe('')
  })

  it('renders nothing when data is undefined', () => {
    queryReturnData = undefined
    const { container } = render(<AllergyList patientId="p1" />)
    expect(container.innerHTML).toBe('')
  })

  it('renders a dash when criticality is undefined', () => {
    queryReturnData = {
      entry: [
        {
          resource: {
            id: 'a3',
            resourceType: 'AllergyIntolerance',
            code: { coding: [{ display: 'Latex' }] },
          },
        },
      ],
    }
    render(<AllergyList patientId="p1" />)
    expect(screen.getByText(/—/)).toBeInTheDocument()
  })
})
