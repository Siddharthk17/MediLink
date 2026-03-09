'use client'

import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Eye, EyeOff, Mail, Lock, Sun, Moon, Shield, Activity } from 'lucide-react'
import Link from 'next/link'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { TOTPInput } from './TOTPInput'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { authAPI, apiClient } from '@medilink/shared'
import { parseAPIError } from '@medilink/shared'
import toast from 'react-hot-toast'

const cardReveal = {
  initial: { opacity: 0, y: 12, scale: 0.98 },
  animate: {
    opacity: 1, y: 0, scale: 1,
    transition: { duration: 0.5, ease: [0.23, 1, 0.32, 1] },
  },
}

const stagger = {
  animate: { transition: { staggerChildren: 0.06, delayChildren: 0.15 } },
}

const fadeUp = {
  initial: { opacity: 0, y: 5 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.25, ease: 'easeOut' } },
  exit: { opacity: 0, y: -5, transition: { duration: 0.15 } },
}

export function LoginForm() {
  const [step, setStep] = useState<'credentials' | 'totp'>('credentials')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [partialToken, setPartialToken] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [totpError, setTotpError] = useState(false)
  const { theme, toggleTheme } = useUIStore()
  const { setAuth, clearAuth } = useAuthStore()

  const fetchAndSetUser = async (accessToken: string, refreshToken: string) => {
    document.cookie = `medilink_access_token=${accessToken}; path=/; samesite=strict`
    apiClient.defaults.headers.common['Authorization'] = `Bearer ${accessToken}`
    const meRes = await authAPI.getMe()
    const profile = meRes.data
    setAuth(
      {
        id: profile.id,
        role: profile.role as 'physician' | 'patient' | 'admin' | 'researcher',
        status: profile.status as 'active' | 'pending' | 'suspended',
        fullName: profile.fullName,
        fhirPatientId: profile.fhirPatientId,
        totpEnabled: profile.totpEnabled,
      },
      { accessToken, refreshToken }
    )
    window.location.href = '/dashboard'
  }

  const handleCredentialSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      clearAuth()
      const res = await authAPI.loginPhysician(email, password)
      const data = res.data
      if (data.requiresTOTP) {
        // Backend returns accessToken as partial token for TOTP step
        setPartialToken(data.accessToken)
        apiClient.defaults.headers.common['Authorization'] = `Bearer ${data.accessToken}`
        setStep('totp')
      } else if (data.accessToken) {
        await fetchAndSetUser(data.accessToken, data.refreshToken || '')
      } else {
        setError('Unexpected server response. Please try again.')
      }
    } catch (err) {
      setError(parseAPIError(err))
    } finally {
      setLoading(false)
    }
  }

  const handleTOTPComplete = async (code: string) => {
    setTotpError(false)
    setLoading(true)
    try {
      apiClient.defaults.headers.common['Authorization'] = `Bearer ${partialToken}`
      const res = await authAPI.verifyTOTP(code)
      const data = res.data
      if (data.accessToken) {
        await fetchAndSetUser(data.accessToken, data.refreshToken || '')
      }
    } catch {
      setTotpError(true)
      toast.error('Invalid verification code. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[var(--color-bg-base)]">
      <div className="ambient-mesh absolute inset-0 pointer-events-none" />
      <div
        className="absolute inset-0 pointer-events-none opacity-[0.35]"
        style={{
          backgroundImage:
            'linear-gradient(var(--color-border-subtle) 1px, transparent 1px), linear-gradient(90deg, var(--color-border-subtle) 1px, transparent 1px)',
          backgroundSize: '64px 64px',
        }}
      />

      <div className="relative z-10 w-full max-w-[420px] mx-4">
        {/* Theme toggle — floats above the card */}
        <div className="flex justify-end mb-4">
          <button
            type="button"
            onClick={toggleTheme}
            className="h-9 w-9 rounded-full border border-[var(--color-border)] glass flex items-center justify-center text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:border-[var(--color-border-hover)] transition-colors"
            aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} theme`}
          >
            {theme === 'light' ? <Moon size={15} /> : <Sun size={15} />}
          </button>
        </div>

        {/* Card */}
        <motion.div
          variants={cardReveal}
          initial="initial"
          animate="animate"
          className="auth-card glass-panel rounded-[20px] overflow-hidden"
          style={{ boxShadow: 'var(--shadow-card)' }}
        >
          {/* Accent strip */}
          <div
            className="h-[2px]"
            style={{
              background: 'linear-gradient(90deg, transparent, var(--color-accent), transparent)',
              opacity: 0.6,
            }}
          />

          <motion.div
            className="px-8 pt-10 pb-8 sm:px-10"
            variants={stagger}
            initial="initial"
            animate="animate"
          >
            {/* Brand */}
            <motion.div variants={fadeUp} className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-2xl mb-4" style={{ background: 'var(--color-accent-subtle)' }}>
                <Activity size={22} style={{ color: 'var(--color-accent)' }} />
              </div>
              <h1 className="font-display text-[28px] text-[var(--color-text-primary)] leading-none">
                MediLink
              </h1>
              <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--color-text-muted)] mt-2">
                Physician Portal
              </p>
            </motion.div>

            {/* Divider */}
            <motion.div variants={fadeUp} className="mx-auto mb-7 h-px w-12 bg-[var(--color-border)]" />

            <AnimatePresence mode="wait">
              {step === 'credentials' ? (
                <motion.form
                  key="credentials"
                  variants={fadeUp}
                  initial="initial"
                  animate="animate"
                  exit="exit"
                  onSubmit={handleCredentialSubmit}
                  className="space-y-3"
                >
                  <Input
                    type="email"
                    placeholder="Email address"
                    value={email}
                    onChange={(e) => { setEmail(e.target.value); if (error) setError('') }}
                    icon={<Mail size={15} />}
                    required
                    autoComplete="email"
                    aria-label="Email address"
                  />
                  <div className="relative">
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      placeholder="Password"
                      value={password}
                      onChange={(e) => { setPassword(e.target.value); if (error) setError('') }}
                      icon={<Lock size={15} />}
                      required
                      autoComplete="current-password"
                      aria-label="Password"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors duration-150"
                      aria-label={showPassword ? 'Hide password' : 'Show password'}
                    >
                      {showPassword ? <EyeOff size={15} /> : <Eye size={15} />}
                    </button>
                  </div>

                  {error && (
                    <p className="text-xs text-center text-[var(--color-danger)]" role="alert">
                      {error}
                    </p>
                  )}

                  <div className="pt-1">
                    <Button type="submit" className="w-full" loading={loading} size="lg">
                      Sign in
                    </Button>
                  </div>
                </motion.form>
              ) : (
                <motion.div
                  key="totp"
                  variants={fadeUp}
                  initial="initial"
                  animate="animate"
                  exit="exit"
                  className="space-y-4"
                >
                  <h2 className="text-sm font-medium text-[var(--color-text-primary)] text-center">
                    Two-factor authentication
                  </h2>
                  <p className="text-xs text-[var(--color-text-muted)] text-center">
                    Open your authenticator app and enter the 6-digit code
                  </p>
                  <TOTPInput onComplete={handleTOTPComplete} error={totpError} disabled={loading} />
                  <button
                    type="button"
                    onClick={() => { setStep('credentials'); setTotpError(false) }}
                    className="block mx-auto text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors duration-150"
                  >
                    Back to login
                  </button>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Register link */}
            <motion.p variants={fadeUp} className="mt-6 text-xs text-[var(--color-text-muted)] text-center">
              Don&apos;t have an account?{' '}
              <Link
                href="/register"
                className="text-[var(--color-accent)] hover:text-[var(--color-text-primary)] transition-colors"
              >
                Register
              </Link>
            </motion.p>
          </motion.div>

          {/* Card footer */}
          <div
            className="flex items-center justify-center gap-4 px-8 py-3 text-center"
            style={{ borderTop: '1px solid var(--color-border-subtle)', background: 'var(--color-bg-elevated)', opacity: 0.8 }}
          >
            <span className="inline-flex items-center gap-1.5 font-mono text-[10px] text-[var(--color-text-muted)]">
              <Shield size={10} /> AES-256 encrypted
            </span>
            <span className="h-2.5 w-px bg-[var(--color-border)]" />
            <span className="font-mono text-[10px] text-[var(--color-text-muted)]">
              FHIR R4 compliant
            </span>
          </div>
        </motion.div>
      </div>
    </div>
  )
}
