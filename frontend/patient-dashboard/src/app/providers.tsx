'use client'

import { QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import { queryClient } from '@/lib/queryClient'
import { useEffect, useState } from 'react'
import { configureClient } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'

function AuthClientSetup() {
  useEffect(() => {
    configureClient({
      baseURL: '/api',
      getTokens: () => {
        const { accessToken, refreshToken } = useAuthStore.getState()
        return { accessToken, refreshToken }
      },
      setTokens: (pair) => {
        if (pair) {
          useAuthStore.getState().setTokens(pair)
          document.cookie = `medilink_patient_token=${pair.accessToken}; path=/; samesite=strict`
        } else {
          useAuthStore.getState().setTokens(null)
          document.cookie = 'medilink_patient_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
        }
      },
      onAuthFailure: () => {
        useAuthStore.getState().clearAuth()
        document.cookie = 'medilink_patient_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
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

    const storedTheme = window.localStorage.getItem('medilink-patient-theme')
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
    window.localStorage.setItem('medilink-patient-theme', theme)
  }, [theme])

  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])

  return (
    <QueryClientProvider client={queryClient}>
      <AuthClientSetup />
      <ThemeController />
      {mounted && children}
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
    </QueryClientProvider>
  )
}
