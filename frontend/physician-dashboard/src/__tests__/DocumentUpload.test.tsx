import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { DocumentUpload } from '@/components/documents/DocumentUpload'

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
)

const mockStartPolling = vi.fn()
const mockStopPolling = vi.fn()
const mockSetJobs = vi.fn()
let mockJobs: any[] = []

vi.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}))

vi.mock('@/hooks/useDocumentJobs', () => ({
  useDocumentJobs: () => ({
    jobs: mockJobs,
    startPolling: mockStartPolling,
    stopPolling: mockStopPolling,
    setJobs: mockSetJobs,
  }),
}))

vi.mock('react-dropzone', () => ({
  useDropzone: ({ onDrop, onDropRejected }: any) => ({
    getRootProps: () => ({
      onClick: vi.fn(),
      role: 'presentation',
      'data-testid': 'dropzone',
    }),
    getInputProps: () => ({
      type: 'file',
      'data-testid': 'file-input',
    }),
    isDragActive: false,
  }),
}))

vi.mock('react-hot-toast', () => ({
  default: {
    loading: vi.fn(() => 'toast-id'),
    success: vi.fn(),
    error: vi.fn(),
    dismiss: vi.fn(),
  },
}))

vi.mock('@medilink/shared', () => ({
  apiClient: { post: vi.fn() },
  getJobStatusDisplay: (status: string) => {
    const map: Record<string, any> = {
      completed: { label: 'Completed', color: '#10B981' },
      processing: { label: 'Processing', color: '#3B82F6' },
      failed: { label: 'Failed', color: '#F43F5E' },
      queued: { label: 'Queued', color: '#F59E0B' },
    }
    return map[status] || { label: status, color: '#999' }
  },
  formatRelative: () => 'just now',
}))

vi.mock('@/components/ui/Card', () => ({
  Card: ({ children, padding }: any) => <div data-testid="card">{children}</div>,
}))

vi.mock('@/components/ui/Badge', () => ({
  Badge: ({ children, variant, dot }: any) => (
    <span data-testid="badge" data-variant={variant}>{children}</span>
  ),
}))

vi.mock('@/components/ui/Button', () => ({
  Button: ({ children, onClick, ...props }: any) => (
    <button onClick={onClick} {...props}>{children}</button>
  ),
}))

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}))

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('DocumentUpload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockJobs = []
  })

  it('renders the dropzone area', () => {
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('Drop lab report here')).toBeInTheDocument()
  })

  it('renders file type and size information', () => {
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText(/Supports PDF, JPG, PNG, WEBP/)).toBeInTheDocument()
    expect(screen.getByText(/max 20MB/)).toBeInTheDocument()
  })

  it('renders file input', () => {
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByTestId('file-input')).toBeInTheDocument()
  })

  it('does not render job list when no jobs', () => {
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.queryByText('Uploaded Reports')).not.toBeInTheDocument()
  })

  it('renders job list when jobs exist', () => {
    mockJobs = [
      {
        jobId: 'j1-abcdef-1234',
        status: 'completed',
        observationsCreated: 5,
        loincMapped: 3,
        uploadedAt: '2024-01-01T00:00:00Z',
        fhirReportId: 'report-1',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('Uploaded Reports')).toBeInTheDocument()
    expect(screen.getByText('Report j1-abcde')).toBeInTheDocument()
  })

  it('renders completed job with observation count', () => {
    mockJobs = [
      {
        jobId: 'j1-abcdef-1234',
        status: 'completed',
        observationsCreated: 8,
        loincMapped: 6,
        uploadedAt: '2024-01-01T00:00:00Z',
        fhirReportId: 'report-1',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText(/8 observations extracted/)).toBeInTheDocument()
    expect(screen.getByText(/6 LOINC codes mapped/)).toBeInTheDocument()
  })

  it('renders failed job with error message', () => {
    mockJobs = [
      {
        jobId: 'j2-abcdef-1234',
        status: 'failed',
        errorMessage: 'OCR failed',
        uploadedAt: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('OCR failed')).toBeInTheDocument()
  })

  it('renders default error for failed job without message', () => {
    mockJobs = [
      {
        jobId: 'j3-abcdef-1234',
        status: 'failed',
        uploadedAt: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('Processing failed')).toBeInTheDocument()
  })

  it('renders View DiagnosticReport link for completed jobs with fhirReportId', () => {
    mockJobs = [
      {
        jobId: 'j4-abcdef-1234',
        status: 'completed',
        observationsCreated: 2,
        loincMapped: 1,
        uploadedAt: '2024-01-01T00:00:00Z',
        fhirReportId: 'fhir-123',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('View DiagnosticReport')).toBeInTheDocument()
    const link = screen.getByText('View DiagnosticReport').closest('a')
    expect(link).toHaveAttribute('href', '/patients/p1/labs')
  })

  it('renders timestamps with formatRelative', () => {
    mockJobs = [
      {
        jobId: 'j5-abcdef-1234',
        status: 'processing',
        uploadedAt: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText(/just now/)).toBeInTheDocument()
  })

  it('renders status badge for each job', () => {
    mockJobs = [
      {
        jobId: 'j6-abcdef-1234',
        status: 'processing',
        uploadedAt: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('Processing')).toBeInTheDocument()
  })

  it('renders multiple jobs', () => {
    mockJobs = [
      { jobId: 'j7-abcdef-1234', status: 'completed', observationsCreated: 1, loincMapped: 1, uploadedAt: '2024-01-01', fhirReportId: 'r1' },
      { jobId: 'j8-abcdef-1234', status: 'pending', uploadedAt: '2024-01-02' },
    ]
    render(<DocumentUpload patientId="p1" />, { wrapper })
    expect(screen.getByText('Report j7-abcde')).toBeInTheDocument()
    expect(screen.getByText('Report j8-abcde')).toBeInTheDocument()
  })
})
