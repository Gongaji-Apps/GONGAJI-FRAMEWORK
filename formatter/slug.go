package formatter

import "strings"

// =======================================================
// ==================== GENERATE SLUG ====================
// =======================================================
func Generate_Slug(value string) string {
	var b strings.Builder
	prevDash := false

	for _, r := range strings.ToLower(value) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}

	return strings.TrimRight(b.String(), "-")
}
