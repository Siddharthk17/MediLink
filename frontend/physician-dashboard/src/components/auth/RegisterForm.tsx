'use client'

import { useState } from 'react'
import { motion } from 'framer-motion'
import { Eye, EyeOff, Mail, Lock, User, Phone, Hash, Stethoscope, Sun, Moon, Shield, Activity, CheckCircle } from 'lucide-react'
import Link from 'next/link'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { useUIStore } from '@/store/uiStore'
import { authAPI } from '@medilink/shared'
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
  animate: { transition: { staggerChildren: 0.04, delayChildren: 0.15 } },
}

const fadeUp = {
  initial: { opacity: 0, y: 5 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.25, ease: 'easeOut' } },
}

export function RegisterForm() {
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [phone, setPhone] = useState('')
  const [mciNumber, setMciNumber] = useState('')
  const [specialization, setSpecialization] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const { theme, toggleTheme } = useUIStore()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    setLoading(true)
    try {
      await authAPI.registerPhysician({
        fullName,
        email,
        password,
        phone: phone || undefined,
        mciNumber,
        specialization: specialization || undefined,
      })
      setSuccess(true)
      toast.success('Registration submitted successfully')
    } catch (err) {
      setError(parseAPIError(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[var(--color-bg-base)] py-12">
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
        {/* Theme toggle */}
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
                Physician Registration
              </p>
            </motion.div>

            <motion.div variants={fadeUp} className="mx-auto mb-7 h-px w-12 bg-[var(--color-border)]" />

            {success ? (
              <motion.div variants={fadeUp} className="text-center space-y-4 py-4">
                <div className="inline-flex items-center justify-center w-14 h-14 rounded-full mb-2" style={{ background: 'var(--color-success-subtle)' }}>
                  <CheckCircle size={28} style={{ color: 'var(--color-success)' }} />
                </div>
                <h2 className="text-sm font-medium text-[var(--color-text-primary)]">Registration submitted</h2>
                <p className="text-xs text-[var(--color-text-muted)] leading-relaxed max-w-[260px] mx-auto">
                  Your account is pending admin approval. You&apos;ll receive an email once approved.
                </p>
                <Link
                  href="/login"
                  className="inline-block text-xs text-[var(--color-accent)] hover:text-[var(--color-text-primary)] transition-colors mt-2"
                >
                  ← Back to sign in
                </Link>
              </motion.div>
            ) : (
              <motion.form variants={fadeUp} onSubmit={handleSubmit} className="space-y-2.5">
                <Input
                  type="text"
                  placeholder="Full name"
                  value={fullName}
                  onChange={(e) => { setFullName(e.target.value); if (error) setError('') }}
                  icon={<User size={15} />}
                  required
                  autoComplete="name"
                  aria-label="Full name"
                />
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

                <div className="grid grid-cols-2 gap-2">
                  <Input
                    type="text"
                    placeholder="MCI number"
                    value={mciNumber}
                    onChange={(e) => { setMciNumber(e.target.value); if (error) setError('') }}
                    icon={<Hash size={15} />}
                    required
                    aria-label="MCI registration number"
                  />
                  <Input
                    type="text"
                    placeholder="Specialization"
                    value={specialization}
                    onChange={(e) => setSpecialization(e.target.value)}
                    icon={<Stethoscope size={15} />}
                    aria-label="Specialization"
                  />
                </div>

                <Input
                  type="tel"
                  placeholder="Phone (optional)"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  icon={<Phone size={15} />}
                  autoComplete="tel"
                  aria-label="Phone number"
                />

                <div className="relative">
                  <Input
                    type={showPassword ? 'text' : 'password'}
                    placeholder="Password"
                    value={password}
                    onChange={(e) => { setPassword(e.target.value); if (error) setError('') }}
                    icon={<Lock size={15} />}
                    required
                    autoComplete="new-password"
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

                <Input
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Confirm password"
                  value={confirmPassword}
                  onChange={(e) => { setConfirmPassword(e.target.value); if (error) setError('') }}
                  icon={<Lock size={15} />}
                  required
                  autoComplete="new-password"
                  aria-label="Confirm password"
                />

                <p className="text-[10px] text-[var(--color-text-muted)] text-left pt-0.5">
                  Min 8 chars · uppercase · lowercase · digit · special character
                </p>

                {error && (
                  <p className="text-xs text-center text-[var(--color-danger)]" role="alert">
                    {error}
                  </p>
                )}

                <div className="pt-1">
                  <Button type="submit" className="w-full" loading={loading} size="lg">
                    Create account
                  </Button>
                </div>
              </motion.form>
            )}

            {!success && (
              <motion.p variants={fadeUp} className="mt-6 text-xs text-[var(--color-text-muted)] text-center">
                Already have an account?{' '}
                <Link
                  href="/login"
                  className="text-[var(--color-accent)] hover:text-[var(--color-text-primary)] transition-colors"
                >
                  Sign in
                </Link>
              </motion.p>
            )}
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
