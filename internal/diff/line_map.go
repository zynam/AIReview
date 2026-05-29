package diff

func AddedLinesNear(lines []LineChange, line int, radius int) []LineChange {
	var nearby []LineChange
	for _, candidate := range lines {
		distance := candidate.Line - line
		if distance < 0 {
			distance = -distance
		}
		if distance <= radius {
			nearby = append(nearby, candidate)
		}
	}
	return nearby
}

func HasAddedLineNear(lines []LineChange, line int, radius int, match func(string) bool) bool {
	for _, candidate := range AddedLinesNear(lines, line, radius) {
		if match(candidate.Content) {
			return true
		}
	}
	return false
}
