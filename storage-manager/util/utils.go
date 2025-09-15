package util

import "fmt"

// humanBytes returns a human-readable byte size like 1.0 KB/MB/GB
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div := int64(unit)
	exp := 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	suffixes := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	if exp >= len(suffixes) {
		return fmt.Sprintf("%.1f B", float64(n))
	}
	return fmt.Sprintf("%.1f %s", value, suffixes[exp])
}
