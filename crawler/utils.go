package crawler

// removeDuplicates : removes the duplicates from the array and returns back the unique elements in the array
func removeDuplicates(input []string) []string {
	// Create a result map to track unique elements
	uniqueMap := make(map[string]bool)
	var result []string

	for _, item := range input {
		if !uniqueMap[item] {
			result = append(result, item)
			uniqueMap[item] = true
		}
	}

	return result
}
