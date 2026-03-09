import { vi } from 'vitest'

const mockFont = {
  className: 'mocked-font-class',
  style: { fontFamily: 'mocked' },
  variable: '--font-mock',
}

vi.mock('next/font/google', () => ({
  Instrument_Serif: vi.fn(() => ({ ...mockFont, variable: '--font-display' })),
  DM_Sans: vi.fn(() => ({ ...mockFont, variable: '--font-body' })),
  JetBrains_Mono: vi.fn(() => ({ ...mockFont, variable: '--font-mono' })),
}))

describe('fonts', () => {
  it('exports instrumentSerif with --font-display variable', async () => {
    const { instrumentSerif } = await import('@/lib/fonts')
    expect(instrumentSerif).toBeDefined()
    expect(instrumentSerif.variable).toBe('--font-display')
  })

  it('exports dmSans with --font-body variable', async () => {
    const { dmSans } = await import('@/lib/fonts')
    expect(dmSans).toBeDefined()
    expect(dmSans.variable).toBe('--font-body')
  })

  it('exports jetbrainsMono with --font-mono variable', async () => {
    const { jetbrainsMono } = await import('@/lib/fonts')
    expect(jetbrainsMono).toBeDefined()
    expect(jetbrainsMono.variable).toBe('--font-mono')
  })

  it('calls font constructors with correct config', async () => {
    const { Instrument_Serif, DM_Sans, JetBrains_Mono } = await import('next/font/google')

    expect(Instrument_Serif).toHaveBeenCalledWith({
      weight: '400',
      subsets: ['latin'],
      display: 'swap',
      variable: '--font-display',
    })

    expect(DM_Sans).toHaveBeenCalledWith({
      subsets: ['latin'],
      display: 'swap',
      variable: '--font-body',
    })

    expect(JetBrains_Mono).toHaveBeenCalledWith({
      subsets: ['latin'],
      display: 'swap',
      variable: '--font-mono',
    })
  })
})
