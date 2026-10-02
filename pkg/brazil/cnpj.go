package brazil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var nonDigitsRegex = regexp.MustCompile(`\D`)

// CleanCNPJ removes all non-numeric characters from the input string.
func CleanCNPJ(cnpj string) string {
	return nonDigitsRegex.ReplaceAllString(cnpj, "")
}

// ExtractRoot extracts the 8-digit root (CNPJ Básico) from a CNPJ string.
func ExtractRoot(cnpj string) string {
	clean := CleanCNPJ(cnpj)
	if len(clean) >= 8 {
		return clean[:8]
	}
	return clean
}

// IsMatrix checks whether the CNPJ belongs to the headquarters (matriz, order 0001).
func IsMatrix(cnpj string) bool {
	clean := CleanCNPJ(cnpj)
	if len(clean) != 14 {
		return false
	}
	return clean[8:12] == "0001"
}

// FormatCNPJ formats a 14-digit CNPJ as 00.000.000/0000-00.
func FormatCNPJ(cnpj string) string {
	clean := CleanCNPJ(cnpj)
	if len(clean) != 14 {
		return cnpj
	}
	return fmt.Sprintf("%s.%s.%s/%s-%s",
		clean[0:2], clean[2:5], clean[5:8], clean[8:12], clean[12:14])
}

// IsValidCNPJ verifies the mathematical check digits of a 14-digit CNPJ.
func IsValidCNPJ(cnpj string) bool {
	clean := CleanCNPJ(cnpj)
	if len(clean) != 14 {
		return false
	}

	// Reject repetitive strings like 00000000000000, 11111111111111, etc.
	allSame := true
	for i := 1; i < 14; i++ {
		if clean[i] != clean[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	calcDigit := func(slice string, weights []int) int {
		sum := 0
		for i, w := range weights {
			d, _ := strconv.Atoi(string(slice[i]))
			sum += d * w
		}
		rem := sum % 11
		if rem < 2 {
			return 0
		}
		return 11 - rem
	}

	d1 := calcDigit(clean[:12], weights1)
	if d1 != int(clean[12]-'0') {
		return false
	}

	d2 := calcDigit(clean[:13], weights2)
	return d2 == int(clean[13]-'0')
}

// SanitizeReason normalizes company names (uppercase, trim spaces).
func SanitizeReason(name string) string {
	return strings.ToUpper(strings.Join(strings.Fields(name), " "))
}
