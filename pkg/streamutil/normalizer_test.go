package streamutil

import (
	"testing"
)

func TestNormalizeText(t *testing.T) {
	input := "São Paulo - República Federativa do Brasil"
	expected := "SAO PAULO - REPUBLICA FEDERATIVA DO BRASIL"
	if got := NormalizeText(input); got != expected {
		t.Errorf("NormalizeText(%q) = %q; want %q", input, got, expected)
	}
}

func TestNormalizePersonName(t *testing.T) {
	input := "José da Silva e Santos de Souza"
	expected := "JOSE SILVA SANTOS SOUZA"
	if got := NormalizePersonName(input); got != expected {
		t.Errorf("NormalizePersonName(%q) = %q; want %q", input, got, expected)
	}
}
