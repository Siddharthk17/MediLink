import { render, screen, fireEvent } from '@testing-library/react'
import { Tooltip } from '@/components/ui/Tooltip'

describe('Tooltip', () => {
  it('renders children', () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>)
    expect(screen.getByText('Hover me')).toBeInTheDocument()
  })

  it('does not show tooltip content initially', () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>)
    expect(screen.queryByText('Help text')).not.toBeInTheDocument()
  })

  it('shows tooltip on mouse enter', () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>)
    fireEvent.mouseEnter(screen.getByText('Hover me').parentElement!)
    expect(screen.getByText('Help text')).toBeInTheDocument()
  })

  it('hides tooltip on mouse leave', () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>)
    const wrapper = screen.getByText('Hover me').parentElement!
    fireEvent.mouseEnter(wrapper)
    expect(screen.getByText('Help text')).toBeInTheDocument()
    fireEvent.mouseLeave(wrapper)
    expect(screen.queryByText('Help text')).not.toBeInTheDocument()
  })

  describe('sides', () => {
    it('defaults to right side', () => {
      render(<Tooltip content="Tip"><button>Btn</button></Tooltip>)
      fireEvent.mouseEnter(screen.getByText('Btn').parentElement!)
      const tip = screen.getByText('Tip')
      expect(tip).toHaveClass('ml-2')
    })

    it('applies top positioning', () => {
      render(<Tooltip content="Tip" side="top"><button>Btn</button></Tooltip>)
      fireEvent.mouseEnter(screen.getByText('Btn').parentElement!)
      const tip = screen.getByText('Tip')
      expect(tip).toHaveClass('mb-2')
    })

    it('applies bottom positioning', () => {
      render(<Tooltip content="Tip" side="bottom"><button>Btn</button></Tooltip>)
      fireEvent.mouseEnter(screen.getByText('Btn').parentElement!)
      const tip = screen.getByText('Tip')
      expect(tip).toHaveClass('mt-2')
    })

    it('applies left positioning', () => {
      render(<Tooltip content="Tip" side="left"><button>Btn</button></Tooltip>)
      fireEvent.mouseEnter(screen.getByText('Btn').parentElement!)
      const tip = screen.getByText('Tip')
      expect(tip).toHaveClass('mr-2')
    })
  })

  it('tooltip has pointer-events-none', () => {
    render(<Tooltip content="Tip"><span>Target</span></Tooltip>)
    fireEvent.mouseEnter(screen.getByText('Target').parentElement!)
    expect(screen.getByText('Tip')).toHaveClass('pointer-events-none')
  })

  it('wraps children in a relative container', () => {
    render(<Tooltip content="Tip"><span>Target</span></Tooltip>)
    const wrapper = screen.getByText('Target').parentElement!
    expect(wrapper).toHaveClass('relative', 'inline-flex')
  })
})
