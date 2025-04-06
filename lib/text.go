package lib

import (
	"regexp"
	"strings"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(str string) string {
	return ansiRegexp.ReplaceAllString(str, "")
}

func PadRightAnsiAware(colored string, width int) string {
	raw := stripAnsi(colored)
	padding := width - len([]rune(raw))
	if padding < 0 {
		padding = 0
	}
	return colored + strings.Repeat(" ", padding)
}
