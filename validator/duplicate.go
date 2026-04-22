package validator

import "errors"

var ErrDuplicate = errors.New("Afwan, Terdapat Data Duplicate pada data anda.")

func CheckDuplicate(values []string) error {
	seen := make(map[string]struct{}, len(values))

	for _, v := range values {
		if _, exists := seen[v]; exists {
			return ErrDuplicate
		}
		seen[v] = struct{}{}
	}
	return nil
}
