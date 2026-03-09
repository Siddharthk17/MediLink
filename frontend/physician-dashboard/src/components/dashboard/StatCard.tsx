'use client'

import { useEffect, useState } from 'react'
import { useMotionValue, useSpring } from 'framer-motion'
import React from 'react'

interface StatBlockProps {
  label: string
  value: number
  change?: string
  changeType?: 'positive' | 'negative' | 'neutral'
}

function AnimatedNumber({ value }: { value: number }) {
  const [display, setDisplay] = useState(0)
  const motionValue = useMotionValue(0)
  const spring = useSpring(motionValue, { stiffness: 100, damping: 30 })

  useEffect(() => {
    motionValue.set(value)
  }, [value, motionValue])

  useEffect(() => {
    const unsubscribe = spring.on('change', (v) => setDisplay(Math.round(v)))
    return unsubscribe
  }, [spring])

  return <>{display.toLocaleString()}</>
}

const changeColor: Record<NonNullable<StatBlockProps['changeType']>, string> = {
  positive: 'var(--color-success)',
  negative: 'var(--color-danger)',
  neutral: 'var(--color-text-muted)',
}

export const StatBlock = React.memo(function StatBlock({
  label,
  value,
  change,
  changeType = 'neutral',
}: StatBlockProps) {
  return (
    <div className="glass-card rounded-[22px] border border-[var(--color-border)] p-5 shadow-card">
      <span
        className="font-mono text-[10px] uppercase tracking-[0.14em]"
        style={{ color: 'var(--color-text-muted)' }}
      >
        {label}
      </span>
      <p className="font-display text-[44px] leading-none mt-2 text-[var(--color-text-primary)]">
        <AnimatedNumber value={value} />
      </p>
      {change && (
        <p className="text-xs mt-2 font-medium" style={{ color: changeColor[changeType] }}>
          {change}
        </p>
      )}
    </div>
  )
})
