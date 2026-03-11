import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockRedirect = vi.fn()
const mockNext = vi.fn()

vi.mock('next/server', () => ({
  NextResponse: {
    redirect: (url: URL) => {
      mockRedirect(url.pathname)
      return { type: 'redirect', url: url.pathname }
    },
    next: () => {
      mockNext()
      return { type: 'next' }
    },
  },
}))

function createMockRequest(pathname: string, cookieToken?: string) {
  return {
    nextUrl: {
      pathname,
    },
    url: 'http://localhost:3002',
    cookies: {
      get: (name: string) => {
        if (name === 'medilink_patient_token' && cookieToken) {
          return { value: cookieToken }
        }
        return undefined
      },
    },
  } as any
}

describe('middleware', () => {
  beforeEach(async () => {
    mockRedirect.mockClear()
    mockNext.mockClear()
    vi.resetModules()
  })

  it('passes through API requests without checking auth', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/api/health'))
    expect(mockNext).toHaveBeenCalled()
    expect(mockRedirect).not.toHaveBeenCalled()
  })

  it('redirects root to /login when no token', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/'))
    expect(mockRedirect).toHaveBeenCalledWith('/login')
  })

  it('redirects root to /dashboard when token exists', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/', 'valid-token'))
    expect(mockRedirect).toHaveBeenCalledWith('/dashboard')
  })

  it('redirects unauthenticated user to /login from protected page', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/dashboard'))
    expect(mockRedirect).toHaveBeenCalledWith('/login')
  })

  it('allows unauthenticated user to access /login', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/login'))
    expect(mockNext).toHaveBeenCalled()
  })

  it('allows unauthenticated user to access /register', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/register'))
    expect(mockNext).toHaveBeenCalled()
  })

  it('redirects authenticated user from /login to /dashboard', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/login', 'valid-token'))
    expect(mockRedirect).toHaveBeenCalledWith('/dashboard')
  })

  it('redirects authenticated user from /register to /dashboard', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/register', 'valid-token'))
    expect(mockRedirect).toHaveBeenCalledWith('/dashboard')
  })

  it('allows authenticated user to access /dashboard', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/dashboard', 'valid-token'))
    expect(mockNext).toHaveBeenCalled()
  })

  it('allows authenticated user to access /health', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/health', 'valid-token'))
    expect(mockNext).toHaveBeenCalled()
  })

  it('allows authenticated user to access /consents', async () => {
    const { middleware } = await import('@/middleware')
    middleware(createMockRequest('/consents', 'valid-token'))
    expect(mockNext).toHaveBeenCalled()
  })

  it('exports config with correct matcher', async () => {
    const { config } = await import('@/middleware')
    expect(config.matcher).toEqual(['/((?!_next/static|_next/image|favicon.ico|public).*)'])
  })
})
