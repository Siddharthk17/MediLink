import { beforeEach } from 'vitest'
import { useUIStore } from '@/store/uiStore'

describe('uiStore', () => {
  beforeEach(() => {
    // Reset to defaults
    const s = useUIStore.getState()
    s.setSidebarPinned(false)
    s.setActivePatient(null)
    s.setTheme('light')
    if (s.notificationDrawerOpen) s.toggleNotifications()
    if (s.commandPaletteOpen) s.toggleCommandPalette()
  })

  describe('initial state', () => {
    it('has correct defaults', () => {
      const state = useUIStore.getState()
      expect(state.sidebarExpanded).toBe(false)
      expect(state.sidebarPinned).toBe(false)
      expect(state.notificationDrawerOpen).toBe(false)
      expect(state.commandPaletteOpen).toBe(false)
      expect(state.activePatientId).toBeNull()
      expect(state.theme).toBe('light')
    })
  })

  describe('sidebar', () => {
    it('toggleSidebar flips sidebarExpanded', () => {
      useUIStore.getState().toggleSidebar()
      expect(useUIStore.getState().sidebarExpanded).toBe(true)
      useUIStore.getState().toggleSidebar()
      expect(useUIStore.getState().sidebarExpanded).toBe(false)
    })

    it('setSidebarPinned sets both pinned and expanded', () => {
      useUIStore.getState().setSidebarPinned(true)
      const state = useUIStore.getState()
      expect(state.sidebarPinned).toBe(true)
      expect(state.sidebarExpanded).toBe(true)
    })

    it('unpinning sidebar also collapses it', () => {
      useUIStore.getState().setSidebarPinned(true)
      useUIStore.getState().setSidebarPinned(false)
      const state = useUIStore.getState()
      expect(state.sidebarPinned).toBe(false)
      expect(state.sidebarExpanded).toBe(false)
    })
  })

  describe('notification drawer', () => {
    it('toggleNotifications opens and closes', () => {
      useUIStore.getState().toggleNotifications()
      expect(useUIStore.getState().notificationDrawerOpen).toBe(true)
      useUIStore.getState().toggleNotifications()
      expect(useUIStore.getState().notificationDrawerOpen).toBe(false)
    })
  })

  describe('command palette', () => {
    it('toggleCommandPalette opens and closes', () => {
      useUIStore.getState().toggleCommandPalette()
      expect(useUIStore.getState().commandPaletteOpen).toBe(true)
      useUIStore.getState().toggleCommandPalette()
      expect(useUIStore.getState().commandPaletteOpen).toBe(false)
    })
  })

  describe('active patient', () => {
    it('setActivePatient sets the ID', () => {
      useUIStore.getState().setActivePatient('patient-123')
      expect(useUIStore.getState().activePatientId).toBe('patient-123')
    })

    it('setActivePatient clears with null', () => {
      useUIStore.getState().setActivePatient('patient-123')
      useUIStore.getState().setActivePatient(null)
      expect(useUIStore.getState().activePatientId).toBeNull()
    })
  })

  describe('theme', () => {
    it('setTheme sets the theme', () => {
      useUIStore.getState().setTheme('dark')
      expect(useUIStore.getState().theme).toBe('dark')
      useUIStore.getState().setTheme('light')
      expect(useUIStore.getState().theme).toBe('light')
    })

    it('toggleTheme switches between light and dark', () => {
      expect(useUIStore.getState().theme).toBe('light')
      useUIStore.getState().toggleTheme()
      expect(useUIStore.getState().theme).toBe('dark')
      useUIStore.getState().toggleTheme()
      expect(useUIStore.getState().theme).toBe('light')
    })
  })
})
