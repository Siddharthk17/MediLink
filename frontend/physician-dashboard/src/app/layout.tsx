import type { Metadata } from 'next'
import { instrumentSerif, dmSans, jetbrainsMono } from '@/lib/fonts'
import './globals.css'
import { Providers } from './providers'

export const metadata: Metadata = {
  title: 'MediLink — Physician Dashboard',
  description: 'FHIR R4-compliant medical dashboard for physicians',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${instrumentSerif.variable} ${dmSans.variable} ${jetbrainsMono.variable}`}
    >
      <body className="font-body antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
