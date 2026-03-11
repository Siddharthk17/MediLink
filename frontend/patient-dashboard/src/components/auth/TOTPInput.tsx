'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { motion } from 'framer-motion'
import { shakeVariants } from '@/lib/motion'
import { cn } from '@/lib/utils'

interface TOTPInputProps {
  onComplete: (code: string) => void
  error?: boolean
  disabled?: boolean
}

export function TOTPInput({ onComplete, error, disabled }: TOTPInputProps) {
  const [digits, setDigits] = useState<string[]>(['', '', '', '', '', ''])
  const refs = useRef<(HTMLInputElement | null)[]>([])

  const resetInputs = useCallback(() => {
    setDigits(['', '', '', '', '', ''])
    refs.current[0]?.focus()
  }, [])

  useEffect(() => {
    if (error) {
      const timer = setTimeout(resetInputs, 400)
      return () => clearTimeout(timer)
    }
  }, [error, resetInputs])

  useEffect(() => {
    refs.current[0]?.focus()
  }, [])

  const handleChange = (index: number, value: string) => {
    const cleaned = value.replace(/\D/g, '')
    if (!cleaned) return

    if (cleaned.length > 1) {
      const chars = cleaned.slice(0, 6 - index).split('')
      const newDigits = [...digits]
      chars.forEach((char, i) => {
        if (index + i < 6) newDigits[index + i] = char
      })
      setDigits(newDigits)
      const lastIndex = Math.min(index + chars.length, 5)
      refs.current[lastIndex]?.focus()
      if (newDigits.every((d) => d !== '')) {
        onComplete(newDigits.join(''))
      }
      return
    }

    const newDigits = [...digits]
    newDigits[index] = cleaned[0]
    setDigits(newDigits)

    if (index < 5) {
      refs.current[index + 1]?.focus()
    }

    if (newDigits.every((d) => d !== '')) {
      onComplete(newDigits.join(''))
    }
  }

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace') {
      if (!digits[index] && index > 0) {
        const newDigits = [...digits]
        newDigits[index - 1] = ''
        setDigits(newDigits)
        refs.current[index - 1]?.focus()
      } else {
        const newDigits = [...digits]
        newDigits[index] = ''
        setDigits(newDigits)
      }
      e.preventDefault()
    } else if (e.key === 'ArrowLeft' && index > 0) {
      refs.current[index - 1]?.focus()
    } else if (e.key === 'ArrowRight' && index < 5) {
      refs.current[index + 1]?.focus()
    } else if (['e', '.', '-', '+'].includes(e.key)) {
      e.preventDefault()
    }
  }

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault()
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6)
    if (!pasted) return
    const newDigits = [...digits]
    pasted.split('').forEach((char, i) => {
      newDigits[i] = char
    })
    setDigits(newDigits)
    const lastIndex = Math.min(pasted.length, 5)
    refs.current[lastIndex]?.focus()
    if (pasted.length === 6) {
      onComplete(newDigits.join(''))
    }
  }

  return (
    <motion.div
      className="flex justify-center gap-2"
      animate={error ? 'shake' : 'idle'}
      variants={shakeVariants}
    >
      {digits.map((digit, i) => (
        <input
          key={i}
          ref={(el) => { refs.current[i] = el }}
          type="text"
          inputMode="numeric"
          maxLength={1}
          value={digit}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          onPaste={handlePaste}
          disabled={disabled}
          aria-label={`Digit ${i + 1}`}
          className={cn(
            'w-12 h-[60px] text-center text-2xl font-mono rounded-lg transition-all duration-[var(--duration-fast)]',
            'bg-[var(--color-bg-card)] border text-[var(--color-text-primary)]',
            'focus:outline-none focus:border-[var(--color-accent)] focus:shadow-glow',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            error
              ? 'border-[var(--color-danger)] bg-[var(--color-danger-subtle)]'
              : digit
                ? 'border-[rgba(20,184,166,0.5)]'
                : 'border-[var(--color-border)]'
          )}
        />
      ))}
    </motion.div>
  )
}
