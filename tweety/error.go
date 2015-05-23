package main

import (
	"os"
	"fmt"
)

func ErrorHandling(err error, msg string, exitCode int)(isError bool) {

	if err != nil {
		fmt.Println(msg, err)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		isError = true
	} else {
		isError = false
	}
	return
}
