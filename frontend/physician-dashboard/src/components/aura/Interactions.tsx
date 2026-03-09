'use client'

import { useEffect, useRef, useState } from 'react'
import { motion, useMotionValue, useSpring, useTransform } from 'framer-motion'
import { ArrowUpRight } from 'lucide-react'

export function AdvancedCursor() {
  const [enabled, setEnabled] = useState(false)
  const [isHovering, setIsHovering] = useState(false)

  const cursorX = useMotionValue(-100)
  const cursorY = useMotionValue(-100)
  const springConfig = { damping: 25, stiffness: 300, mass: 0.5 }
  const cursorXSpring = useSpring(cursorX, springConfig)
  const cursorYSpring = useSpring(cursorY, springConfig)

  useEffect(() => {
    if (typeof window === 'undefined') return
    const media = window.matchMedia('(pointer: fine)')
    const update = () => setEnabled(media.matches)
    update()
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  useEffect(() => {
    if (!enabled) return

    const styleTag = document.createElement('style')
    styleTag.id = 'aura-cursor-style'
    styleTag.textContent = '*{cursor:none!important;}'
    document.head.appendChild(styleTag)

    const moveCursor = (e: MouseEvent) => {
      cursorX.set(e.clientX)
      cursorY.set(e.clientY)

      const target = e.target as HTMLElement
      const tagName = target.tagName.toLowerCase()
      const isInteractive =
        window.getComputedStyle(target).cursor === 'pointer' ||
        ['a', 'button', 'input', 'textarea', 'select', 'label'].includes(tagName) ||
        Boolean(target.closest('a,button,[role="button"]'))

      setIsHovering(isInteractive)
    }

    window.addEventListener('mousemove', moveCursor)
    return () => {
      window.removeEventListener('mousemove', moveCursor)
      styleTag.remove()
    }
  }, [enabled, cursorX, cursorY])

  if (!enabled) return null

  return (
    <>
      <motion.div
        className="fixed top-0 left-0 w-2 h-2 bg-[#1D1D1F] rounded-full pointer-events-none z-[100] mix-blend-difference"
        style={{ x: cursorX, y: cursorY, translateX: '-50%', translateY: '-50%' }}
        animate={{ opacity: isHovering ? 0 : 1 }}
      />
      <motion.div
        className="fixed top-0 left-0 w-8 h-8 border border-[#1D1D1F] rounded-full pointer-events-none z-[99] mix-blend-difference flex items-center justify-center"
        style={{ x: cursorXSpring, y: cursorYSpring, translateX: '-50%', translateY: '-50%' }}
        animate={{
          scale: isHovering ? 2.4 : 1,
          backgroundColor: isHovering ? '#FFFFFF' : 'transparent',
          borderColor: isHovering ? '#FFFFFF' : '#1D1D1F',
        }}
      >
        {isHovering && <ArrowUpRight className="w-3 h-3 text-black" />}
      </motion.div>
    </>
  )
}

export function Magnetic({ children, className }: { children: React.ReactNode; className?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const [mounted, setMounted] = useState(false)
  const [position, setPosition] = useState({ x: 0, y: 0 })

  useEffect(() => setMounted(true), [])

  const handleMouseMove = (e: React.MouseEvent) => {
    const rect = ref.current?.getBoundingClientRect()
    if (!rect) return
    const middleX = e.clientX - (rect.left + rect.width / 2)
    const middleY = e.clientY - (rect.top + rect.height / 2)
    setPosition({ x: middleX * 0.2, y: middleY * 0.2 })
  }

  return (
    <div
      ref={ref}
      className={className}
      onMouseMove={mounted ? handleMouseMove : undefined}
      onMouseLeave={mounted ? () => setPosition({ x: 0, y: 0 }) : undefined}
      style={mounted ? {
        transform: `translate(${position.x}px, ${position.y}px)`,
        transition: 'transform 0.15s ease-out',
      } : undefined}
    >
      {children}
    </div>
  )
}

export function TiltCard({ children, className }: { children: React.ReactNode; className?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const x = useMotionValue(0)
  const y = useMotionValue(0)
  const mouseXSpring = useSpring(x, { stiffness: 300, damping: 30 })
  const mouseYSpring = useSpring(y, { stiffness: 300, damping: 30 })
  const rotateX = useTransform(mouseYSpring, [-0.5, 0.5], ['4deg', '-4deg'])
  const rotateY = useTransform(mouseXSpring, [-0.5, 0.5], ['-4deg', '4deg'])

  const handleMouseMove = (e: React.MouseEvent) => {
    const rect = ref.current?.getBoundingClientRect()
    if (!rect) return
    const xPct = (e.clientX - rect.left) / rect.width - 0.5
    const yPct = (e.clientY - rect.top) / rect.height - 0.5
    x.set(xPct)
    y.set(yPct)
  }

  return (
    <motion.div
      ref={ref}
      className={className}
      onMouseMove={handleMouseMove}
      onMouseLeave={() => {
        x.set(0)
        y.set(0)
      }}
      style={{ rotateX, rotateY, transformStyle: 'preserve-3d' }}
    >
      <div className="h-full w-full" style={{ transform: 'translateZ(20px)' }}>
        {children}
      </div>
    </motion.div>
  )
}

export function WordReveal({ text, className }: { text: string; className?: string }) {
  const words = text.split(' ')
  return (
    <h1 className={className}>
      {words.map((word, i) => (
        <span key={`${word}-${i}`} className="inline-block overflow-hidden mr-[0.25em] pb-2">
          <motion.span
            className="inline-block origin-bottom"
            initial={{ y: '100%', rotate: 5, opacity: 0 }}
            animate={{ y: '0%', rotate: 0, opacity: 1 }}
            transition={{ duration: 0.8, delay: i * 0.05, ease: [0.16, 1, 0.3, 1] }}
          >
            {word}
          </motion.span>
        </span>
      ))}
    </h1>
  )
}
