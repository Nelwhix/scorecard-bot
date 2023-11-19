package utils

import (
	"strconv"
	"strings"
)

func GetPhoneNumber(message string) string {
	stringArray := strings.Split(message, " ")

	return stringArray[4]
}

func GetAwardedScore(message string) int {
	stringArray := strings.Split(message, " ")

	score, _ := strconv.Atoi(stringArray[2])

	return score
}
