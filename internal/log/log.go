package log

import "fmt"

func Info(msg string) {
	fmt.Printf("🦎 %s", msg)
}

func Error(errMsg string) {
	fmt.Printf("👾 %s", errMsg)
}
