'use client'

import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from '@/components/ui/Table'
import { Badge } from '@/components/ui/Badge'
import { getObservationValue, getCodeDisplay, formatDate, getObservationStatus } from '@medilink/shared'
import type { Observation } from '@medilink/shared'

interface ObservationTableProps {
  observations: Observation[]
}

export function ObservationTable({ observations }: ObservationTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableHead>Date</TableHead>
        <TableHead>Value</TableHead>
        <TableHead>Unit</TableHead>
        <TableHead>Status</TableHead>
        <TableHead>Interpretation</TableHead>
      </TableHeader>
      <TableBody>
        {observations.map((obs) => {
          const status = getObservationStatus(obs)
          return (
            <TableRow
              key={obs.id}
              className={status === 'high' || status === 'low' ? 'bg-[var(--color-danger-subtle)]' : ''}
            >
              <TableCell>{formatDate(obs.effectiveDateTime ?? '')}</TableCell>
              <TableCell className="font-mono text-[var(--color-text-primary)]">
                {obs.valueQuantity?.value ?? '—'}
              </TableCell>
              <TableCell>{obs.valueQuantity?.unit || '—'}</TableCell>
              <TableCell>
                <Badge variant="muted" size="sm">{obs.status}</Badge>
              </TableCell>
              <TableCell>
                {status === 'high' && <Badge variant="danger" size="sm">HIGH</Badge>}
                {status === 'low' && <Badge variant="warning" size="sm">LOW</Badge>}
                {status === 'normal' && <Badge variant="success" size="sm">NORMAL</Badge>}
                {status === 'unknown' && <span style={{ color: 'var(--color-text-muted)' }}>—</span>}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
