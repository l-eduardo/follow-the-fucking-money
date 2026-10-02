package brazil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var cpfDigitsRegex = regexp.MustCompile(`\D`)

// CleanCPF extracts only digits from a CPF string.
func CleanCPF(cpf string) string {
	return cpfDigitsRegex.ReplaceAllString(cpf, "")
}

// ExtractMaskedMiddle extracts the 6 central digits from a full CPF or masked string.
// Example: "***123456**" -> "123456", "111.123.456-78" -> "123456"
func ExtractMaskedMiddle(cpf string) string {
	digits := CleanCPF(cpf)
	if len(digits) == 6 {
		return digits
	}
	if len(digits) == 11 {
		return digits[3:9]
	}
	// Fallback to substring regex if original string contained asterisks
	re := regexp.MustCompile(`\*{3}(\d{6})\*{2}`)
	matches := re.FindStringSubmatch(cpf)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// FormatMaskedCPF formats the 6 central digits as ***.XXX.XXX-**
func FormatMaskedCPF(middle6 string) string {
	if len(middle6) != 6 {
		return middle6
	}
	return fmt.Sprintf("***.%s.%s-**", middle6[0:3], middle6[3:6])
}

// IsMaskedCPF checks if the CPF string represents an LGPD-masked format.
func IsMaskedCPF(cpf string) bool {
	return strings.Contains(cpf, "*") || len(CleanCPF(cpf)) == 6
}

// IsValidCPF checks whether an unmasked 11-digit CPF is mathematically valid.
func IsValidCPF(cpf string) bool {
	clean := CleanCPF(cpf)
	if len(clean) != 11 {
		return false
	}

	allSame := true
	for i := 1; i < 11; i++ {
		if clean[i] != clean[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}

	calcDigit := func(slice string, factor int) int {
		sum := 0
		for i := 0; i < len(slice); i++ {
			d, _ := strconv.Atoi(string(slice[i]))
			sum += d * factor
			factor--
		}
		rem := (sum * 10) % 11
		if rem == 10 {
			return 0
		}
		return rem
	}

	d1 := calcDigit(clean[:9], 10)
	if d1 != int(clean[9]-'0') {
		return false
	}

	d2 := calcDigit(clean[:10], 11)
	return d2 == int(clean[10]-'0')
}
