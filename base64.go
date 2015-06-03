package main

import "encoding/base64"
import "fmt"


func Decode(str string) (data []byte){

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	return
}
