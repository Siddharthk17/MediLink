package llm

import "fmt"

// ExtractionSystemPrompt is the system prompt for the LLM extraction.
const ExtractionSystemPrompt = `You are a medical data extraction specialist.
You extract structured lab test results from OCR text of Indian medical lab reports.
You output ONLY valid JSON matching the exact schema provided.
You NEVER add explanations, markdown, or text outside the JSON.
You NEVER invent data that is not present in the OCR text.
If a value cannot be extracted with confidence, omit it.`

// BuildExtractionPrompt builds the user prompt with OCR text injected.
func BuildExtractionPrompt(ocrText string) string {
	return fmt.Sprintf(`Extract all lab test results from the following OCR text of an Indian medical lab report.

OUTPUT ONLY this exact JSON structure, nothing else:
{
  "reportDate": "YYYY-MM-DD or empty string if not found",
  "labName": "laboratory name or empty string",
  "patientName": "patient name from report or empty string",
  "orderingDoctor": "doctor name or empty string",
  "reportType": "type of report e.g. CBC, LFT, KFT, Lipid Profile, or empty string",
  "results": [
    {
      "testName": "exact test name as on report",
      "value": 0.0,
      "valueString": "for non-numeric results like Positive/Negative",
      "isNumeric": true,
      "unit": "unit of measurement",
      "referenceRange": "reference range as string e.g. 4.0 - 6.0",
      "refRangeLow": 0.0,
      "refRangeHigh": 0.0,
      "isAbnormal": false,
      "abnormalFlag": "H or L or HH or LL or empty string"
    }
  ]
}

Rules:
- Include ALL test results found in the text, even if partial
- For isNumeric: true only if value is a real number
- For isAbnormal: true if the value is outside the reference range
- For abnormalFlag: H=High, L=Low, HH=Critically High, LL=Critically Low
- refRangeLow and refRangeHigh should be 0 if reference range is not parseable
- If a field cannot be determined, use empty string or 0.0, never null
- Do NOT include LOINC codes — those are added separately
- Do NOT include the patient's personal details beyond patientName

OCR Text:
%s`, ocrText)
}
