import { describe, it, expect } from 'vitest'
import { getSeverityDisplay, getConsentDisplay, getJobStatusDisplay } from '@medilink/shared'

describe('Severity Utils', () => {
  describe('getSeverityDisplay', () => {
    it('returns display for none severity', () => {
      const result = getSeverityDisplay('none')
      expect(result.label).toBe('No Interactions')
      expect(result.color).toBeDefined()
    })

    it('returns display for major severity', () => {
      const result = getSeverityDisplay('major')
      expect(result.label).toBe('Major')
    })

    it('returns display for contraindicated', () => {
      const result = getSeverityDisplay('contraindicated')
      expect(result.label).toBe('Contraindicated')
    })

    it('returns display for moderate', () => {
      const result = getSeverityDisplay('moderate')
      expect(result.label).toBe('Moderate')
    })

    it('returns display for minor', () => {
      const result = getSeverityDisplay('minor')
      expect(result.label).toBe('Minor')
    })

    it('returns display for unknown', () => {
      const result = getSeverityDisplay('unknown')
      expect(result.label).toBe('Unknown')
    })

    it('includes bgColor and borderColor', () => {
      const result = getSeverityDisplay('major')
      expect(result.bgColor).toBeDefined()
      expect(result.borderColor).toBeDefined()
      expect(result.icon).toBeDefined()
    })
  })

  describe('getConsentDisplay', () => {
    it('returns display for active consent', () => {
      const result = getConsentDisplay('active')
      expect(result.label).toBe('Active')
    })

    it('returns display for revoked consent', () => {
      const result = getConsentDisplay('revoked')
      expect(result.label).toBe('Revoked')
    })

    it('returns display for expired consent', () => {
      const result = getConsentDisplay('expired')
      expect(result.label).toBe('Expired')
    })
  })

  describe('getJobStatusDisplay', () => {
    it('returns display for pending', () => {
      const result = getJobStatusDisplay('pending')
      expect(result.label).toBe('Pending')
    })

    it('returns display for processing', () => {
      const result = getJobStatusDisplay('processing')
      expect(result.label).toBe('Processing')
    })

    it('returns display for completed', () => {
      const result = getJobStatusDisplay('completed')
      expect(result.label).toBe('Completed')
    })

    it('returns display for failed', () => {
      const result = getJobStatusDisplay('failed')
      expect(result.label).toBe('Failed')
    })

    it('returns display for needs-manual-review', () => {
      const result = getJobStatusDisplay('needs-manual-review')
      expect(result.label).toBe('Needs Review')
    })
  })
})
