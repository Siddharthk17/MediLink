export const pageVariants = {
  initial: { opacity: 0 },
  animate: { opacity: 1, transition: { duration: 0.15, ease: 'easeOut' } },
  exit:    { opacity: 0, transition: { duration: 0.1 } },
}

export const staggerContainer = {
  animate: { transition: { staggerChildren: 0.04 } }
}

export const staggerItem = {
  initial: { opacity: 0, y: 6 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.18, ease: 'easeOut' } },
}

export const sidebarVariants = {
  collapsed: { width: 64 },
  expanded:  { width: 240, transition: { type: 'spring', stiffness: 400, damping: 35 } },
}

export const alertPanelVariants = {
  hidden:  { y: '100%', opacity: 0 },
  visible: { y: 0, opacity: 1, transition: { type: 'spring', stiffness: 300, damping: 28 } },
}

export const shakeVariants = {
  idle: {},
  shake: {
    x: [0, -6, 6, -4, 4, -2, 2, 0],
    transition: { duration: 0.35 }
  }
}
