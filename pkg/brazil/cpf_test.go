package brazil

import (
	"testing"
)

func TestExtractMaskedMiddle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"***123456**", "123456"},
		{"***.123.456-**", "123456"},
		{"123456", "123456"},
		{"111.456.789-00", "456789"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractMaskedMiddle(tt.input); got != tt.expected {
				t.Errorf("ExtractMaskedMiddle(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatMaskedCPF(t *testing.T) {
	input := "123456"
	expected := "***.123.456-**"
	if got := FormatMaskedCPF(input); got != expected {
		t.Errorf("FormatMaskedCPF(%q) = %q; want %q", input, got, expected)
	}
}

func TestIsValidCPF(t *testing.T) {
	tests := []struct {
		cpf   string
		valid bool
	}{
		{"111.111.111-11", false},
		{"123.456.789-00", false},
		{"000.000.000-00", false},
	}

	for _, tt := range tests {
		t.Run(tt.cpf, func(t *testing.T) {
			if got := IsValidCPF(tt.cpf); got != tt.valid {
				t.Errorf("IsValidCPF(%q) = %v; want %v", tt.cpf, got, tt.valid)
			}
		})
	}
}
