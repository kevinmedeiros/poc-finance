package security

import "regexp"

// ValidatePassword checks password complexity requirements
// Returns (valid, errorMessage) where valid indicates if password meets all requirements
// and errorMessage contains a user-friendly error message in Portuguese if validation fails
func ValidatePassword(password string) (bool, string) {
	// Check minimum length
	if len(password) < 8 {
		return false, "A senha deve ter pelo menos 8 caracteres"
	}

	// Check for uppercase letters
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	if !hasUpper {
		return false, "A senha deve conter letras maiúsculas"
	}

	// Check for lowercase letters
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	if !hasLower {
		return false, "A senha deve conter letras minúsculas"
	}

	// Check for numbers
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	if !hasNumber {
		return false, "A senha deve conter números"
	}

	// Check for special characters
	hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)
	if !hasSpecial {
		return false, "A senha deve conter pelo menos um caractere especial (!@#$%^&*)"
	}

	// Check against common passwords list
	if IsCommonPassword(password) {
		return false, "Esta senha é muito comum. Por favor, escolha uma senha mais segura"
	}

	return true, ""
}
