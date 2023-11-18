package main

import "strings"

func getPhoneNumber(message string) string {
	stringArray := strings.Split(message, " ")

	return stringArray[4]
}

func getAwardedScore(message string) string {
	stringArray := strings.Split(message, " ")

	return stringArray[2]
}
