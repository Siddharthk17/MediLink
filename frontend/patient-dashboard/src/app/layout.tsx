import type { Metadata } from 'next'
import { instrumentSerif, dmSans, jetbrainsMono } from '@/lib/fonts'
import './globals.css'
import { Providers } from './providers'

export const metadata: Metadata = {
  title: 'MediLink — My Health',
  description: 'Your personal health records and care management portal',
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
