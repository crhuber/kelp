package logging

import "log"

var logVerbose = false

func SetLogVerbose(verbose bool) {
	logVerbose = verbose
}

func LogDebug(message string, args ...any) {
	if logVerbose {
		log.Printf(message, args...)
	}
}

func LogInfo(message string, args ...any) {
	log.Printf(message, args...)
}

func init() {
	// set log output to the simplest as possible, without anything but the message
	log.SetFlags(0)
}
