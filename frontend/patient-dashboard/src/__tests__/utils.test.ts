import { cn } from '@/lib/utils'

describe('cn utility', () => {
  it('merges multiple class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible')
  })

  it('handles undefined and null values', () => {
    expect(cn('base', undefined, null, 'end')).toBe('base end')
  })

  it('deduplicates conflicting tailwind classes', () => {
    const result = cn('px-4', 'px-6')
    expect(result).toBe('px-6')
  })

  it('handles empty arguments', () => {
    expect(cn()).toBe('')
  })

  it('handles single argument', () => {
    expect(cn('only-class')).toBe('only-class')
  })

  it('handles array-style clsx inputs', () => {
    expect(cn(['a', 'b'])).toBe('a b')
  })

  it('merges tailwind variants correctly', () => {
    const result = cn('text-sm text-red-500', 'text-blue-500')
    expect(result).toBe('text-sm text-blue-500')
  })
})
