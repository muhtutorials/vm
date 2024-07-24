package cpu

import (
	"fmt"
	"os"
)

// debugPrintf outputs when "DEBUG=1"
func debugPrintf(format string, args ...any) {
	if os.Getenv("DEBUG") == "" {
		return
	}
	fmt.Printf(format, args...)
}
