package main

import (

	"net/http"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
)



func GetFines() (data []Fine) {

	t := time.Now().Local()
	tstamp := t.Format("20060102150405")
	toEncode := tstamp + "#" + APP_NAME + "#" + "twitter"
	fmt.Printf("To encode: %v\n", toEncode)
	token := EncHmacMD5(toEncode, hmacKey)
	fmt.Printf("Encoded: %v\n", token)

	timultoUrl := "http://beta.timulto.org/api/fines/twitter"
	//timultoUrl := "http://beta.timulto.org/api/token/twitter"
	req, _ := http.NewRequest("GET", timultoUrl, nil)
	req.Header.Add("timestamp", tstamp)
	req.Header.Add("app", APP_NAME)
	req.Header.Add("token", token)
	client := &http.Client{}
	resp, err := client.Do(req)

	fmt.Printf("Resp Code: %v\n", resp.StatusCode)
	//resp, err := http.Get(timultoUrl)
	ErrorHandling(err, "Error while requesting data: ", 1)

	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	ErrorHandling(err, "Error while parsing body: ", 1)

//	fmt.Println("JSON FINES: " + fmt.Sprintf("%s", jsonDataFromHttp))

	//data []Fine
	err = json.Unmarshal(jsonDataFromHttp, &data)
	ErrorHandling(err, "Error while unmarshalling fines: ", 1)

	fmt.Printf("Found %v Fines....\n", len(data))
	return
}
