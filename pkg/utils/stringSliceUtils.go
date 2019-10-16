package k8s

// AddStringIfMissing adds a string to a slice of strings
// returns true, list with appended string if string was not in the list
// returns false, old list if string was already in the list
func AddStringIfMissing(slice []string, s string) (bool, []string) {
	for _, item := range slice {
		if item == s {
			return false, slice
		}
	}
	return true, append(slice, s)
}

// RemoveString removes a string from a slice of strings
// returns true, list with removed string if string was contained in the list
// returns false, old list if string was not in the list
func RemoveString(slice []string, s string) (bool, []string) {
	removed := false
	result := []string{}
	for _, item := range slice {
		if item == s {
			removed = true
			continue
		}
		result = append(result, item)
	}
	return removed, result
}
