package main

import (

	"net/http"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"net/url"
	"github.com/kurrik/twittergo"
	"strconv"
	"bytes"
	"errors"
)



func GetFines() (data []Fine) {

	t := time.Now().Local()
	tstamp := t.Format("20060102150405")
	toEncode := tstamp + "#" + APP_NAME + "#" + "twitter"
	token := EncHmacMD5(toEncode, hmacKey)

	timultoUrl := "http://beta.timulto.org/api/fines/twitter"
	req, _ := http.NewRequest("GET", timultoUrl, nil)
	req.Header.Add("timestamp", tstamp)
	req.Header.Add("app", APP_NAME)
	req.Header.Add("token", token)
	client := &http.Client{}
	resp, err := client.Do(req)

	fmt.Printf("Resp Code: %v\n", resp.StatusCode)
	ErrorHandling(err, "Error while requesting data: ", 1)

	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	ErrorHandling(err, "Error while parsing body: ", 1)

//	fmt.Println("JSON FINES: " + fmt.Sprintf("%s", jsonDataFromHttp))

	//data []Fine
	err = json.Unmarshal(jsonDataFromHttp, &data)
	ErrorHandling(err, "Error while unmarshalling fines: ", 0)

	fmt.Printf("Found %v Fines....\n", len(data))
	return
}

func MarkAsTwitted(tweet *twittergo.Tweet, element Fine) {

	url1 := "http://beta.timulto.org"
	resource := "/api/fine/" + element.Id + "/twitter"

	tId := strconv.FormatUint(tweet.Id(), 10)

	parameters := url.Values{}
	parameters.Add("postId", tId)

	u, _ := url.ParseRequestURI(url1)
	u.Path = resource
	urlStr := fmt.Sprintf("%v", u)

	t := time.Now().Local()
	tstamp := t.Format("20060102150405")
	toEncode := tstamp + "#" + APP_NAME + "#" + "twitter" + "#" + element.Id
	token := EncHmacMD5(toEncode, hmacKey)

	req, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(parameters.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(parameters.Encode())))
	req.Header.Add("timestamp", tstamp)
	req.Header.Add("app", APP_NAME)
	req.Header.Add("token", token)

	client := &http.Client{}
	resp1, err1 := client.Do(req)
	if !ErrorHandling(err1, "Problem while marking tweet "+tId+" published", 0) {

		r, err2 := ioutil.ReadAll(resp1.Body)
		if !ErrorHandling(err2, "Problem while reading response: ", 0) {

			respCode := resp1.StatusCode
			fmt.Printf("Resp Code: %v\n", respCode)
			if respCode != 200 {
				fmt.Println("Error: " + fmt.Sprintf("%s", r))
				ErrorHandling(errors.New("Error while trying to mark tweet as red"), "Error: ", 10)
			}
			fmt.Println("Tweet " + tId + " marked as published.")
		}
	}
}

