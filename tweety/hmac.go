package main

import(
	"crypto/md5"
	"crypto/hmac"
	"fmt"
)

func EncHmacMD5 (token string, key string) string{

	t := []byte(token)
	k := []byte(key)

	h := hmac.New(md5.New, k)
	h.Write(t)
	sum := fmt.Sprintf("%x", h.Sum(nil))
	//fmt.Printf("SUM: %v\n", sum)

	return sum
}
