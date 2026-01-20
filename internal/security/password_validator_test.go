package security

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
		errMsg   string
	}{
		{
			name:     "valid password with all requirements",
			password: "SecurePass123!",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "valid password minimal requirements",
			password: "Abcd123!",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "too short - 7 characters",
			password: "Abc12!",
			valid:    false,
			errMsg:   "A senha deve ter pelo menos 8 caracteres",
		},
		{
			name:     "too short - empty string",
			password: "",
			valid:    false,
			errMsg:   "A senha deve ter pelo menos 8 caracteres",
		},
		{
			name:     "no uppercase letters",
			password: "password123!",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas",
		},
		{
			name:     "no lowercase letters",
			password: "PASSWORD123!",
			valid:    false,
			errMsg:   "A senha deve conter letras minúsculas",
		},
		{
			name:     "no numbers",
			password: "PasswordABC!",
			valid:    false,
			errMsg:   "A senha deve conter números",
		},
		{
			name:     "no special characters",
			password: "Password123",
			valid:    false,
			errMsg:   "A senha deve conter pelo menos um caractere especial (!@#$%^&*)",
		},
		{
			name:     "all special characters allowed - exclamation",
			password: "Password123!",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - at sign",
			password: "Password123@",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - hash",
			password: "Password123#",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - dollar",
			password: "Password123$",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - percent",
			password: "Password123%",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - caret",
			password: "Password123^",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - ampersand",
			password: "Password123&",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "all special characters allowed - asterisk",
			password: "Password123*",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "multiple special characters",
			password: "Pass123!@#$",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "exactly 8 characters",
			password: "Pass123!",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "long password",
			password: "ThisIsAVeryLongAndSecurePassword123!",
			valid:    true,
			errMsg:   "",
		},
		{
			name:     "only uppercase and numbers with special",
			password: "PASSWORD123!",
			valid:    false,
			errMsg:   "A senha deve conter letras minúsculas",
		},
		{
			name:     "only lowercase and numbers with special",
			password: "password123!",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas",
		},
		{
			name:     "only letters and special",
			password: "PasswordABC!",
			valid:    false,
			errMsg:   "A senha deve conter números",
		},
		{
			name:     "only numbers",
			password: "12345678",
			valid:    false,
			errMsg:   "A senha deve conter letras maiúsculas",
		},
		{
			name:     "unicode characters not counted as special",
			password: "Password123€",
			valid:    false,
			errMsg:   "A senha deve conter pelo menos um caractere especial (!@#$%^&*)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errMsg := ValidatePassword(tt.password)

			if valid != tt.valid {
				t.Errorf("ValidatePassword(%q) valid = %v, want %v", tt.password, valid, tt.valid)
			}

			if errMsg != tt.errMsg {
				t.Errorf("ValidatePassword(%q) errMsg = %q, want %q", tt.password, errMsg, tt.errMsg)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		isCommon bool
	}{
		{
			name:     "common password - 123456",
			password: "123456",
			isCommon: true,
		},
		{
			name:     "common password - password",
			password: "password",
			isCommon: true,
		},
		{
			name:     "common password - qwerty",
			password: "qwerty",
			isCommon: true,
		},
		{
			name:     "common password - password123",
			password: "password123",
			isCommon: true,
		},
		{
			name:     "common password - admin",
			password: "admin",
			isCommon: true,
		},
		{
			name:     "common password - senha123 lowercase",
			password: "senha123",
			isCommon: true,
		},
		{
			name:     "common password - Senha123 mixed case",
			password: "Senha123",
			isCommon: true,
		},
		{
			name:     "common password - PASSWORD uppercase",
			password: "PASSWORD",
			isCommon: true,
		},
		{
			name:     "common password - QWERTY uppercase",
			password: "QWERTY",
			isCommon: true,
		},
		{
			name:     "common password - 12345678",
			password: "12345678",
			isCommon: true,
		},
		{
			name:     "common password - welcome",
			password: "welcome",
			isCommon: true,
		},
		{
			name:     "common password - iloveyou",
			password: "iloveyou",
			isCommon: true,
		},
		{
			name:     "common password - admin123",
			password: "admin123",
			isCommon: true,
		},
		{
			name:     "common password - letmein",
			password: "letmein",
			isCommon: true,
		},
		{
			name:     "common password - p@ssw0rd",
			password: "p@ssw0rd",
			isCommon: true,
		},
		{
			name:     "common password - P@ssw0rd mixed case",
			password: "P@ssw0rd",
			isCommon: true,
		},
		{
			name:     "not common - unique password",
			password: "MyUniqueP@ss123",
			isCommon: false,
		},
		{
			name:     "not common - random string",
			password: "xK9mP2qL8nR5",
			isCommon: false,
		},
		{
			name:     "not common - secure password",
			password: "SecurePass123!",
			isCommon: false,
		},
		{
			name:     "not common - complex password",
			password: "Tr0ng&Secure!",
			isCommon: false,
		},
		{
			name:     "not common - empty string",
			password: "",
			isCommon: false,
		},
		{
			name:     "not common - similar but different",
			password: "password1234567",
			isCommon: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCommon := IsCommonPassword(tt.password)

			if isCommon != tt.isCommon {
				t.Errorf("IsCommonPassword(%q) = %v, want %v", tt.password, isCommon, tt.isCommon)
			}
		})
	}
}
