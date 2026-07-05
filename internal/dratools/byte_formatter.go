package dratools

import "fmt"

func formatBytes(size int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	value := float64(size)
	unit := units[0]
	for _, next := range units[1:] {
		if value < 1024 {
			break
		}
		value /= 1024
		unit = next
	}
	if unit == "B" {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}
