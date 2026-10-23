package generator

import (
	"strings"
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	for _, length := range []int{1, 6, 16, 100} {
		code := GenerateShortCode(length)
		if len(code) != length {
			t.Errorf("len(GenerateShortCode(%d)) = %d", length, len(code))
		}
		for _, c := range code {
			if !strings.ContainsRune(alphabet, c) {
				t.Errorf("code %q contains %q outside alphabet", code, c)
			}
		}
	}
}

func TestGenerateShortCode_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		code := GenerateShortCode(8)
		if seen[code] {
			t.Fatalf("duplicate code %q after %d iterations", code, i)
		}
		seen[code] = true
	}
}
