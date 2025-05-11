package util

import "log"

func Success(format string, a ...any) {
	log.Printf("[SUCCESS] "+format, a...)
}

func Fail(format string, a ...any) {
	log.Printf("[FAIL] "+format, a...)
}

func Warn(format string, a ...any) {
	log.Printf("[WARNING] "+format, a...)
}

func Info(format string, a ...any) {
	log.Printf("[INFO] "+format, a...)
}
