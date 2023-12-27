package crawler

func removeDuplicates(input []string) []string {
	// Create a map to track unique elements
	uniqueMap := make(map[string]bool)
	var result []string

	// Iterate over the input slice
	for _, item := range input {
		// If the item is not already in the map, add it to the result slice and the map
		if !uniqueMap[item] {
			result = append(result, item)
			uniqueMap[item] = true
		}
	}

	return result
}
