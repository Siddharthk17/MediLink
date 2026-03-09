'use client'

import { Drawer } from '@/components/ui/Drawer'
import { useUIStore } from '@/store/uiStore'
import { Bell } from 'lucide-react'

export function NotificationDrawer() {
  const { notificationDrawerOpen, toggleNotifications } = useUIStore()

  return (
    <Drawer open={notificationDrawerOpen} onClose={toggleNotifications} title="Notifications" width={380}>
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <Bell className="w-10 h-10 text-[var(--color-text-muted)] mb-4" />
        <p className="text-sm font-medium text-[var(--color-text-primary)] mb-1">No notifications</p>
        <p className="text-xs text-[var(--color-text-muted)] max-w-[220px]">
          Notifications about lab results, consents, and documents will appear here
        </p>
      </div>
    </Drawer>
  )
}
