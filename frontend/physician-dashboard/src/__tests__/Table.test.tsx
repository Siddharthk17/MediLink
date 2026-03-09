import { render, screen, fireEvent } from '@testing-library/react'
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'

function renderFullTable(onRowClick?: () => void) {
  return render(
    <Table>
      <TableHeader>
        <TableHead>Name</TableHead>
        <TableHead>Age</TableHead>
      </TableHeader>
      <TableBody>
        <TableRow onClick={onRowClick}>
          <TableCell>Alice</TableCell>
          <TableCell>30</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Bob</TableCell>
          <TableCell>25</TableCell>
        </TableRow>
      </TableBody>
    </Table>
  )
}

describe('Table', () => {
  it('renders a table element', () => {
    const { container } = renderFullTable()
    expect(container.querySelector('table')).toBeInTheDocument()
  })

  it('wraps table in an overflow container', () => {
    const { container } = renderFullTable()
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper).toHaveClass('overflow-x-auto')
  })

  it('merges custom className on table', () => {
    const { container } = render(
      <Table className="my-table">
        <TableBody>
          <TableRow><TableCell>Cell</TableCell></TableRow>
        </TableBody>
      </Table>
    )
    expect(container.querySelector('table')).toHaveClass('my-table')
    expect(container.querySelector('table')).toHaveClass('w-full')
  })
})

describe('TableHeader', () => {
  it('renders thead with tr', () => {
    const { container } = renderFullTable()
    expect(container.querySelector('thead')).toBeInTheDocument()
    expect(container.querySelector('thead tr')).toBeInTheDocument()
  })
})

describe('TableHead', () => {
  it('renders th elements', () => {
    renderFullTable()
    const headers = screen.getAllByRole('columnheader')
    expect(headers).toHaveLength(2)
    expect(headers[0]).toHaveTextContent('Name')
    expect(headers[1]).toHaveTextContent('Age')
  })

  it('merges custom className on th', () => {
    const { container } = render(
      <Table>
        <TableHeader>
          <TableHead className="w-1/2">Col</TableHead>
        </TableHeader>
      </Table>
    )
    const th = container.querySelector('th')
    expect(th).toHaveClass('w-1/2')
    expect(th).toHaveClass('px-4')
  })
})

describe('TableBody', () => {
  it('renders tbody', () => {
    const { container } = renderFullTable()
    expect(container.querySelector('tbody')).toBeInTheDocument()
  })
})

describe('TableRow', () => {
  it('renders tr elements with cells', () => {
    renderFullTable()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
  })

  it('calls onClick when row is clicked', () => {
    const handleClick = vi.fn()
    renderFullTable(handleClick)
    fireEvent.click(screen.getByText('Alice'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('has cursor-pointer class when onClick is provided', () => {
    const handleClick = vi.fn()
    renderFullTable(handleClick)
    const row = screen.getByText('Alice').closest('tr')
    expect(row).toHaveClass('cursor-pointer')
  })

  it('does not have cursor-pointer class when onClick is absent', () => {
    renderFullTable()
    const row = screen.getByText('Bob').closest('tr')
    expect(row).not.toHaveClass('cursor-pointer')
  })

  it('merges custom className on tr', () => {
    const { container } = render(
      <Table>
        <TableBody>
          <TableRow className="highlight"><TableCell>X</TableCell></TableRow>
        </TableBody>
      </Table>
    )
    expect(container.querySelector('tr')).toHaveClass('highlight')
  })
})

describe('TableCell', () => {
  it('renders td elements', () => {
    renderFullTable()
    const cells = screen.getAllByRole('cell')
    expect(cells.length).toBeGreaterThanOrEqual(4)
  })

  it('merges custom className on td', () => {
    const { container } = render(
      <Table>
        <TableBody>
          <TableRow><TableCell className="font-bold">Styled</TableCell></TableRow>
        </TableBody>
      </Table>
    )
    const td = container.querySelector('td')
    expect(td).toHaveClass('font-bold')
    expect(td).toHaveClass('px-4')
  })
})
