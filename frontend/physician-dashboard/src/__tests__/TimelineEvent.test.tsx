import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

vi.mock('@medilink/shared', () => ({
  getCodeDisplay: vi.fn((codeableConcept: any) => {
    if (!codeableConcept) return 'Unknown'
    return codeableConcept?.coding?.[0]?.display || codeableConcept?.text || 'Unknown'
  }),
  formatRelative: vi.fn((date: string) => 'recently'),
}))

import { TimelineEvent } from '@/components/patients/TimelineEvent'

describe('TimelineEvent', () => {
  it('renders the resource type badge', () => {
    render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e1', class: { display: 'Outpatient' }, period: { start: '2024-01-01' } }}
      />
    )
    expect(screen.getByText('Encounter')).toBeInTheDocument()
  })

  it('renders encounter title with class display', () => {
    render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e1', class: { display: 'Outpatient' } }}
      />
    )
    expect(screen.getByText('Outpatient encounter')).toBeInTheDocument()
  })

  it('defaults encounter title to Visit when class has no display', () => {
    render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e1' }}
      />
    )
    expect(screen.getByText('Visit encounter')).toBeInTheDocument()
  })

  it('renders condition title from code display', () => {
    render(
      <TimelineEvent
        resource={{
          resourceType: 'Condition',
          id: 'c1',
          code: { coding: [{ display: 'Diabetes' }] },
          onsetDateTime: '2024-02-01',
        }}
      />
    )
    expect(screen.getByText('Diabetes')).toBeInTheDocument()
  })

  it('renders medication title from medicationCodeableConcept', () => {
    render(
      <TimelineEvent
        resource={{
          resourceType: 'MedicationRequest',
          id: 'm1',
          medicationCodeableConcept: { coding: [{ display: 'Metformin' }] },
          authoredOn: '2024-03-01',
        }}
      />
    )
    expect(screen.getByText('Metformin')).toBeInTheDocument()
  })

  it('renders formatted date', () => {
    render(
      <TimelineEvent
        resource={{ resourceType: 'Observation', id: 'o1', code: { coding: [{ display: 'HbA1c' }] }, effectiveDateTime: '2024-04-01' }}
      />
    )
    expect(screen.getByText('recently')).toBeInTheDocument()
  })

  it('renders dash when no date is available', () => {
    render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e2' }}
      />
    )
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('renders connector line when isLast is false', () => {
    const { container } = render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e1' }}
        isLast={false}
      />
    )
    const connector = container.querySelector('.absolute')
    expect(connector).toBeInTheDocument()
  })

  it('does not render connector line when isLast is true', () => {
    const { container } = render(
      <TimelineEvent
        resource={{ resourceType: 'Encounter', id: 'e1' }}
        isLast={true}
      />
    )
    const connector = container.querySelector('.absolute')
    expect(connector).not.toBeInTheDocument()
  })

  it('renders the icon container with resource-specific colors', () => {
    const { container } = render(
      <TimelineEvent
        resource={{ resourceType: 'Condition', id: 'c2', code: { coding: [{ display: 'Test' }] } }}
      />
    )
    const iconContainer = container.querySelector('.w-10.h-10.rounded-full')
    expect(iconContainer).toHaveStyle({ color: 'var(--color-type-condition)' })
  })

  it('renders immunization title from vaccineCode', () => {
    render(
      <TimelineEvent
        resource={{
          resourceType: 'Immunization',
          id: 'i1',
          vaccineCode: { coding: [{ display: 'COVID-19 Vaccine' }] },
          occurrenceDateTime: '2024-05-01',
        }}
      />
    )
    expect(screen.getByText('COVID-19 Vaccine')).toBeInTheDocument()
  })

  it('falls back to Encounter config for unknown resource types', () => {
    const { container } = render(
      <TimelineEvent
        resource={{ resourceType: 'UnknownType', id: 'u1' }}
      />
    )
    const iconContainer = container.querySelector('.w-10.h-10.rounded-full')
    // Falls back to Encounter config color
    expect(iconContainer).toHaveStyle({ color: 'var(--color-type-encounter)' })
  })
})
