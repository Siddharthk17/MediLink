import { create } from 'zustand'

interface UIState {
  theme: 'light' | 'dark'
  commandPaletteOpen: boolean
  setTheme: (theme: 'light' | 'dark') => void
  toggleTheme: () => void
  toggleCommandPalette: () => void
}

export const useUIStore = create<UIState>((set) => ({
  theme: 'light',
  commandPaletteOpen: false,
  setTheme: (theme) => set({ theme }),
  toggleTheme: () => set((s) => ({ theme: s.theme === 'light' ? 'dark' : 'light' })),
  toggleCommandPalette: () => set((s) => ({ commandPaletteOpen: !s.commandPaletteOpen })),
}))
