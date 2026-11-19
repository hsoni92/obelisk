package controller

import "strings"

func execAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "forbidden") && strings.Contains(msg, "pods/exec")
}