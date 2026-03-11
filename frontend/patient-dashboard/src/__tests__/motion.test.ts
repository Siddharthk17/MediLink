import { pageVariants, staggerContainer, staggerItem, cardReveal } from '@/lib/motion'

describe('motion variants', () => {
  describe('pageVariants', () => {
    it('has initial, animate, and exit states', () => {
      expect(pageVariants).toHaveProperty('initial')
      expect(pageVariants).toHaveProperty('animate')
      expect(pageVariants).toHaveProperty('exit')
    })

    it('initial state is fully transparent', () => {
      expect(pageVariants.initial.opacity).toBe(0)
    })

    it('animate state is fully opaque', () => {
      expect(pageVariants.animate.opacity).toBe(1)
    })

    it('exit state is transparent', () => {
      expect(pageVariants.exit.opacity).toBe(0)
    })

    it('animate has easeOut transition', () => {
      expect(pageVariants.animate.transition.ease).toBe('easeOut')
    })

    it('animate transition duration is 0.15s', () => {
      expect(pageVariants.animate.transition.duration).toBe(0.15)
    })
  })

  describe('staggerContainer', () => {
    it('has animate with staggerChildren', () => {
      expect(staggerContainer.animate.transition.staggerChildren).toBe(0.04)
    })
  })

  describe('staggerItem', () => {
    it('has initial and animate states', () => {
      expect(staggerItem).toHaveProperty('initial')
      expect(staggerItem).toHaveProperty('animate')
    })

    it('starts transparent and offset vertically', () => {
      expect(staggerItem.initial.opacity).toBe(0)
      expect(staggerItem.initial.y).toBe(6)
    })

    it('animates to full opacity and zero offset', () => {
      expect(staggerItem.animate.opacity).toBe(1)
      expect(staggerItem.animate.y).toBe(0)
    })
  })

  describe('cardReveal', () => {
    it('has initial and animate states', () => {
      expect(cardReveal).toHaveProperty('initial')
      expect(cardReveal).toHaveProperty('animate')
    })

    it('starts slightly scaled down', () => {
      expect(cardReveal.initial.scale).toBe(0.98)
    })

    it('animates to full scale', () => {
      expect(cardReveal.animate.scale).toBe(1)
    })

    it('initial has vertical offset of 12', () => {
      expect(cardReveal.initial.y).toBe(12)
    })

    it('uses cubic bezier ease', () => {
      expect(cardReveal.animate.transition.ease).toEqual([0.16, 1, 0.3, 1])
    })
  })
})
