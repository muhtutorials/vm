package cpu

import (
	"fmt"
	"os"
	"regexp"
)

// debugPrintf outputs when "DEBUG=1"
func debugPrintf(format string, args ...any) {
	if os.Getenv("DEBUG") == "" {
		return
	}
	fmt.Printf(format, args...)
}

// splitCommand splits a string into tokens but keeps anything "quoted" together.
//
// So this input:
//
//	/bin/sh -c "ls /etc"
//
// Would give output of the form:
//
//	/bin/sh
//	-c
//	ls /etc
func splitCommand(input string) []string {
	r := regexp.MustCompile(`[^\s"']+|"[^"]*"|'[^']*'`)
	result := r.FindAllString(input, -1)

	// resulting pieces might be quoted, so we have to remove them, if present
	var strSlice []string
	for _, s := range result {
		strSlice = append(strSlice, trimStringEnds(s, '"'))
	}

	return result
}

// trimStringEnds removes balanced characters around a string
func trimStringEnds(str string, c byte) string {
	if len(str) >= 2 {
		if str[0] == c && str[len(str)-1] == c {
			return str[1 : len(str)-1]
		}
	}
	return str
}
