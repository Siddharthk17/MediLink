'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { motion } from 'framer-motion'
import { Heart, Mail, Lock, User, Phone, Calendar, ArrowRight } from 'lucide-react'
import { authAPI } from '@medilink/shared'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import Link from 'next/link'

export default function RegisterPage() {
  const router = useRouter()
  const [form, setForm] = useState({
    fullName: '',
    email: '',
    password: '',
    confirmPassword: '',
    phone: '',
    dateOfBirth: '',
    gender: 'male',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const update = (field: string) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setForm((prev) => ({ ...prev, [field]: e.target.value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (form.password !== form.confirmPassword) {
      setError('Passwords do not match.')
      return
    }

    if (form.password.length < 8) {
      setError('Password must be at least 8 characters.')
      return
    }

    setLoading(true)
    try {
      await authAPI.registerPatient({
        fullName: form.fullName,
        email: form.email,
        password: form.password,
        phone: form.phone || undefined,
        dateOfBirth: form.dateOfBirth,
        gender: form.gender,
      })

      router.push('/login?registered=true')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { issue?: Array<{ diagnostics?: string }> } } })
        ?.response?.data?.issue?.[0]?.diagnostics
      setError(msg || 'Registration failed. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg-base)] px-4 py-12">
      <div className="mesh-bg absolute inset-0 z-0 pointer-events-none opacity-60" />

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
        className="relative z-10 w-full max-w-md"
      >
        <div className="text-center mb-8">
          <div className="w-14 h-14 bg-[var(--color-accent)] rounded-2xl flex items-center justify-center mx-auto mb-4">
            <Heart className="w-7 h-7 text-white" />
          </div>
          <h1 className="font-display text-4xl tracking-tight text-gradient">
            Create account
          </h1>
          <p className="mt-2 text-[var(--color-text-muted)]">
            Get started with your personal health portal
          </p>
        </div>

        <div className="glass-panel rounded-3xl p-8 shadow-card">
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              id="fullName"
              label="Full Name"
              placeholder="Your full name"
              value={form.fullName}
              onChange={update('fullName')}
              icon={<User className="w-4 h-4" />}
              required
            />

            <Input
              id="email"
              type="email"
              label="Email"
              placeholder="you@example.com"
              value={form.email}
              onChange={update('email')}
              icon={<Mail className="w-4 h-4" />}
              required
              autoComplete="email"
            />

            <div className="grid grid-cols-2 gap-3">
              <Input
                id="dateOfBirth"
                type="date"
                label="Date of Birth"
                value={form.dateOfBirth}
                onChange={update('dateOfBirth')}
                icon={<Calendar className="w-4 h-4" />}
                required
              />
              <div className="space-y-1.5">
                <label htmlFor="gender" className="block text-sm font-medium text-[var(--color-text-secondary)]">
                  Gender
                </label>
                <select
                  id="gender"
                  value={form.gender}
                  onChange={update('gender')}
                  className="w-full h-11 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] px-4 text-sm text-[var(--color-text-primary)] transition-colors hover:border-[var(--color-border-hover)] focus:outline-none focus:border-[var(--color-border-focus)] focus:ring-2 focus:ring-[var(--color-accent-subtle)]"
                >
                  <option value="male">Male</option>
                  <option value="female">Female</option>
                  <option value="other">Other</option>
                </select>
              </div>
            </div>

            <Input
              id="phone"
              type="tel"
              label="Phone (optional)"
              placeholder="+91 98765 43210"
              value={form.phone}
              onChange={update('phone')}
              icon={<Phone className="w-4 h-4" />}
            />

            <Input
              id="password"
              type="password"
              label="Password"
              placeholder="Min 8 characters"
              value={form.password}
              onChange={update('password')}
              icon={<Lock className="w-4 h-4" />}
              required
              autoComplete="new-password"
            />

            <Input
              id="confirmPassword"
              type="password"
              label="Confirm Password"
              placeholder="Repeat password"
              value={form.confirmPassword}
              onChange={update('confirmPassword')}
              icon={<Lock className="w-4 h-4" />}
              required
              autoComplete="new-password"
            />

            {error && (
              <motion.p
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                className="text-sm text-[var(--color-danger)] bg-[var(--color-danger-subtle)] rounded-xl px-4 py-2.5"
              >
                {error}
              </motion.p>
            )}

            <Button
              type="submit"
              disabled={loading}
              className="w-full"
              size="lg"
            >
              {loading ? 'Creating account…' : 'Create account'}
              {!loading && <ArrowRight className="w-4 h-4 ml-1" />}
            </Button>
          </form>
        </div>

        <p className="text-center mt-6 text-sm text-[var(--color-text-muted)]">
          Already have an account?{' '}
          <Link href="/login" className="text-[var(--color-accent)] hover:underline font-medium">
            Sign in
          </Link>
        </p>
      </motion.div>
    </div>
  )
}
