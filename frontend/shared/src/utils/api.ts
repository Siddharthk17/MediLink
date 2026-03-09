export function parseAPIError(error: unknown): string {
  if (!error || typeof error !== 'object') return 'An unexpected error occurred'

  const axiosError = error as { response?: { data?: unknown; status?: number } }
  const data = axiosError.response?.data

  if (data && typeof data === 'object' && 'resourceType' in data) {
    const outcome = data as { issue?: Array<{ details?: { text?: string }; diagnostics?: string }> }
    const text = outcome.issue?.[0]?.details?.text
    if (text) return text
  }

  const status = axiosError.response?.status
  if (status === 403) return 'Access denied. Consent may have been revoked.'
  if (status === 409) return 'Drug interaction requires acknowledgment before prescribing.'
  if (status === 401) return 'Session expired. Redirecting to login...'
  if (status === 422) return 'Invalid request. Please check the form and try again.'
  if (status === 500) return 'Server error. Please try again in a moment.'

  return 'Something went wrong. Please try again.'
}
