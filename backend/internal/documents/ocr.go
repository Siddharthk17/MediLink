package documents

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	tmpDir, err := os.MkdirTemp("", "medilink-ocr-*")
	if err != nil {
		return "", 0, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0600); err != nil {
		return "", 0, fmt.Errorf("write temp pdf: %w", err)
	}

	// Convert PDF pages to PNG images using pdftoppm
	imgPrefix := filepath.Join(tmpDir, "page")
	convertCmd := exec.CommandContext(ctx, "pdftoppm",
		"-png", "-r", "300", pdfPath, imgPrefix,
	)
	if output, err := convertCmd.CombinedOutput(); err != nil {
		return "", 0, fmt.Errorf("pdftoppm conversion: %w: %s", err, string(output))
	}

	// Find generated page images
	pages, err := filepath.Glob(imgPrefix + "-*.png")
	if err != nil || len(pages) == 0 {
		return "", 0, fmt.Errorf("no pages extracted from PDF")
	}

	// OCR each page image
	var allText strings.Builder
	var totalConf, confCount float64

	for _, pagePath := range pages {
		pageBytes, err := os.ReadFile(pagePath)
		if err != nil {
			continue
		}
		text, conf, err := t.processImage(ctx, pageBytes)
		if err != nil {
			continue
		}
		allText.WriteString(text)
		allText.WriteString("\n")
		totalConf += conf
		confCount++
	}

	if confCount == 0 {
		return "", 0, fmt.Errorf("OCR failed on all PDF pages")
	}

	avgConf := totalConf / confCount
	return allText.String(), avgConf, nil
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
