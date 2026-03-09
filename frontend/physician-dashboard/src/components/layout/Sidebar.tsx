'use client'

import { useRef, useCallback } from 'react'
import { motion } from 'framer-motion'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  LayoutDashboard, Users, Shield, Search,
  Bell, Settings, FileText, UserCog,
  ChevronLeft, ChevronRight, LogOut
} from 'lucide-react'
import { sidebarVariants } from '@/lib/motion'
import { useUIStore } from '@/store/uiStore'
import { useAuthStore } from '@/store/authStore'
import { cn } from '@/lib/utils'
import { Tooltip } from '@/components/ui/Tooltip'

interface NavItem {
  href: string
  icon: React.ElementType
  label: string
  adminOnly?: boolean
}

const mainNav: NavItem[] = [
  { href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { href: '/patients', icon: Users, label: 'Patients' },
  { href: '/consents', icon: Shield, label: 'Consents' },
  { href: '/search', icon: Search, label: 'Search' },
]

const adminNav: NavItem[] = [
  { href: '/admin', icon: Settings, label: 'Admin', adminOnly: true },
  { href: '/admin/audit-logs', icon: FileText, label: 'Audit Logs', adminOnly: true },
  { href: '/admin/users', icon: UserCog, label: 'Users', adminOnly: true },
]

export function Sidebar() {
  const pathname = usePathname()
  const { sidebarExpanded, sidebarPinned, setSidebarPinned } = useUIStore()
  const { user, clearAuth, hasRole } = useAuthStore()
  const hoverTimeout = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isAdmin = hasRole('admin')

  const handleMouseEnter = useCallback(() => {
    if (sidebarPinned) return
    hoverTimeout.current = setTimeout(() => {
      useUIStore.setState({ sidebarExpanded: true })
    }, 220)
  }, [sidebarPinned])

  const handleMouseLeave = useCallback(() => {
    if (sidebarPinned) return
    if (hoverTimeout.current) {
      clearTimeout(hoverTimeout.current)
      hoverTimeout.current = null
    }
    useUIStore.setState({ sidebarExpanded: false })
  }, [sidebarPinned])

  const handlePinToggle = () => {
    setSidebarPinned(!sidebarPinned)
  }

  const handleLogout = () => {
    clearAuth()
    document.cookie = 'medilink_access_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
    window.location.href = '/login'
  }

  const renderNavItem = (item: NavItem) => {
    const isActive = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href + '/'))
    const Icon = item.icon

    const link = (
      <Link
        href={item.href}
        className={cn(
          'group relative flex items-center h-10 rounded-[var(--radius)] transition-all',
          sidebarExpanded ? 'gap-3 px-3.5' : 'justify-center px-0',
          isActive
            ? 'text-[var(--color-text-primary)] bg-[var(--color-accent-subtle)]'
            : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]'
        )}
      >
        {isActive && (
          <span className="absolute left-1.5 top-1/2 -translate-y-1/2 h-5 w-1 rounded-full bg-[var(--color-accent)]" />
        )}
        <Icon size={17} className="shrink-0" />
        {sidebarExpanded && (
          <motion.span
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.12 }}
            className="text-[13px] whitespace-nowrap"
          >
            {item.label}
          </motion.span>
        )}
      </Link>
    )

    if (!sidebarExpanded) {
      return (
        <Tooltip key={item.href} content={item.label} side="right">
          {link}
        </Tooltip>
      )
    }

    return <div key={item.href}>{link}</div>
  }

  return (
    <motion.nav
      className="glass fixed left-4 top-4 bottom-4 z-40 flex flex-col border border-[var(--color-border)] rounded-[28px] shadow-card overflow-hidden"
      variants={sidebarVariants}
      animate={sidebarExpanded ? 'expanded' : 'collapsed'}
      initial="collapsed"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      aria-label="Main navigation"
    >
      <div className={cn(
        'flex items-center h-14 shrink-0 border-b border-[var(--color-border-subtle)]',
        sidebarExpanded ? 'px-4' : 'justify-center'
      )}>
        <span className="font-display text-xl leading-none text-[var(--color-accent)]">M</span>
        {sidebarExpanded && (
          <motion.span
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.1 }}
            className="ml-2 font-semibold tracking-tight text-[15px] leading-none text-[var(--color-text-primary)]"
          >
            MediLink
          </motion.span>
        )}
      </div>

      <div className={cn(
        'flex-1 py-3 space-y-1 overflow-y-auto',
        sidebarExpanded ? 'px-2' : 'px-1.5'
      )}>
        {mainNav.map(renderNavItem)}
        {isAdmin && (
          <>
            <div className="my-2 mx-2 h-px bg-[var(--color-border-subtle)]" />
            {adminNav.map(renderNavItem)}
          </>
        )}
      </div>

      <div className={cn(
        'py-2 space-y-1 border-t border-[var(--color-border-subtle)]',
        sidebarExpanded ? 'px-2' : 'px-1.5'
      )}>
        {renderNavItem({ href: '/notifications', icon: Bell, label: 'Notifications' })}

        {sidebarExpanded && (
          <button
            onClick={handlePinToggle}
            className="flex items-center gap-3 px-3.5 h-10 w-full rounded-[var(--radius)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
            aria-label={sidebarPinned ? 'Unpin sidebar' : 'Pin sidebar'}
          >
            {sidebarPinned ? <ChevronLeft size={17} /> : <ChevronRight size={17} />}
            <span className="text-[13px]">{sidebarPinned ? 'Unpin' : 'Pin sidebar'}</span>
          </button>
        )}

        <div className={cn(
          'flex items-center py-2',
          sidebarExpanded ? 'gap-3 px-3.5' : 'justify-center'
        )}>
          <span className="text-sm font-semibold shrink-0 text-[var(--color-accent)]">
            {user?.fullName?.[0]?.toUpperCase() || 'D'}
          </span>
          {sidebarExpanded && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.12 }}
              className="flex-1 min-w-0"
            >
              <p className="text-[13px] truncate text-[var(--color-text-primary)]">
                {user?.fullName || 'Physician'}
              </p>
              <button
                onClick={handleLogout}
                className="flex items-center gap-1 text-[11px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
              >
                <LogOut size={10} /> Sign out
              </button>
            </motion.div>
          )}
        </div>
      </div>
    </motion.nav>
  )
}
