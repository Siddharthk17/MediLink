'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { motion, AnimatePresence } from 'framer-motion'
import { Eye, EyeOff, Mail, Lock, Sun, Moon, Shield, Heart } from 'lucide-react'
import { authAPI, apiClient } from '@medilink/shared'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { TOTPInput } from '@/components/auth/TOTPInput'
import Link from 'next/link'

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

export default function LoginPage() {
  const router = useRouter()
  const { setAuth } = useAuthStore()
  const { theme, toggleTheme } = useUIStore()
  const [step, setStep] = useState<'credentials' | 'totp'>('credentials')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [partialToken, setPartialToken] = useState('')
  const [error, setError] = useState('')
  const [totpError, setTotpError] = useState(false)
  const [loading, setLoading] = useState(false)
  const [mounted, setMounted] = useState(false)

  useEffect(() => setMounted(true), [])

  const fetchAndSetUser = async (accessToken: string, refreshToken: string) => {
    document.cookie = `medilink_patient_token=${accessToken}; path=/; samesite=strict`
    apiClient.defaults.headers.common['Authorization'] = `Bearer ${accessToken}`

    const meRes = await authAPI.getMe()
    const me = meRes.data

    if (me.role !== 'patient') {
      document.cookie = 'medilink_patient_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
      const physicianUrl = process.env.NEXT_PUBLIC_PHYSICIAN_PORTAL_URL || `${window.location.protocol}//${window.location.hostname}:3000`
      window.location.href = `${physicianUrl}/dashboard`
      return
    }

    setAuth(
      {
        id: me.id,
        role: me.role as 'patient',
        fullName: me.fullName,
        status: me.status as 'active' | 'pending' | 'suspended',
        totpEnabled: me.totpEnabled,
        email: me.email,
        fhirPatientId: me.fhirPatientId,
        phone: me.phone,
      },
      { accessToken, refreshToken }
    )

    router.push('/dashboard')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const { data } = await authAPI.login(email, password)

      if (data.requiresTOTP) {
        setPartialToken(data.accessToken)
        apiClient.defaults.headers.common['Authorization'] = `Bearer ${data.accessToken}`
        setStep('totp')
        setLoading(false)
        return
      }

      if (!data.accessToken || !data.refreshToken) {
        setError('Login failed. Please try again.')
        setLoading(false)
        return
      }

      await fetchAndSetUser(data.accessToken, data.refreshToken)
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { issue?: Array<{ diagnostics?: string }> } } })
        ?.response?.data?.issue?.[0]?.diagnostics
      setError(msg || 'Invalid email or password.')
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
      setError('Invalid verification code. Please try again.')
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

        <motion.div
          variants={cardReveal}
          initial="initial"
          animate="animate"
          className="glass-panel rounded-[20px] overflow-hidden"
          style={{ boxShadow: 'var(--shadow-card)' }}
        >
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
            <motion.div variants={fadeUp} className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-2xl mb-4" style={{ background: 'var(--color-accent-subtle)' }}>
                <Heart size={22} style={{ color: 'var(--color-accent)' }} />
              </div>
              <h1 className="font-display text-[28px] text-[var(--color-text-primary)] leading-none">
                MediLink
              </h1>
              <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--color-text-muted)] mt-2">
                Patient Portal
              </p>
            </motion.div>

            <motion.div variants={fadeUp} className="mx-auto mb-7 h-px w-12 bg-[var(--color-border)]" />

            <AnimatePresence mode="wait">
              {step === 'credentials' ? (
                <motion.form
                  key="credentials"
                  variants={fadeUp}
                  initial="initial"
                  animate="animate"
                  exit="exit"
                  onSubmit={handleSubmit}
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
                    <Button type="submit" className="w-full" disabled={loading} size="lg">
                      {loading ? 'Signing in…' : 'Sign in'}
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
                  {error && (
                    <p className="text-xs text-center text-[var(--color-danger)]" role="alert">
                      {error}
                    </p>
                  )}
                  <button
                    type="button"
                    onClick={() => { setStep('credentials'); setTotpError(false); setError('') }}
                    className="block mx-auto text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors duration-150"
                  >
                    Back to login
                  </button>
                </motion.div>
              )}
            </AnimatePresence>

            <motion.p variants={fadeUp} className="mt-6 text-xs text-[var(--color-text-muted)] text-center">
              Don&apos;t have an account?{' '}
              <Link
                href="/register"
                className="text-[var(--color-accent)] hover:text-[var(--color-text-primary)] transition-colors"
              >
                Register
              </Link>
            </motion.p>
            <motion.p variants={fadeUp} className="mt-3 text-center">
              <a
                href={mounted ? `${process.env.NEXT_PUBLIC_PHYSICIAN_PORTAL_URL || `${window.location.protocol}//${window.location.hostname}:3000`}/login` : '/login'}
                className="text-sm font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
              >
                Physician Portal →
              </a>
            </motion.p>
          </motion.div>

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
