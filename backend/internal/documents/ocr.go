package documents

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// OCRResult contains the extracted text and metadata from OCR processing.
type OCRResult struct {
	Text       string
	Confidence float64
	PageCount  int
	Language   string
}

// OCREngine defines the interface for text extraction from documents.
type OCREngine interface {
	ExtractText(ctx context.Context, fileBytes []byte, contentType string) (*OCRResult, error)
	Health(ctx context.Context) bool
}

// TesseractOCR implements OCREngine using Tesseract.
type TesseractOCR struct{}

// NewTesseractOCR creates a new TesseractOCR.
func NewTesseractOCR() *TesseractOCR {
	return &TesseractOCR{}
}

// ExtractText extracts text from a file using Tesseract.
func (t *TesseractOCR) ExtractText(ctx context.Context, fileBytes []byte, contentType string) (*OCRResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var textOutput string
	var confidence float64

	switch contentType {
	case "application/pdf":
		text, conf, err := t.processPDF(ctx, fileBytes)
		if err != nil {
			return nil, err
		}
		textOutput = text
		confidence = conf
	case "image/jpeg", "image/png", "image/webp":
		text, conf, err := t.processImage(ctx, fileBytes)
		if err != nil {
			return nil, err
		}
		textOutput = text
		confidence = conf
	default:
		return nil, fmt.Errorf("unsupported content type for OCR: %s", contentType)
	}

	return &OCRResult{
		Text:       textOutput,
		Confidence: confidence,
		PageCount:  1,
		Language:   "eng",
	}, nil
}

// Health checks if Tesseract is available.
func (t *TesseractOCR) Health(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "tesseract", "--version")
	return cmd.Run() == nil
}

func (t *TesseractOCR) processImage(ctx context.Context, imageBytes []byte) (string, float64, error) {
	// Get text output
	textCmd := exec.CommandContext(ctx, "tesseract",
		"stdin", "stdout",
		"-l", "eng+hin+mar",
		"--oem", "3",
		"--psm", "6",
	)
	textCmd.Stdin = bytes.NewReader(imageBytes)
	textOutput, err := textCmd.Output()
	if err != nil {
		return "", 0, fmt.Errorf("tesseract text extraction: %w", err)
	}

	// Get confidence via TSV output
	tsvCmd := exec.CommandContext(ctx, "tesseract",
		"stdin", "stdout",
		"-l", "eng+hin+mar",
		"--oem", "3",
		"--psm", "6",
		"tsv",
	)
	tsvCmd.Stdin = bytes.NewReader(imageBytes)
	tsvOutput, err := tsvCmd.Output()
	if err != nil {
		// Return text without confidence if TSV fails
		return string(textOutput), 50.0, nil
	}

	confidence := parseConfidenceFromTSV(string(tsvOutput))
	return string(textOutput), confidence, nil
}

func (t *TesseractOCR) processPDF(ctx context.Context, pdfBytes []byte) (string, float64, error) {
	// For PDF, we try using Tesseract directly if it supports PDF input
	// Otherwise, treat the entire PDF as an image (which Tesseract can sometimes handle)
	textCmd := exec.CommandContext(ctx, "tesseract",
		"stdin", "stdout",
		"-l", "eng+hin+mar",
		"--oem", "3",
		"--psm", "6",
	)
	textCmd.Stdin = bytes.NewReader(pdfBytes)
	textOutput, err := textCmd.Output()
	if err != nil {
		return "", 0, fmt.Errorf("tesseract PDF extraction: %w", err)
	}

	return string(textOutput), 70.0, nil // default confidence for PDFs
}

func parseConfidenceFromTSV(tsv string) float64 {
	lines := strings.Split(tsv, "\n")
	if len(lines) < 2 {
		return 0
	}

	var total, count float64
	for i, line := range lines {
		if i == 0 {
			continue // skip header
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 12 {
			continue
		}
		conf, err := strconv.ParseFloat(strings.TrimSpace(fields[10]), 64)
		if err != nil || conf < 0 {
			continue
		}
		total += conf
		count++
	}

	if count == 0 {
		return 0
	}
	return total / count
}

// NoopOCREngine is a no-op implementation for testing.
type NoopOCREngine struct{}

func (n *NoopOCREngine) ExtractText(_ context.Context, _ []byte, _ string) (*OCRResult, error) {
	return &OCRResult{
		Text:       "Sample OCR text for testing",
		Confidence: 95.0,
		PageCount:  1,
		Language:   "eng",
	}, nil
}

func (n *NoopOCREngine) Health(_ context.Context) bool { return true }
