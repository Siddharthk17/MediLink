import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/store/authStore'

const mockUser = {
  id: '1',
  role: 'physician' as const,
  status: 'active' as const,
  fullName: 'Dr. Test User',
  totpEnabled: false,
}

const mockTokens = {
  accessToken: 'test-token',
  refreshToken: 'test-refresh',
}

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
  })

  it('starts with null user and no token', () => {
    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.accessToken).toBeNull()
  })

  it('sets user and tokens on setAuth', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    const state = useAuthStore.getState()
    expect(state.user?.fullName).toBe('Dr. Test User')
    expect(state.accessToken).toBe('test-token')
  })

  it('clears state on clearAuth', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().clearAuth()
    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.accessToken).toBeNull()
  })

  it('isAuthenticated returns true when token and user exist', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    expect(useAuthStore.getState().isAuthenticated()).toBe(true)
  })

  it('isAuthenticated returns false when no token', () => {
    expect(useAuthStore.getState().isAuthenticated()).toBe(false)
  })

  it('setTokens updates tokens', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().setTokens({ accessToken: 'new-token', refreshToken: 'new-refresh' })
    const state = useAuthStore.getState()
    expect(state.accessToken).toBe('new-token')
    expect(state.refreshToken).toBe('new-refresh')
  })

  it('setTokens with null clears auth', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().setTokens(null)
    const state = useAuthStore.getState()
    expect(state.accessToken).toBeNull()
    expect(state.user).toBeNull()
  })

  it('hasRole returns true for matching role', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    expect(useAuthStore.getState().hasRole('physician')).toBe(true)
  })

  it('hasRole returns false for non-matching role', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    expect(useAuthStore.getState().hasRole('admin')).toBe(false)
  })
})
