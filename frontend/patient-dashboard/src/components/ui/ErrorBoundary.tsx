'use client'

import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback || (
          <div className="flex flex-col items-center justify-center min-h-[40vh] gap-4">
            <div className="w-12 h-12 rounded-full bg-[var(--color-danger-subtle)] flex items-center justify-center">
              <span className="text-[var(--color-danger)] text-xl">!</span>
            </div>
            <p className="text-sm text-[var(--color-text-muted)]">
              Something went wrong. Please refresh the page.
            </p>
            <button
              onClick={() => this.setState({ hasError: false, error: null })}
              className="text-sm text-[var(--color-accent)] hover:underline"
            >
              Try again
            </button>
          </div>
        )
      )
    }

    return this.props.children
  }
}
