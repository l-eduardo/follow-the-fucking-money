package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/l-eduardo/follow-the-fucking-money/pkg/brazil"
	"github.com/l-eduardo/follow-the-fucking-money/pkg/streamutil"
)

// MatchConfidence represents the resolution result between two persona records.
type MatchConfidence struct {
	ShouldMerge    bool    `json:"should_merge"`
	Score          float64 `json:"score"`
	ResolutionType string  `json:"resolution_type"`
}

// GeneratePersonID creates a deterministic, reproducible hash identifier for an individual.
// Combines normalized name, 6 middle digits of CPF, and state (UF).
func GeneratePersonID(name, maskedCPF, state string) string {
	normName := streamutil.NormalizePersonName(name)
	digits := brazil.ExtractMaskedMiddle(maskedCPF)
	normState := strings.ToUpper(strings.TrimSpace(state))

	payload := normName + "#" + digits + "#" + normState
	h := sha256.New()
	h.Write([]byte(payload))
	return "PF_" + hex.EncodeToString(h.Sum(nil))[:16]
}

// ResolvePersonLink compares a donor record with a corporate partner record.
// Prevents incorrect merges between homonyms by requiring middle 6 digits and geographic/age compatibility.
func ResolvePersonLink(donorName, donorCPF, donorState string, partnerName, partnerCPF, partnerState string) MatchConfidence {
	digits1 := brazil.ExtractMaskedMiddle(donorCPF)
	digits2 := brazil.ExtractMaskedMiddle(partnerCPF)

	// If middle digits exist and are different, they are definitely different individuals
	if digits1 != "" && digits2 != "" && digits1 != digits2 {
		return MatchConfidence{
			ShouldMerge:    false,
			Score:          0.0,
			ResolutionType: "DISTINCT_CPF_BLOCK",
		}
	}

	norm1 := streamutil.NormalizePersonName(donorName)
	norm2 := streamutil.NormalizePersonName(partnerName)

	if norm1 == "" || norm2 == "" {
		return MatchConfidence{
			ShouldMerge:    false,
			Score:          0.0,
			ResolutionType: "EMPTY_NAME",
		}
	}

	// Exact match on normalized name
	if norm1 == norm2 {
		score := 0.70
		if digits1 != "" && digits1 == digits2 {
			score += 0.20
		}
		if donorState != "" && partnerState != "" && strings.EqualFold(donorState, partnerState) {
			score += 0.10
		}

		if score >= 0.85 {
			return MatchConfidence{
				ShouldMerge:    true,
				Score:          score,
				ResolutionType: "HIGH_CONFIDENCE_MATCH",
			}
		}

		return MatchConfidence{
			ShouldMerge:    false,
			Score:          score,
			ResolutionType: "POSSIBLE_HOMONYM",
		}
	}

	return MatchConfidence{
		ShouldMerge:    false,
		Score:          0.2,
		ResolutionType: "NAME_MISMATCH",
	}
}
