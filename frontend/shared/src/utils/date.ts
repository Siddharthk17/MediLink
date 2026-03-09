import { format, formatDistanceToNow, differenceInDays } from 'date-fns'

export function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd MMM yyyy')
}

export function formatDateTime(dateStr: string): string {
  return format(new Date(dateStr), 'dd MMM yyyy, HH:mm')
}

export function formatRelative(dateStr: string): string {
  return formatDistanceToNow(new Date(dateStr), { addSuffix: true })
}

export function daysUntil(dateStr: string): number {
  return differenceInDays(new Date(dateStr), new Date())
}
