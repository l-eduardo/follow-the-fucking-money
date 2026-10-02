package brazil

import (
	"testing"
)

func TestCleanCNPJ(t *testing.T) {
	input := "12.345.678/0001-95"
	expected := "12345678000195"
	if got := CleanCNPJ(input); got != expected {
		t.Errorf("CleanCNPJ(%q) = %q; want %q", input, got, expected)
	}
}

func TestExtractRoot(t *testing.T) {
	input := "12.345.678/0002-76"
	expected := "12345678"
	if got := ExtractRoot(input); got != expected {
		t.Errorf("ExtractRoot(%q) = %q; want %q", input, got, expected)
	}
}

func TestIsMatrix(t *testing.T) {
	matriz := "12.345.678/0001-95"
	filial := "12.345.678/0002-76"

	if !IsMatrix(matriz) {
		t.Errorf("IsMatrix(%q) should be true", matriz)
	}
	if IsMatrix(filial) {
		t.Errorf("IsMatrix(%q) should be false", filial)
	}
}

func TestIsValidCNPJ(t *testing.T) {
	tests := []struct {
		cnpj  string
		valid bool
	}{
		// Valid known CNPJs
		{"00.000.000/0001-91", true}, // Banco do Brasil
		{"33.000.167/0001-01", true}, // Petrobras
		// Invalid check digits
		{"00.000.000/0001-92", false},
		{"11.111.111/1111-11", false},
		{"123", false},
	}

	for _, tt := range tests {
		t.Run(tt.cnpj, func(t *testing.T) {
			if got := IsValidCNPJ(tt.cnpj); got != tt.valid {
				t.Errorf("IsValidCNPJ(%q) = %v; want %v", tt.cnpj, got, tt.valid)
			}
		})
	}
}
