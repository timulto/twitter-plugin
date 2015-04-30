package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
)

func decode() {

	str := "c29tZSBkYXRhIHdpdGggACBhbmQg77u/"
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("%q\n", data)
}

func main() {

	var (
		//err error
		media  []byte
	)

	media, err := ioutil.ReadFile("media.png")
	if err != nil {
		os.Exit(1);
	}
	//data := []byte("any + old & data")
	str := base64.StdEncoding.EncodeToString(media)
	b64 := []byte(str)
	err = ioutil.WriteFile("encoded.txt", b64, 0644)
	if err != nil {
		os.Exit(1);
	}
	fmt.Println(str)
}
