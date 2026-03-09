'use client'

import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Search, Beaker } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Badge } from '@/components/ui/Badge'
import { Skeleton } from '@/components/ui/Skeleton'
import { DrugAlertBanner } from './DrugAlertBanner'
import { useDrugCheck } from '@/hooks/useDrugCheck'
import { cn } from '@/lib/utils'

interface MedicationSuggestion {
  name: string
  displayName: string
  rxnormCode: string
  drugClass: string
  form: string
}

const COMMON_MEDICATIONS: MedicationSuggestion[] = [
  { name: 'Metformin Hydrochloride 500mg Tablet', displayName: 'Metformin 500mg', rxnormCode: '861007', drugClass: 'Biguanide', form: 'Tablet' },
  { name: 'Atorvastatin Calcium 10mg Tablet', displayName: 'Atorvastatin 10mg', rxnormCode: '259255', drugClass: 'Statin', form: 'Tablet' },
  { name: 'Amlodipine Besylate 5mg Tablet', displayName: 'Amlodipine 5mg', rxnormCode: '329528', drugClass: 'CCB', form: 'Tablet' },
  { name: 'Losartan Potassium 50mg Tablet', displayName: 'Losartan 50mg', rxnormCode: '979480', drugClass: 'ARB', form: 'Tablet' },
  { name: 'Omeprazole 20mg Capsule', displayName: 'Omeprazole 20mg', rxnormCode: '198053', drugClass: 'PPI', form: 'Capsule' },
  { name: 'Aspirin 75mg Tablet', displayName: 'Aspirin 75mg', rxnormCode: '198464', drugClass: 'Antiplatelet', form: 'Tablet' },
  { name: 'Warfarin Sodium 5mg Tablet', displayName: 'Warfarin 5mg', rxnormCode: '855332', drugClass: 'Anticoagulant', form: 'Tablet' },
  { name: 'Clopidogrel 75mg Tablet', displayName: 'Clopidogrel 75mg', rxnormCode: '309362', drugClass: 'Antiplatelet', form: 'Tablet' },
  { name: 'Pantoprazole 40mg Tablet', displayName: 'Pantoprazole 40mg', rxnormCode: '261257', drugClass: 'PPI', form: 'Tablet' },
  { name: 'Levothyroxine 50mcg Tablet', displayName: 'Levothyroxine 50mcg', rxnormCode: '966222', drugClass: 'Thyroid', form: 'Tablet' },
  { name: 'Ramipril 5mg Capsule', displayName: 'Ramipril 5mg', rxnormCode: '261962', drugClass: 'ACE Inhibitor', form: 'Capsule' },
  { name: 'Amoxicillin 500mg Capsule', displayName: 'Amoxicillin 500mg', rxnormCode: '308182', drugClass: 'Penicillin', form: 'Capsule' },
  { name: 'Azithromycin 500mg Tablet', displayName: 'Azithromycin 500mg', rxnormCode: '141962', drugClass: 'Macrolide', form: 'Tablet' },
  { name: 'Ciprofloxacin 500mg Tablet', displayName: 'Ciprofloxacin 500mg', rxnormCode: '309309', drugClass: 'Fluoroquinolone', form: 'Tablet' },
  { name: 'Paracetamol 500mg Tablet', displayName: 'Paracetamol 500mg', rxnormCode: '198440', drugClass: 'Analgesic', form: 'Tablet' },
  { name: 'Ibuprofen 400mg Tablet', displayName: 'Ibuprofen 400mg', rxnormCode: '197806', drugClass: 'NSAID', form: 'Tablet' },
  { name: 'Furosemide 40mg Tablet', displayName: 'Furosemide 40mg', rxnormCode: '197417', drugClass: 'Loop Diuretic', form: 'Tablet' },
  { name: 'Prednisolone 5mg Tablet', displayName: 'Prednisolone 5mg', rxnormCode: '198144', drugClass: 'Corticosteroid', form: 'Tablet' },
  { name: 'Salbutamol 100mcg Inhaler', displayName: 'Salbutamol 100mcg', rxnormCode: '245314', drugClass: 'Bronchodilator', form: 'Inhaler' },
  { name: 'Cetirizine 10mg Tablet', displayName: 'Cetirizine 10mg', rxnormCode: '198052', drugClass: 'Antihistamine', form: 'Tablet' },
  { name: 'Sertraline 50mg Tablet', displayName: 'Sertraline 50mg', rxnormCode: '312938', drugClass: 'SSRI', form: 'Tablet' },
  { name: 'Insulin Glargine 100U/ml', displayName: 'Insulin Glargine', rxnormCode: '274783', drugClass: 'Insulin', form: 'Injection' },
  { name: 'Vitamin D3 60000IU Sachet', displayName: 'Vitamin D3 60000IU', rxnormCode: '636671', drugClass: 'Supplement', form: 'Sachet' },
  { name: 'Folic Acid 5mg Tablet', displayName: 'Folic Acid 5mg', rxnormCode: '315966', drugClass: 'Supplement', form: 'Tablet' },
]

interface DrugCheckPanelProps {
  patientId: string
}

export function DrugCheckPanel({ patientId }: DrugCheckPanelProps) {
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMed, setSelectedMed] = useState<MedicationSuggestion | null>(null)
  const [showDropdown, setShowDropdown] = useState(false)
  const [reviewed, setReviewed] = useState(false)
  const { result, isChecking, check, acknowledge, reset } = useDrugCheck(patientId)

  const suggestions = searchQuery.length >= 2
    ? COMMON_MEDICATIONS.filter(
        (m) =>
          m.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          m.drugClass.toLowerCase().includes(searchQuery.toLowerCase())
      ).slice(0, 8)
    : []

  const handleSelect = (med: MedicationSuggestion) => {
    setSelectedMed(med)
    setSearchQuery(med.displayName)
    setShowDropdown(false)
    reset()
    setReviewed(false)
  }

  const handleCheck = () => {
    if (selectedMed) check(selectedMed.rxnormCode)
  }

  const canPrescribe = result && (
    result.highestSeverity === 'none' ||
    (result.highestSeverity !== 'contraindicated' && reviewed)
  )

  const handlePrescribe = async () => {
    if (!selectedMed || !canPrescribe) return
    try {
      const { fhirAPI } = await import('@medilink/shared')
      await fhirAPI.createResource('MedicationRequest', {
        resourceType: 'MedicationRequest',
        status: 'active',
        intent: 'order',
        medicationCodeableConcept: {
          coding: [{ system: 'http://www.nlm.nih.gov/research/umls/rxnorm', code: selectedMed.rxnormCode, display: selectedMed.displayName }],
          text: selectedMed.displayName,
        },
        subject: { reference: `Patient/${patientId}` },
        authoredOn: new Date().toISOString(),
      })
      const { default: toast } = await import('react-hot-toast')
      toast.success('Prescription saved. Patient will be notified.')
      queryClient.invalidateQueries({ queryKey: ['patient', patientId] })
      setSelectedMed(null)
      setSearchQuery('')
      reset()
    } catch {
      const { default: toast } = await import('react-hot-toast')
      toast.error('Failed to save prescription.')
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[3fr_2fr] gap-6">
      {/* LEFT: Prescription Builder */}
      <Card padding="lg">
        <h2 className="font-display text-xl mb-4" style={{ color: 'var(--color-text-primary)' }}>
          New Prescription
        </h2>

        <div className="relative">
          <Input
            placeholder="Search medication (e.g. Metformin, Warfarin)..."
            value={searchQuery}
            onChange={(e) => { setSearchQuery(e.target.value); setShowDropdown(true); setSelectedMed(null); reset() }}
            onFocus={() => setShowDropdown(true)}
            icon={<Search size={16} />}
            label="Medication"
          />
          {showDropdown && suggestions.length > 0 && (
            <div className="absolute z-20 left-0 right-0 mt-1 max-h-64 overflow-y-auto rounded-card border border-[var(--color-border)] bg-[var(--color-bg-card)] shadow-elevated">
              {suggestions.map((med) => (
                <button
                  key={med.rxnormCode}
                  onClick={() => handleSelect(med)}
                  className="w-full text-left px-4 py-2.5 hover:bg-[var(--color-bg-elevated)] transition-colors flex items-center justify-between"
                >
                  <div>
                    <p className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>{med.displayName}</p>
                    <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{med.drugClass} · {med.form}</p>
                  </div>
                  <span className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>{med.rxnormCode}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        <Button
          className="w-full mt-4"
          onClick={handleCheck}
          disabled={!selectedMed}
          loading={isChecking}
          size="lg"
        >
          Check Interactions
        </Button>

        {result && result.highestSeverity !== 'none' && result.highestSeverity !== 'contraindicated' && (
          <label className="flex items-center gap-2 mt-4 cursor-pointer">
            <input
              type="checkbox"
              checked={reviewed}
              onChange={(e) => setReviewed(e.target.checked)}
              className="w-4 h-4 rounded accent-[var(--color-accent)]"
            />
            <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              I have reviewed the interaction warnings
            </span>
          </label>
        )}

        <Button
          className="w-full mt-4"
          variant={result?.highestSeverity === 'contraindicated' ? 'danger' : 'primary'}
          disabled={!canPrescribe}
          onClick={handlePrescribe}
          size="lg"
        >
          {result?.highestSeverity === 'contraindicated'
            ? 'Blocked — Acknowledge Required'
            : result && result.highestSeverity !== 'none'
              ? 'Prescribe with Warning'
              : 'Prescribe'}
        </Button>
      </Card>

      {/* RIGHT: Interaction Results */}
      <div>
        {isChecking ? (
          <div className="space-y-4">
            <Skeleton className="h-16" />
            <Skeleton className="h-32" />
            <Skeleton className="h-24" />
          </div>
        ) : result ? (
          result.highestSeverity === 'none' ? (
            <Card padding="lg" className="text-center">
              <div className="text-5xl mb-3">✓</div>
              <p className="font-display text-xl" style={{ color: 'var(--color-success)' }}>
                No known interactions found
              </p>
              <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
                Safe to prescribe
              </p>
            </Card>
          ) : (
            <DrugAlertBanner result={result} onAcknowledge={acknowledge} />
          )
        ) : (
          <Card padding="lg" className="text-center">
            <Beaker size={48} className="mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
            <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              Select a medication to check for interactions
            </p>
          </Card>
        )}
      </div>
    </div>
  )
}
