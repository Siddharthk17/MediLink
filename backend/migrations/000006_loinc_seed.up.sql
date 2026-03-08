-- 000006_loinc_seed.up.sql
-- Seeds the most common Indian lab test LOINC mappings.

INSERT INTO loinc_mappings (test_name, test_name_lower, loinc_code, loinc_display, unit, normal_low, normal_high, category) VALUES
-- Diabetes panel
('Hemoglobin A1c', 'hemoglobin a1c', '4548-4', 'Hemoglobin A1c/Hemoglobin.total in Blood', '%', 4.0, 6.0, 'laboratory'),
('HbA1c', 'hba1c', '4548-4', 'Hemoglobin A1c/Hemoglobin.total in Blood', '%', 4.0, 6.0, 'laboratory'),
('Glycated Haemoglobin', 'glycated haemoglobin', '4548-4', 'Hemoglobin A1c/Hemoglobin.total in Blood', '%', 4.0, 6.0, 'laboratory'),
('Fasting Blood Glucose', 'fasting blood glucose', '1558-6', 'Fasting glucose [Mass/volume] in Capillary blood', 'mg/dL', 70, 100, 'laboratory'),
('FBS', 'fbs', '1558-6', 'Fasting glucose [Mass/volume] in Capillary blood', 'mg/dL', 70, 100, 'laboratory'),
('Random Blood Sugar', 'random blood sugar', '2339-0', 'Glucose [Mass/volume] in Blood', 'mg/dL', 70, 140, 'laboratory'),
('RBS', 'rbs', '2339-0', 'Glucose [Mass/volume] in Blood', 'mg/dL', 70, 140, 'laboratory'),
('Post Prandial Glucose', 'post prandial glucose', '1521-4', 'Glucose [Mass/volume] in Serum or Plasma --2 hours post meal', 'mg/dL', 70, 140, 'laboratory'),
('PP Blood Sugar', 'pp blood sugar', '1521-4', 'Glucose [Mass/volume] in Serum or Plasma --2 hours post meal', 'mg/dL', 70, 140, 'laboratory'),

-- Complete Blood Count (CBC)
('Haemoglobin', 'haemoglobin', '718-7', 'Hemoglobin [Mass/volume] in Blood', 'g/dL', 12.0, 17.5, 'laboratory'),
('Hemoglobin', 'hemoglobin', '718-7', 'Hemoglobin [Mass/volume] in Blood', 'g/dL', 12.0, 17.5, 'laboratory'),
('Hb', 'hb', '718-7', 'Hemoglobin [Mass/volume] in Blood', 'g/dL', 12.0, 17.5, 'laboratory'),
('WBC Count', 'wbc count', '6690-2', 'Leukocytes [#/volume] in Blood by Automated count', '10^3/uL', 4.5, 11.0, 'laboratory'),
('Total Leukocyte Count', 'total leukocyte count', '6690-2', 'Leukocytes [#/volume] in Blood by Automated count', '10^3/uL', 4.5, 11.0, 'laboratory'),
('TLC', 'tlc', '6690-2', 'Leukocytes [#/volume] in Blood by Automated count', '10^3/uL', 4.5, 11.0, 'laboratory'),
('Platelet Count', 'platelet count', '777-3', 'Platelets [#/volume] in Blood by Automated count', '10^3/uL', 150, 400, 'laboratory'),
('RBC Count', 'rbc count', '789-8', 'Erythrocytes [#/volume] in Blood by Automated count', '10^6/uL', 4.0, 5.5, 'laboratory'),
('Haematocrit', 'haematocrit', '4544-3', 'Hematocrit [Volume Fraction] of Blood by Automated count', '%', 36, 52, 'laboratory'),
('PCV', 'pcv', '4544-3', 'Hematocrit [Volume Fraction] of Blood by Automated count', '%', 36, 52, 'laboratory'),
('MCV', 'mcv', '787-2', 'MCV [Entitic volume] by Automated count', 'fL', 80, 100, 'laboratory'),
('MCH', 'mch', '785-6', 'MCH [Entitic mass] by Automated count', 'pg', 27, 33, 'laboratory'),
('MCHC', 'mchc', '786-4', 'MCHC [Mass/volume] by Automated count', 'g/dL', 32, 36, 'laboratory'),

-- Kidney Function Tests (KFT)
('Serum Creatinine', 'serum creatinine', '2160-0', 'Creatinine [Mass/volume] in Serum or Plasma', 'mg/dL', 0.6, 1.2, 'laboratory'),
('Creatinine', 'creatinine', '2160-0', 'Creatinine [Mass/volume] in Serum or Plasma', 'mg/dL', 0.6, 1.2, 'laboratory'),
('Blood Urea Nitrogen', 'blood urea nitrogen', '3094-0', 'Urea nitrogen [Mass/volume] in Serum or Plasma', 'mg/dL', 7, 20, 'laboratory'),
('BUN', 'bun', '3094-0', 'Urea nitrogen [Mass/volume] in Serum or Plasma', 'mg/dL', 7, 20, 'laboratory'),
('Serum Urea', 'serum urea', '3091-6', 'Urea [Mass/volume] in Serum or Plasma', 'mg/dL', 15, 40, 'laboratory'),
('Uric Acid', 'uric acid', '3084-1', 'Urate [Mass/volume] in Serum or Plasma', 'mg/dL', 2.4, 7.0, 'laboratory'),
('eGFR', 'egfr', '62238-1', 'GFR/1.73 sq M.predicted [Volume Rate/Area] in Serum, Plasma or Blood', 'mL/min/1.73m2', 60, 120, 'laboratory'),

-- Liver Function Tests (LFT)
('SGPT', 'sgpt', '1742-6', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 7, 40, 'laboratory'),
('ALT', 'alt', '1742-6', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 7, 40, 'laboratory'),
('SGOT', 'sgot', '1920-8', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 10, 40, 'laboratory'),
('AST', 'ast', '1920-8', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 10, 40, 'laboratory'),
('Total Bilirubin', 'total bilirubin', '1975-2', 'Bilirubin.total [Mass/volume] in Serum or Plasma', 'mg/dL', 0.1, 1.2, 'laboratory'),
('Direct Bilirubin', 'direct bilirubin', '1968-7', 'Bilirubin.direct [Mass/volume] in Serum or Plasma', 'mg/dL', 0.0, 0.3, 'laboratory'),
('Alkaline Phosphatase', 'alkaline phosphatase', '6768-6', 'Alkaline phosphatase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 44, 147, 'laboratory'),
('ALP', 'alp', '6768-6', 'Alkaline phosphatase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 44, 147, 'laboratory'),
('Total Protein', 'total protein', '2885-2', 'Protein [Mass/volume] in Serum or Plasma', 'g/dL', 6.3, 8.2, 'laboratory'),
('Albumin', 'albumin', '1751-7', 'Albumin [Mass/volume] in Serum or Plasma', 'g/dL', 3.5, 5.0, 'laboratory'),

-- Lipid Profile
('Total Cholesterol', 'total cholesterol', '2093-3', 'Cholesterol [Mass/volume] in Serum or Plasma', 'mg/dL', 0, 200, 'laboratory'),
('HDL Cholesterol', 'hdl cholesterol', '2085-9', 'Cholesterol in HDL [Mass/volume] in Serum or Plasma', 'mg/dL', 40, 60, 'laboratory'),
('LDL Cholesterol', 'ldl cholesterol', '2089-1', 'Cholesterol in LDL [Mass/volume] in Serum or Plasma', 'mg/dL', 0, 100, 'laboratory'),
('Triglycerides', 'triglycerides', '2571-8', 'Triglyceride [Mass/volume] in Serum or Plasma', 'mg/dL', 0, 150, 'laboratory'),
('VLDL Cholesterol', 'vldl cholesterol', '13458-5', 'Cholesterol in VLDL [Mass/volume] in Serum or Plasma', 'mg/dL', 0, 30, 'laboratory'),

-- Thyroid Function
('TSH', 'tsh', '3016-3', 'Thyrotropin [Units/volume] in Serum or Plasma', 'mIU/L', 0.4, 4.0, 'laboratory'),
('T3', 't3', '3051-0', 'Triiodothyronine (T3) [Mass/volume] in Serum or Plasma', 'ng/dL', 80, 200, 'laboratory'),
('T4', 't4', '3026-2', 'Thyroxine (T4) [Mass/volume] in Serum or Plasma', 'ug/dL', 5.1, 14.1, 'laboratory'),
('Free T3', 'free t3', '3052-8', 'Triiodothyronine (T3) Free [Mass/volume] in Serum or Plasma', 'pg/mL', 2.0, 4.4, 'laboratory'),
('Free T4', 'free t4', '3024-7', 'Thyroxine (T4) Free [Mass/volume] in Serum or Plasma', 'ng/dL', 0.8, 1.8, 'laboratory'),

-- Vitamins and Minerals
('Vitamin D', 'vitamin d', '1989-3', '25-hydroxyvitamin D3 [Mass/volume] in Serum or Plasma', 'ng/mL', 30, 100, 'laboratory'),
('Vitamin B12', 'vitamin b12', '2132-9', 'Cobalamin (Vitamin B12) [Mass/volume] in Serum or Plasma', 'pg/mL', 200, 900, 'laboratory'),
('Serum Iron', 'serum iron', '2498-4', 'Iron [Mass/volume] in Serum or Plasma', 'ug/dL', 60, 170, 'laboratory'),
('Ferritin', 'ferritin', '2276-4', 'Ferritin [Mass/volume] in Serum or Plasma', 'ng/mL', 12, 300, 'laboratory'),
('Folate', 'folate', '2284-8', 'Folate [Mass/volume] in Serum or Plasma', 'ng/mL', 3.1, 17.5, 'laboratory'),
('Calcium', 'calcium', '17861-6', 'Calcium [Mass/volume] in Serum or Plasma', 'mg/dL', 8.5, 10.5, 'laboratory'),

-- Vitals (for OCR from discharge summaries)
('Systolic BP', 'systolic bp', '8480-6', 'Systolic blood pressure', 'mmHg', 90, 120, 'vital-signs'),
('Diastolic BP', 'diastolic bp', '8462-4', 'Diastolic blood pressure', 'mmHg', 60, 80, 'vital-signs'),
('Heart Rate', 'heart rate', '8867-4', 'Heart rate', '/min', 60, 100, 'vital-signs'),
('Pulse Rate', 'pulse rate', '8867-4', 'Heart rate', '/min', 60, 100, 'vital-signs'),
('Temperature', 'temperature', '8310-5', 'Body temperature', 'Cel', 36.1, 37.2, 'vital-signs'),
('SpO2', 'spo2', '2708-6', 'Oxygen saturation in Arterial blood', '%', 95, 100, 'vital-signs'),
('Respiratory Rate', 'respiratory rate', '9279-1', 'Respiratory rate', '/min', 12, 20, 'vital-signs'),
('BMI', 'bmi', '39156-5', 'Body mass index (BMI) [Ratio]', 'kg/m2', 18.5, 24.9, 'vital-signs'),
('Weight', 'weight', '29463-7', 'Body weight', 'kg', NULL, NULL, 'vital-signs'),
('Height', 'height', '8302-2', 'Body height', 'cm', NULL, NULL, 'vital-signs');
