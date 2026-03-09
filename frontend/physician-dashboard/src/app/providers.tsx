'use client'

import { QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import { queryClient } from '@/lib/queryClient'
import { useEffect, useState } from 'react'
import { configureClient } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { CommandPalette } from '@/components/ui/CommandPalette'

let mswReady = false

async function startMSW() {
  if (typeof window === 'undefined') return
  if (process.env.NODE_ENV !== 'development') return
  if (process.env.NEXT_PUBLIC_DISABLE_MSW === 'true') return

  const { worker } = await import('@/lib/msw/browser')
  await worker.start({
    onUnhandledRequest: 'bypass',
    serviceWorker: { url: '/mockServiceWorker.js' },
  })
  mswReady = true
}

function MSWProvider({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(
    process.env.NODE_ENV !== 'development' || process.env.NEXT_PUBLIC_DISABLE_MSW === 'true'
  )

  useEffect(() => {
    if (!ready) {
      startMSW().then(() => setReady(true))
    }
  }, [ready])

  if (!ready) return null
  return <>{children}</>
}

function AuthClientSetup() {
  useEffect(() => {
    const baseURL = mswReady ? 'http://localhost:8580' : '/api'

    configureClient({
      baseURL,
      // Read directly from zustand store to avoid stale closure after token refresh
      getTokens: () => {
        const { accessToken, refreshToken } = useAuthStore.getState()
        return { accessToken, refreshToken }
      },
      setTokens: (pair) => {
        if (pair) {
          useAuthStore.getState().setTokens(pair)
          document.cookie = `medilink_access_token=${pair.accessToken}; path=/; samesite=strict`
        } else {
          useAuthStore.getState().setTokens(null)
          document.cookie = 'medilink_access_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
        }
      },
      onAuthFailure: () => {
        useAuthStore.getState().clearAuth()
        document.cookie = 'medilink_access_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
        window.location.href = '/login'
      },
    })
  }, [])

  return null
}

function ThemeController() {
  const { theme, setTheme } = useUIStore()

  useEffect(() => {
    if (typeof window === 'undefined') return

    const storedTheme = window.localStorage.getItem('medilink-theme')
    if (storedTheme === 'light' || storedTheme === 'dark') {
      setTheme(storedTheme)
      return
    }

    const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    setTheme(systemTheme)
  }, [setTheme])

  useEffect(() => {
    if (typeof window === 'undefined') return
    document.documentElement.setAttribute('data-theme', theme)
    window.localStorage.setItem('medilink-theme', theme)
  }, [theme])

  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <MSWProvider>
      <QueryClientProvider client={queryClient}>
        <AuthClientSetup />
        <ThemeController />
        {children}
        <Toaster
          position="bottom-right"
          toastOptions={{
            style: {
              background: 'var(--color-bg-elevated)',
              color: 'var(--color-text-primary)',
              border: '1px solid var(--color-border)',
              borderRadius: 'var(--radius)',
            },
          }}
        />
        <CommandPalette />
      </QueryClientProvider>
    </MSWProvider>
  )
}
