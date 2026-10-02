package resolver

import (
	"testing"
)

func TestGeneratePersonID(t *testing.T) {
	id1 := GeneratePersonID("João da Silva", "***123456**", "SP")
	id2 := GeneratePersonID("JOAO SILVA", "123456", "sp")
	id3 := GeneratePersonID("Maria dos Santos", "***123456**", "SP")

	if id1 != id2 {
		t.Errorf("GeneratePersonID should be deterministic for identical normalized identities: got %s != %s", id1, id2)
	}

	if id1 == id3 {
		t.Errorf("GeneratePersonID should differ for distinct names: got %s == %s", id1, id3)
	}
}

func TestResolvePersonLink(t *testing.T) {
	// High confidence match: same name, same middle 6, same UF
	res1 := ResolvePersonLink("Eduardo Souza", "***987654**", "SC", "EDUARDO DE SOUZA", "***987654**", "SC")
	if !res1.ShouldMerge || res1.Score < 0.85 {
		t.Errorf("Expected high confidence match, got: %+v", res1)
	}

	// Distinct CPF block
	res2 := ResolvePersonLink("José Silva", "***111222**", "SP", "José Silva", "***333444**", "SP")
	if res2.ShouldMerge || res2.ResolutionType != "DISTINCT_CPF_BLOCK" {
		t.Errorf("Expected distinct CPF block, got: %+v", res2)
	}
}
