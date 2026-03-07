package auth

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// ValidatePasswordStrength enforces MediLink password policy.
// Returns a slice of violation messages (empty = valid).
// emailPrefix is the part before @ — password must not contain it.
func ValidatePasswordStrength(password string, emailPrefix string) []string {
	var violations []string

	if len(password) < 8 {
		violations = append(violations, "Password must be at least 8 characters")
	}
	if len(password) > 128 {
		violations = append(violations, "Password must be at most 128 characters")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	if !hasUpper {
		violations = append(violations, "Password must contain at least one uppercase letter")
	}
	if !hasLower {
		violations = append(violations, "Password must contain at least one lowercase letter")
	}
	if !hasDigit {
		violations = append(violations, "Password must contain at least one digit")
	}
	if !hasSpecial {
		violations = append(violations, "Password must contain at least one special character")
	}

	// Check common passwords
	lower := strings.ToLower(password)
	for _, common := range commonPasswords {
		if strings.ToLower(common) == lower {
			violations = append(violations, "Password is too common")
			break
		}
	}

	// Check email prefix
	if emailPrefix != "" && len(emailPrefix) >= 3 {
		if strings.Contains(strings.ToLower(password), strings.ToLower(emailPrefix)) {
			violations = append(violations, "Password must not contain your email")
		}
	}

	return violations
}

// HashPassword hashes a password using bcrypt with cost factor 12.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a password against a bcrypt hash.
func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var commonPasswords = []string{
	"Password1!", "Admin1234!", "Welcome1!", "Medilink1!",
	"Hospital1!", "Doctor123!", "Patient1!", "Health123!",
	"Qwerty123!", "Letmein1!", "Password123!", "Abc12345!",
	"Admin123!", "Welcome123!", "Change123!", "Temp1234!",
	"Test1234!", "User1234!", "Login1234!", "Access123!",
	"Secure123!", "Master123!", "Hello1234!", "Super1234!",
	"P@ssw0rd!", "P@ssword1!", "Passw0rd!", "1Qaz2wsx!",
	"Zaq12wsx!", "Trust#123", "Summer2024!", "Winter2024!",
	"Spring2024!", "Autumn2024!", "January2024!", "Monday123!",
	"Sunshine1!", "Shadow123!", "Dragon123!", "Football1!",
	"Baseball1!", "Cricket1!", "Michael1!", "Jennifer1!",
	"Princess1!", "Starwars1!", "Mustang1!", "Monkey123!",
	"Iloveyou1!", "Ashley123!", "Thomas123!", "Charlie1!",
	"Robert123!", "William1!", "Jessica1!", "Daniel123!",
	"Matthew1!", "Andrew123!", "Joshua123!", "James1234!",
	"India12345!", "Bharat123!", "Mumbai1234!", "Delhi12345!",
	"Chennai123!", "Pune123456!", "Bangalore1!", "Kolkata1!",
	"Hyderabad1!", "Ahmedabad1!", "Jaipur123!", "Lucknow1!",
	"Medico123!", "Surgeon1!", "Nursing1!", "Pharma123!",
	"Healthcare1!", "Clinical1!", "Therapy1!", "Medicine1!",
	"Diagnose1!", "Prescription1!", "Emergency1!", "Ambulance1!",
	"Stethoscope1!", "Hospital123!", "Wellness1!", "Recovery1!",
	"Treatment1!", "Operation1!", "Radiology1!", "Pathology1!",
	"Cardiology1!", "Neurology1!", "Oncology1!", "Pediatrics1!",
	"Dermatology1!", "Orthopedic1!", "Psychiatry1!", "Urology1!",
	"Nephrology1!", "Gynecology1!", "Ophthalmology!", "Anesthesia1!",
	"Gastro12345!", "Pulmonology1!", "Endocrine1!", "Rheumatology!",
	"Neonatal123!", "Geriatrics1!", "Rehabilitation!", "Intensive1!",
}
