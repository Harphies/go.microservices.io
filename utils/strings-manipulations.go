package utils

import (
	"fmt"
	"strings"
)

func containsAction(actionList []string, action string) bool {
	for _, a := range actionList {
		if a == action {
			return true
		}
	}
	return false
}

func TransformingStrings() {
	data := "{\"total\":3,\"successful\":3,\"skipped\":0,\"failed\":0}"
	// Trim a string
	// // https://sipfront.com/blog/2023/04/golang-logging-directly-to-aws-opensearch/
	fmt.Println("trimmed string:", strings.Trim(string(data), "{}"))
}
