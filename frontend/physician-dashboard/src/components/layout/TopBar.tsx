'use client'

import { useState, useRef, useEffect } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { motion } from 'framer-motion'
import { Bell, Search, Sparkles, Moon, Sun, LogOut, User } from 'lucide-react'
import { useUIStore } from '@/store/uiStore'
import { useAuthStore } from '@/store/authStore'
import { Magnetic } from '@/components/aura/Interactions'
import { cn } from '@/lib/utils'

const baseNavItems = [
  { href: '/dashboard', label: 'Overview' },
  { href: '/patients', label: 'Patients' },
  { href: '/consents', label: 'Consents' },
  { href: '/search', label: 'Search' },
  { href: '/notifications', label: 'Notifications' },
]

export function TopBar() {
  const pathname = usePathname()
  const { toggleNotifications, toggleCommandPalette, theme, toggleTheme } = useUIStore()
  const { user, hasRole, clearAuth } = useAuthStore()
  const [mounted, setMounted] = useState(false)
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => setMounted(true), [])

  const isAdmin = mounted && hasRole('admin')
  const navItems = isAdmin
    ? [...baseNavItems, { href: '/admin', label: 'Admin' }]
    : baseNavItems

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleLogout = () => {
    clearAuth()
    document.cookie = 'medilink_access_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
    window.location.href = '/login'
  }

  return (
    <header className="fixed top-6 left-1/2 -translate-x-1/2 z-40 w-[95%] max-w-7xl">
      <motion.div
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="glass-panel rounded-full px-6 py-4 flex items-center justify-between shadow-[0_8px_32px_rgba(0,0,0,0.06)]"
      >
        <div className="flex items-center gap-10">
          <Magnetic>
            <Link href="/dashboard" className="font-semibold tracking-tight text-lg cursor-pointer flex items-center gap-2">
              <div className="w-6 h-6 bg-[var(--color-text-primary)] rounded-full flex items-center justify-center">
                <Sparkles className="w-3 h-3 text-[var(--color-text-inverse)]" />
              </div>
              <span className="text-[var(--color-text-primary)]">MediLink</span>
            </Link>
          </Magnetic>

          <nav className="hidden md:flex items-center gap-7 text-sm font-medium text-[var(--color-text-muted)]">
            {navItems.map((item) => {
              const active = pathname === item.href || pathname.startsWith(`${item.href}/`)
              return (
                <Magnetic key={item.href}>
                  <Link
                    href={item.href}
                    className={cn(
                      'transition-colors block p-2',
                      active ? 'text-[var(--color-text-primary)]' : 'hover:text-[var(--color-text-primary)]'
                    )}
                  >
                    {item.label}
                  </Link>
                </Magnetic>
              )
            })}
          </nav>
        </div>

        <div className="flex items-center gap-3">
          <Magnetic>
            <button
              onClick={toggleTheme}
              className="w-10 h-10 rounded-full border border-[var(--color-border)] flex items-center justify-center hover:bg-[var(--color-bg-hover)] transition-colors"
              aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} theme`}
            >
              {theme === 'light' ? (
                <Moon className="w-4 h-4 text-[var(--color-text-primary)]" />
              ) : (
                <Sun className="w-4 h-4 text-[var(--color-text-primary)]" />
              )}
            </button>
          </Magnetic>

          <Magnetic>
            <button
              onClick={toggleCommandPalette}
              className="w-10 h-10 rounded-full border border-[var(--color-border)] flex items-center justify-center hover:bg-[var(--color-bg-hover)] transition-colors"
              aria-label="Open search command palette"
            >
              <Search className="w-4 h-4 text-[var(--color-text-primary)]" />
            </button>
          </Magnetic>

          <Magnetic>
            <button
              onClick={toggleNotifications}
              className="w-10 h-10 rounded-full border border-[var(--color-border)] flex items-center justify-center hover:bg-[var(--color-bg-hover)] transition-colors relative"
              aria-label="Open notifications"
            >
              <Bell className="w-4 h-4 text-[var(--color-text-primary)]" />
            </button>
          </Magnetic>

          <div className="relative" ref={menuRef}>
            <Magnetic>
              <button
                onClick={() => setUserMenuOpen(!userMenuOpen)}
                className="w-10 h-10 rounded-full overflow-hidden border border-[var(--color-border)] flex items-center justify-center text-sm font-semibold text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                aria-label="User menu"
                aria-expanded={userMenuOpen}
              >
                {mounted ? (user?.fullName?.[0]?.toUpperCase() || 'U') : 'U'}
              </button>
            </Magnetic>

            {userMenuOpen && (
              <motion.div
                initial={{ opacity: 0, y: 4, scale: 0.97 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ duration: 0.12 }}
                className="absolute right-0 top-full mt-2 w-56 rounded-2xl border border-[var(--color-border)] bg-[var(--color-bg-surface)] shadow-lg overflow-hidden z-50"
              >
                <div className="px-4 py-3 border-b border-[var(--color-border-subtle)]">
                  <p className="text-sm font-medium text-[var(--color-text-primary)] truncate">
                    {mounted ? (user?.fullName || 'User') : 'User'}
                  </p>
                  <p className="text-[11px] text-[var(--color-text-muted)] capitalize">
                    {mounted ? (user?.role || 'physician') : ''}
                  </p>
                </div>

                {isAdmin && (
                  <Link
                    href="/admin"
                    onClick={() => setUserMenuOpen(false)}
                    className="flex items-center gap-2.5 px-4 py-2.5 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                  >
                    <User size={15} />
                    Admin Panel
                  </Link>
                )}

                <Link
                  href="/profile"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2.5 px-4 py-2.5 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  <User size={15} />
                  My Profile
                </Link>

                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2.5 w-full px-4 py-2.5 text-sm text-[var(--color-danger)] hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  <LogOut size={15} />
                  Sign out
                </button>
              </motion.div>
            )}
          </div>
        </div>
      </motion.div>
    </header>
  )
}
