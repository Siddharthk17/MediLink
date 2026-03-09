import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthUser, TokenPair } from '@medilink/shared'

interface AuthState {
  user: AuthUser | null
  accessToken: string | null
  refreshToken: string | null
  setAuth: (user: AuthUser, tokens: TokenPair) => void
  setTokens: (tokens: TokenPair | null) => void
  clearAuth: () => void
  isAuthenticated: () => boolean
  hasRole: (role: string) => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,

      setAuth: (user, tokens) => set({ user, ...tokens }),
      setTokens: (tokens) => tokens
        ? set(tokens)
        : set({ accessToken: null, refreshToken: null, user: null }),
      clearAuth: () => set({ user: null, accessToken: null, refreshToken: null }),
      isAuthenticated: () => !!get().accessToken && !!get().user,
      hasRole: (role) => get().user?.role === role,
    }),
    {
      name: 'medilink-physician-auth',
      partialize: (s) => ({
        accessToken: s.accessToken,
        refreshToken: s.refreshToken,
        user: s.user,
      }),
    }
  )
)
