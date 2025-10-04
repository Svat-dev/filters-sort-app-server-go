package cerror

import "log"

func errorLogger(message string, status int) {
	log.Printf("[Server Error] %d %s", status, message)
}
