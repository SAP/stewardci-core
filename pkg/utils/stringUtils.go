package k8s

import (
	"regexp"
	"strings"
)

//Trim removes blanks
func Trim(input string) (output string) {
	return strings.Trim(input, " \t\n\r")
}

//ShortenMessage Removes line breaks and shortens the message
func ShortenMessage(message string, length int) (shortenedMessage string) {
	if length < 3 {
		length = 3
	}
	shortenedMessage = Trim(message)
	re := regexp.MustCompile(`\s+`) //any whitespaces
	shortenedMessage = re.ReplaceAllString(shortenedMessage, " ")
	if len(shortenedMessage) > length {
		shortenedMessage = shortenedMessage[:length-3] + "..."
	}
	return
}
