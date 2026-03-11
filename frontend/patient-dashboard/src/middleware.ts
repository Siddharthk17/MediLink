import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const AUTH_PATHS = ['/login', '/register']

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl

  if (pathname.startsWith('/api')) {
    return NextResponse.next()
  }

  const token = req.cookies.get('medilink_patient_token')?.value
  const isAuthPage = AUTH_PATHS.some((p) => pathname.startsWith(p))

  if (pathname === '/') {
    return NextResponse.redirect(new URL(token ? '/dashboard' : '/login', req.url))
  }

  if (!isAuthPage && !token) {
    return NextResponse.redirect(new URL('/login', req.url))
  }

  if (isAuthPage && token) {
    return NextResponse.redirect(new URL('/dashboard', req.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico|public).*)'],
}
