package i32

// Determine if there is a duplicate value in the slice
func ContainsDuplicate(s []int32) bool {
	k := make(map[int32]bool, len(s))
	for _, ss := range s {
		if k[ss] {
			return true
		} else {
			k[ss] = true
		}
	}
	return false
}

// Return the elements in slice a that are not in slice b
func Diff(a []int32, b []int32) []int32 {
	k := make(map[int32]bool, len(b))
	for _, bb := range b {
		k[bb] = true
	}
	diff := []int32{}
	for _, aa := range a {
		if _, exists := k[aa]; !exists {
			diff = append(diff, aa)
		}
	}
	return diff
}

// Return the maximum element in the slice
func Max(s []int32) int32 {
	var max int32 = s[0]
	for _, v := range s {
		if max < v {
			max = v
		}
	}
	return max
}
