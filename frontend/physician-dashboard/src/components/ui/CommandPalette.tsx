'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { motion, AnimatePresence } from 'framer-motion'
import { Search, User, FileText, Pill, FlaskConical, Settings } from 'lucide-react'
import { useUIStore } from '@/store/uiStore'
import { cn } from '@/lib/utils'

interface CommandItem {
  id: string
  label: string
  description?: string
  icon: React.ElementType
  action: () => void
  keywords: string[]
}

export function CommandPalette() {
  const router = useRouter()
  const { commandPaletteOpen, toggleCommandPalette } = useUIStore()
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const commands: CommandItem[] = [
    { id: 'dashboard', label: 'Dashboard', description: 'Go to home', icon: Settings, action: () => router.push('/dashboard'), keywords: ['home', 'main'] },
    { id: 'patients', label: 'Patient List', description: 'Browse consented patients', icon: User, action: () => router.push('/patients'), keywords: ['patient', 'list', 'browse'] },
    { id: 'search', label: 'Search Records', description: 'Search across FHIR resources', icon: Search, action: () => router.push('/search'), keywords: ['search', 'find', 'query'] },
    { id: 'consents', label: 'Consent Management', description: 'View consent status', icon: FileText, action: () => router.push('/consents'), keywords: ['consent', 'permission'] },
    { id: 'notifications', label: 'Notifications', description: 'View all notifications', icon: FlaskConical, action: () => router.push('/notifications'), keywords: ['notification', 'alert'] },
    { id: 'admin', label: 'Admin Panel', description: 'System administration', icon: Settings, action: () => router.push('/admin'), keywords: ['admin', 'system', 'manage'] },
  ]

  const filteredCommands = query.length === 0
    ? commands
    : commands.filter(
        (c) =>
          c.label.toLowerCase().includes(query.toLowerCase()) ||
          c.keywords.some((k) => k.includes(query.toLowerCase()))
      )

  useEffect(() => {
    if (commandPaletteOpen) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [commandPaletteOpen])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        toggleCommandPalette()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [toggleCommandPalette])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((i) => Math.min(i + 1, filteredCommands.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter' && filteredCommands[selectedIndex]) {
      filteredCommands[selectedIndex].action()
      toggleCommandPalette()
    } else if (e.key === 'Escape') {
      toggleCommandPalette()
    }
  }, [filteredCommands, selectedIndex, toggleCommandPalette])

  return (
    <AnimatePresence>
      {commandPaletteOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 backdrop-blur-sm"
            style={{ background: 'var(--color-bg-overlay)' }}
            onClick={toggleCommandPalette}
          />
          {/* Panel */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: -20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: -20 }}
            transition={{ duration: 0.15 }}
            className="fixed inset-x-0 top-[20vh] z-50 mx-auto w-full max-w-lg"
          >
            <div className="glass rounded-[20px] border border-[var(--color-border)] shadow-elevated overflow-hidden">
              {/* Search input */}
              <div className="flex items-center gap-3 px-4 border-b border-[var(--color-border)]">
                <Search size={18} style={{ color: 'var(--color-text-muted)' }} />
                <input
                  ref={inputRef}
                  type="text"
                  value={query}
                  onChange={(e) => { setQuery(e.target.value); setSelectedIndex(0) }}
                  onKeyDown={handleKeyDown}
                  placeholder="Type a command or search..."
                  className="flex-1 py-4 bg-transparent text-sm outline-none"
                  style={{ color: 'var(--color-text-primary)' }}
                />
                <kbd className="px-1.5 py-0.5 text-[10px] font-mono rounded border border-[var(--color-border)] text-[var(--color-text-muted)]">
                  ESC
                </kbd>
              </div>
              {/* Results */}
              <div className="max-h-72 overflow-y-auto p-2">
                {filteredCommands.length === 0 ? (
                  <p className="p-4 text-center text-sm" style={{ color: 'var(--color-text-muted)' }}>
                    No results found
                  </p>
                ) : (
                  filteredCommands.map((cmd, i) => {
                    const Icon = cmd.icon
                    return (
                      <button
                        key={cmd.id}
                        onClick={() => { cmd.action(); toggleCommandPalette() }}
                        className={cn(
                          'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors',
                          i === selectedIndex ? 'bg-[var(--color-accent-subtle)]' : 'hover:bg-[var(--color-bg-elevated)]'
                        )}
                      >
                        <Icon size={18} style={{ color: i === selectedIndex ? 'var(--color-accent)' : 'var(--color-text-muted)' }} />
                        <div>
                          <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>{cmd.label}</p>
                          {cmd.description && (
                            <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{cmd.description}</p>
                          )}
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
