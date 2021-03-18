package jsonnet

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runeCmp(a, b rune) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	} else {
		return 0
	}
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	} else {
		return 0
	}
}

func float64Cmp(a, b float64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	} else {
		return 0
	}
}
