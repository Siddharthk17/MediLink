import { render } from '@testing-library/react'
import { Spinner, PageSpinner } from '@/components/ui/Spinner'

vi.mock('@/lib/utils', () => ({
  cn: (...args: any[]) => args.filter(Boolean).join(' '),
}))

describe('Spinner', () => {
  it('renders a div element', () => {
    const { container } = render(<Spinner />)
    expect(container.firstChild?.nodeName).toBe('DIV')
  })

  it('has animate-spin class', () => {
    const { container } = render(<Spinner />)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('animate-spin')
  })

  it('has rounded-full class', () => {
    const { container } = render(<Spinner />)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('rounded-full')
  })

  it('has border styling', () => {
    const { container } = render(<Spinner />)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('border-2')
  })

  describe('sizes', () => {
    it('defaults to md size (w-6 h-6)', () => {
      const { container } = render(<Spinner />)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('w-6')
      expect(el.className).toContain('h-6')
    })

    it('applies sm size (w-4 h-4)', () => {
      const { container } = render(<Spinner size="sm" />)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('w-4')
      expect(el.className).toContain('h-4')
    })

    it('applies lg size (w-10 h-10)', () => {
      const { container } = render(<Spinner size="lg" />)
      const el = container.firstChild as HTMLElement
      expect(el.className).toContain('w-10')
      expect(el.className).toContain('h-10')
    })
  })

  it('merges custom className', () => {
    const { container } = render(<Spinner className="text-red-500" />)
    const el = container.firstChild as HTMLElement
    expect(el.className).toContain('text-red-500')
    expect(el.className).toContain('animate-spin')
  })
})

describe('PageSpinner', () => {
  it('renders a centered container with min height', () => {
    const { container } = render(<PageSpinner />)
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper.className).toContain('flex')
    expect(wrapper.className).toContain('items-center')
    expect(wrapper.className).toContain('justify-center')
    expect(wrapper.className).toContain('min-h-[60vh]')
  })

  it('contains a large spinner', () => {
    const { container } = render(<PageSpinner />)
    const spinner = container.querySelector('.animate-spin') as HTMLElement
    expect(spinner).toBeInTheDocument()
    expect(spinner.className).toContain('w-10')
    expect(spinner.className).toContain('h-10')
  })
})
