import { vi } from 'vitest'
import { queryClient } from '@/lib/queryClient'

describe('queryClient', () => {
  it('is a QueryClient instance', () => {
    expect(queryClient).toBeDefined()
    expect(typeof queryClient.getDefaultOptions).toBe('function')
  })

  it('has staleTime set to 30 seconds', () => {
    const opts = queryClient.getDefaultOptions()
    expect(opts.queries?.staleTime).toBe(30_000)
  })

  it('has gcTime set to 5 minutes', () => {
    const opts = queryClient.getDefaultOptions()
    expect(opts.queries?.gcTime).toBe(5 * 60_000)
  })

  it('has retry set to 1', () => {
    const opts = queryClient.getDefaultOptions()
    expect(opts.queries?.retry).toBe(1)
  })

  it('has refetchOnWindowFocus enabled', () => {
    const opts = queryClient.getDefaultOptions()
    expect(opts.queries?.refetchOnWindowFocus).toBe(true)
  })

  it('has a mutation onError handler that only logs in development', () => {
    const opts = queryClient.getDefaultOptions()
    expect(typeof opts.mutations?.onError).toBe('function')

    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const testError = new Error('test mutation error')
    opts.mutations!.onError!(testError, undefined as never, undefined as never, undefined as never)
    // In test environment (NODE_ENV=test), console.error should NOT be called
    expect(spy).not.toHaveBeenCalled()
    spy.mockRestore()
  })
})
