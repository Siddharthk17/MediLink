import { describe, it, expect, beforeEach } from 'vitest'
import { useUIStore } from '@/store/uiStore'

describe('uiStore', () => {
  beforeEach(() => {
    const s = useUIStore.getState()
    s.setTheme('light')
    if (s.commandPaletteOpen) s.toggleCommandPalette()
  })

  describe('initial state', () => {
    it('has correct defaults', () => {
      const state = useUIStore.getState()
      expect(state.theme).toBe('light')
      expect(state.commandPaletteOpen).toBe(false)
    })
  })

  describe('theme', () => {
    it('setTheme sets the theme to dark', () => {
      useUIStore.getState().setTheme('dark')
      expect(useUIStore.getState().theme).toBe('dark')
    })

    it('setTheme sets the theme to light', () => {
      useUIStore.getState().setTheme('dark')
      useUIStore.getState().setTheme('light')
      expect(useUIStore.getState().theme).toBe('light')
    })

    it('toggleTheme switches from light to dark', () => {
      expect(useUIStore.getState().theme).toBe('light')
      useUIStore.getState().toggleTheme()
      expect(useUIStore.getState().theme).toBe('dark')
    })

    it('toggleTheme switches from dark to light', () => {
      useUIStore.getState().setTheme('dark')
      useUIStore.getState().toggleTheme()
      expect(useUIStore.getState().theme).toBe('light')
    })

    it('toggleTheme is idempotent through two cycles', () => {
      useUIStore.getState().toggleTheme()
      useUIStore.getState().toggleTheme()
      expect(useUIStore.getState().theme).toBe('light')
    })
  })

  describe('command palette', () => {
    it('toggleCommandPalette opens', () => {
      useUIStore.getState().toggleCommandPalette()
      expect(useUIStore.getState().commandPaletteOpen).toBe(true)
    })

    it('toggleCommandPalette closes when open', () => {
      useUIStore.getState().toggleCommandPalette()
      useUIStore.getState().toggleCommandPalette()
      expect(useUIStore.getState().commandPaletteOpen).toBe(false)
    })

    it('command palette toggle is independent of theme', () => {
      useUIStore.getState().toggleTheme()
      useUIStore.getState().toggleCommandPalette()
      expect(useUIStore.getState().theme).toBe('dark')
      expect(useUIStore.getState().commandPaletteOpen).toBe(true)
    })
  })
})
