import { create } from 'zustand'

interface UIState {
  sidebarExpanded: boolean
  sidebarPinned: boolean
  notificationDrawerOpen: boolean
  commandPaletteOpen: boolean
  activePatientId: string | null
  theme: 'light' | 'dark'
  toggleSidebar: () => void
  setSidebarPinned: (pinned: boolean) => void
  toggleNotifications: () => void
  toggleCommandPalette: () => void
  setActivePatient: (id: string | null) => void
  setTheme: (theme: 'light' | 'dark') => void
  toggleTheme: () => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarExpanded: false,
  sidebarPinned: false,
  notificationDrawerOpen: false,
  commandPaletteOpen: false,
  activePatientId: null,
  theme: 'light',
  toggleSidebar: () => set((s) => ({ sidebarExpanded: !s.sidebarExpanded })),
  setSidebarPinned: (pinned) => set({ sidebarPinned: pinned, sidebarExpanded: pinned }),
  toggleNotifications: () => set((s) => ({ notificationDrawerOpen: !s.notificationDrawerOpen })),
  toggleCommandPalette: () => set((s) => ({ commandPaletteOpen: !s.commandPaletteOpen })),
  setActivePatient: (id) => set({ activePatientId: id }),
  setTheme: (theme) => set({ theme }),
  toggleTheme: () => set((s) => ({ theme: s.theme === 'light' ? 'dark' : 'light' })),
}))
