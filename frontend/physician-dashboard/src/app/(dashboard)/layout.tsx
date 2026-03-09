'use client'

import { TopBar } from '@/components/layout/TopBar'
import { NotificationDrawer } from '@/components/layout/NotificationDrawer'
import { ErrorBoundary } from '@/components/ui/ErrorBoundary'
import { AdvancedCursor } from '@/components/aura/Interactions'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative min-h-screen bg-[var(--color-bg-base)] overflow-hidden pb-12">
      <AdvancedCursor />
      <div className="mesh-bg absolute inset-0 z-0 pointer-events-none opacity-70" />
      <TopBar />
      <main className="relative z-10 pt-32 px-6">
        <ErrorBoundary>{children}</ErrorBoundary>
      </main>
      <NotificationDrawer />
    </div>
  )
}
