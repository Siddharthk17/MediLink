'use client'

import React, { useMemo } from 'react'
import {
  ComposedChart, Area, Line, XAxis, YAxis, CartesianGrid,
  Tooltip as RechartsTooltip, ReferenceArea, ResponsiveContainer
} from 'recharts'
import type { Observation } from '@medilink/shared'
import { getObservationValue, formatDate, getObservationStatus } from '@medilink/shared'

interface LabTrendChartProps {
  observations: Observation[]
  loincCode: string
}

interface ChartPoint {
  date: string
  value: number
  fullDate: string
  interpretation: string | null
  refLow?: number
  refHigh?: number
  outOfRange: boolean
}

function CustomDot(props: { cx?: number; cy?: number; payload?: ChartPoint }) {
  const { cx, cy, payload } = props
  if (!cx || !cy || !payload) return null
  if (payload.outOfRange) {
    return <circle cx={cx} cy={cy} r={5} fill="var(--color-danger)" stroke="var(--color-bg-card)" strokeWidth={2} />
  }
  return null
}

function CustomTooltipContent({ active, payload }: { active?: boolean; payload?: Array<{ payload: ChartPoint }> }) {
  if (!active || !payload?.[0]) return null
  const data = payload[0].payload
  return (
    <div className="p-3 rounded-card border border-[var(--color-border)] bg-[var(--color-bg-card)] shadow-elevated">
      <p className="text-xs font-medium" style={{ color: 'var(--color-text-primary)' }}>{data.fullDate}</p>
      <p className="text-lg font-display mt-1" style={{ color: data.outOfRange ? 'var(--color-danger)' : 'var(--color-accent)' }}>
        {data.value}
      </p>
      {data.refLow !== undefined && data.refHigh !== undefined && (
        <p className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
          Normal: {data.refLow}–{data.refHigh}
        </p>
      )}
      {data.outOfRange && (
        <p className="text-xs font-medium mt-1" style={{ color: 'var(--color-danger)' }}>
          {data.value > (data.refHigh ?? Infinity) ? 'ABOVE NORMAL' : 'BELOW NORMAL'}
        </p>
      )}
    </div>
  )
}

export const LabTrendChart = React.memo(function LabTrendChart({ observations, loincCode }: LabTrendChartProps) {
  const chartData = useMemo(() => {
    return observations
      .filter((obs) => obs.valueQuantity?.value !== undefined)
      .sort((a, b) => new Date(a.effectiveDateTime || '').getTime() - new Date(b.effectiveDateTime || '').getTime())
      .map((obs): ChartPoint => {
        const value = obs.valueQuantity?.value ?? 0
        const refLow = obs.referenceRange?.[0]?.low?.value
        const refHigh = obs.referenceRange?.[0]?.high?.value
        const outOfRange = (refHigh !== undefined && value > refHigh) || (refLow !== undefined && value < refLow)
        return {
          date: formatDate(obs.effectiveDateTime ?? '')!,
          value,
          fullDate: obs.effectiveDateTime || '',
          interpretation: obs.interpretation?.[0]?.coding?.[0]?.code || null,
          refLow,
          refHigh,
          outOfRange,
        }
      })
  }, [observations])

  if (chartData.length === 0) {
    return (
      <div className="h-80 flex items-center justify-center" style={{ color: 'var(--color-text-muted)' }}>
        No data for this test type
      </div>
    )
  }

  const refLow = chartData[0]?.refLow
  const refHigh = chartData[0]?.refHigh
  const allValues = chartData.map((d) => d.value)
  const minVal = Math.min(...allValues, refLow ?? Infinity)
  const maxVal = Math.max(...allValues, refHigh ?? -Infinity)
  const yMin = Math.floor(minVal * 0.8)
  const yMax = Math.ceil(maxVal * 1.2)

  return (
    <ResponsiveContainer width="100%" height={320}>
      <ComposedChart data={chartData} key={loincCode} margin={{ top: 10, right: 10, bottom: 0, left: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border-subtle)" vertical={false} />
        <XAxis
          dataKey="date"
          tick={{ fill: 'var(--color-text-muted)', fontSize: 11 }}
          axisLine={false}
          tickLine={false}
        />
        <YAxis
          domain={[yMin, yMax]}
          tick={{ fill: 'var(--color-text-muted)', fontSize: 11 }}
          axisLine={false}
          tickLine={false}
        />
        <RechartsTooltip content={<CustomTooltipContent />} />
        {refLow !== undefined && refHigh !== undefined && (
          <ReferenceArea
            y1={refLow}
            y2={refHigh}
            fill="var(--color-success-subtle)"
            stroke="var(--color-success)"
            strokeOpacity={0.25}
            strokeDasharray="4 4"
          />
        )}
        <defs>
          <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-accent)" stopOpacity={0.2} />
            <stop offset="100%" stopColor="var(--color-accent)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <Area type="monotone" dataKey="value" fill="url(#areaGradient)" stroke="none" />
        <Line
          type="monotone"
          dataKey="value"
          stroke="var(--color-accent)"
          strokeWidth={2}
          dot={<CustomDot />}
          activeDot={{ r: 6, fill: 'var(--color-accent)', stroke: 'var(--color-bg-card)', strokeWidth: 2 }}
        />
      </ComposedChart>
    </ResponsiveContainer>
  )
})
