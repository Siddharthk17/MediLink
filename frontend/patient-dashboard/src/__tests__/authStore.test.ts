import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/store/authStore'

const mockUser = {
  id: 'patient-1',
  role: 'patient' as const,
  status: 'active' as const,
  fullName: 'Meera Patient',
  totpEnabled: false,
}

const mockTokens = {
  accessToken: 'test-access-token',
  refreshToken: 'test-refresh-token',
}

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth()
  })

  it('starts with null user and no tokens', () => {
    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.accessToken).toBeNull()
    expect(state.refreshToken).toBeNull()
  })

  it('sets user and tokens on setAuth', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    const state = useAuthStore.getState()
    expect(state.user?.fullName).toBe('Meera Patient')
    expect(state.accessToken).toBe('test-access-token')
    expect(state.refreshToken).toBe('test-refresh-token')
  })

  it('clears state on clearAuth', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().clearAuth()
    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.accessToken).toBeNull()
    expect(state.refreshToken).toBeNull()
  })

  it('isAuthenticated returns true when token and user exist', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    expect(useAuthStore.getState().isAuthenticated()).toBe(true)
  })

  it('isAuthenticated returns false when no token', () => {
    expect(useAuthStore.getState().isAuthenticated()).toBe(false)
  })

  it('isAuthenticated returns false when token but no user', () => {
    useAuthStore.setState({ accessToken: 'some-token', user: null })
    expect(useAuthStore.getState().isAuthenticated()).toBe(false)
  })

  it('setTokens updates tokens', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().setTokens({ accessToken: 'new-token', refreshToken: 'new-refresh' })
    const state = useAuthStore.getState()
    expect(state.accessToken).toBe('new-token')
    expect(state.refreshToken).toBe('new-refresh')
  })

  it('setTokens with null clears auth entirely', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    useAuthStore.getState().setTokens(null)
    const state = useAuthStore.getState()
    expect(state.accessToken).toBeNull()
    expect(state.refreshToken).toBeNull()
    expect(state.user).toBeNull()
  })

  it('persists under medilink-patient-auth key', () => {
    expect(useAuthStore.persist.getOptions().name).toBe('medilink-patient-auth')
  })

  it('partializes only accessToken, refreshToken, and user', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    const state = useAuthStore.getState()
    const partialize = useAuthStore.persist.getOptions().partialize!
    const partial = partialize(state)
    expect(partial).toHaveProperty('accessToken')
    expect(partial).toHaveProperty('refreshToken')
    expect(partial).toHaveProperty('user')
    expect(partial).not.toHaveProperty('setAuth')
    expect(partial).not.toHaveProperty('clearAuth')
  })

  it('preserves user data across multiple setAuth calls', () => {
    useAuthStore.getState().setAuth(mockUser, mockTokens)
    const secondUser = { ...mockUser, id: 'patient-2', fullName: 'Second Patient' }
    useAuthStore.getState().setAuth(secondUser, { accessToken: 'tok2', refreshToken: 'ref2' })
    const state = useAuthStore.getState()
    expect(state.user?.fullName).toBe('Second Patient')
    expect(state.accessToken).toBe('tok2')
  })
})
