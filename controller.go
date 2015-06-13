package main

import (
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
	"net/url"
	"github.com/kurrik/twittergo"
	"strconv"
	"bytes"
	"errors"
	"os"
	"mime/multipart"
	"io"
	"net/http"
	"path/filepath"
)

const(
	FB_ACCESS_TOKEN = "FB_ACCESS_TOKEN"
	FB_PAGE_ID = "FB_PAGE_ID"

)
var (
	body io.ReadWriter
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

	fmt.Printf("[Controller.GetFines] Resp Code: %v\n", resp.StatusCode)
	ErrorHandling(err, "Controller.GetFines] Error while requesting data: ", 1)

	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	ErrorHandling(err, "[Controller.GetFines] Error while parsing body: ", 1)

	//data []Fine
	err = json.Unmarshal(jsonDataFromHttp, &data)
	ErrorHandling(err, "[Controller.GetFines] Error while unmarshalling fines: ", 0)

	fmt.Printf("[Controller.GetFines] Found %v Fines....\n", len(data))
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
	if !ErrorHandling(err1, "[Controller.MarkAsTwitted] Problem while marking tweet "+tId+" published", 0) {

		r, err2 := ioutil.ReadAll(resp1.Body)
		if !ErrorHandling(err2, "[Controller.MarkAsTwitted] Problem while reading response: ", 0) {

			respCode := resp1.StatusCode
			fmt.Printf("[Controller.MarkAsTwitted] Resp Code: %v\n", respCode)
			if respCode != 200 {
				fmt.Println("Error: " + fmt.Sprintf("%s", r))
				ErrorHandling(errors.New("[Controller.MarkAsTwitted] Error while trying to mark tweet as red"), "Error: ", 10)
			}
			fmt.Println("[Controller.MarkAsTwitted] Tweet " + tId + " marked as published.")
		}
	}


}

// Facebook routines
func GetFBPageToken() (string, error) {

	fbUrl := "https://graph.facebook.com"
	fbResourceMe := fbUrl + "/" + os.Getenv(FB_PAGE_ID) + "?access_token=" + os.Getenv(FB_ACCESS_TOKEN) + "&fields=id,name,access_token"
	var me Me

	reqFB, _ := http.NewRequest("GET", fbResourceMe, nil)
	client := &http.Client{}
	respFB, errFB := client.Do(reqFB)
	ErrorHandling(errFB, "Error while requesting page access token: ", 0)

	jsonDataFromHttp, errFB := ioutil.ReadAll(respFB.Body)
	//fmt.Println("[controller.GetFBPageToken] JSON accounts details: " + fmt.Sprintf("%s", jsonDataFromHttp))
	ErrorHandling(errFB, "Error while parsing body: ", 0)
	errFB = json.Unmarshal(jsonDataFromHttp, &me)

	respCode := respFB.StatusCode
	fmt.Printf("[controller.GetFBPageToken] Resp Code: %v\n", respCode)
	if respCode != 200 {
		e := errors.New("[controller.GetFBPageToken] Error while retreiving facebook access token")
		ErrorHandling(e, "Error: ", 0)
		return "", e
	}
	pageAccessToken := me.Access_token
	fmt.Println("[controller.GetFBPageToken] got access token")

	return pageAccessToken, nil

}

func FBUploadPhoto(pageAccessToken string, image []byte) (string, error) {

	fmt.Println("[controller.FBUploadPhoto] Called")
	// creating temp file
	fileErr := ioutil.WriteFile("tempfile", image, 0777)
	ErrorHandling(fileErr, "[controller.FBUploadPhoto] Problem while writing temp file", 0)

	file, fileTempErr := os.Open("tempfile")
	ErrorHandling(fileTempErr, "[controller.FBUploadPhoto] Problem while readind temp file", 0)

	defer file.Close();

	fbUrl := "https://graph.facebook.com"
	fbResourceUpload := "/" + os.Getenv(FB_PAGE_ID) + "/photos"
	var photo Photo

	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("source", filepath.Base("tempfile"))
	ErrorHandling(err, "[controller.FBUploadPhoto] Error while creating part file", 0)

	_, err = io.Copy(part, file)
	if ErrorHandling(err, "[controller.FBUploadPhoto] Error while copying file in to the part", 0) {
		return "", err
	}

	_ = writer.WriteField("access_token", pageAccessToken)
	_ = writer.WriteField("no_story", "true")

	err = writer.Close()
	if ErrorHandling(err, "[controller.FBUploadPhoto] Error while closing writer", 0) {
		return "", err
	}

	reqFB, errFB := http.NewRequest("POST", (fbUrl + fbResourceUpload), body)
	reqFB.Header.Set("Content-Type", writer.FormDataContentType())

	clientFB := &http.Client{}
	respFB, errFB1 := clientFB.Do(reqFB)
	if ErrorHandling(errFB1, "[controller.FBUploadPhoto] Problem while posting feed on Facebook", 0) {
		return "", errFB1
	}

	jsonDataFromHttp, errParse := ioutil.ReadAll(respFB.Body)
	//fmt.Println("[controller.FBUploadPhoto] JSON Photo Upload Details: " + fmt.Sprintf("%s", jsonDataFromHttp))
	if ErrorHandling(errParse, "[controller.FBUploadPhoto] Error while parsing Facebook upload body: ", 0) {
		return "", errParse
	}

	errFB = json.Unmarshal(jsonDataFromHttp, &photo)
	if ErrorHandling(errFB, "[controller.FBUploadPhoto] Error while converting Facebook upload body to json: ", 0) {
		return "", errFB
	}

	respCode := respFB.StatusCode
	fmt.Printf("[controller.FBUploadPhoto] Resp Code: %v\n", respCode)
	if respCode != 200 {
		errFB = errors.New("[controller.FBUploadPhoto]  Error while trying to retrieve photoId")
		ErrorHandling(errFB, "", 0)
		return "", errFB
	}

	photoId := photo.Id
	fmt.Printf("[controller.FBUploadPhoto] Uploaded photo on Facebook id: %v\n", photoId)

	return photoId, nil
}

func GetFBPhotoDetails(photoId string, pageAccessToken string) (string, error) {

	var photoDetails PhotoDetails
	fbUrl := "https://graph.facebook.com"
	fbResourcePhotoDetails := fbUrl + "/" + photoId + "?access_token=" + pageAccessToken + "&fields=link"
	reqFB, errFB := http.NewRequest("GET", fbResourcePhotoDetails, nil)
	client := &http.Client{}
	respFB, errFB := client.Do(reqFB)
	if ErrorHandling(errFB, "[controller.GetFBPhotoDetails] Error while requesting Facebook photo details: ", 0) {
		return "", errFB
	}

	jsonDataFromHttp, errRead := ioutil.ReadAll(respFB.Body)
	if ErrorHandling(errRead, "[controller.GetFBPhotoDetails] Error while parsing Facebook photo details body: ", 0) {
		return "", errRead
	}

	//fmt.Println("[controller.GetFBPhotoDetails] JSON Photo Details: " + fmt.Sprintf("%s", jsonDataFromHttp))

	errFB = json.Unmarshal(jsonDataFromHttp, &photoDetails)
	if ErrorHandling(errFB, "[controller.GetFBPhotoDetails] Error while unmarshaling Facebook photo details: ", 0) {
		return "", errFB
	}

	photoLink := photoDetails.Link
	fmt.Printf("[controller.GetFBPhotoDetails] Facebook photo link: %v\n", photoLink)

	return photoLink, nil
}

func FBPostFeed (msgCategory string, msgText string, msgAddress string, pageAccessToken string, photoLink string) {

	fbUrl := "https://graph.facebook.com"
	var feed Feed
	fbMessage := Category[msgCategory]
	if msgText != "" {
		fbMessage = fbMessage+" - " + msgText
	}
	if msgAddress != "" {
		fbMessage = fbMessage+" In " + msgAddress
	}

	parameters := url.Values{}
	parameters.Add("access_token", pageAccessToken)
	parameters.Add("message", fbMessage)
	parameters.Add("link", photoLink)

	reqFB, _ := http.NewRequest("POST", fbUrl+"/"+os.Getenv(FB_PAGE_ID)+"/feed", bytes.NewBufferString(parameters.Encode()))

	client := &http.Client{}
	respFB, errFB := client.Do(reqFB)
	if !ErrorHandling(errFB, "Problem while posting feed", 0) {

		jsonDataFromHttp, errFB1 := ioutil.ReadAll(respFB.Body)
		if !ErrorHandling(errFB1, "Problem while reading response: ", 0) {

			respCode := respFB.StatusCode
			fmt.Printf("Resp Code: %v\n", respCode)
			if respCode != 200 {
				ErrorHandling(errors.New("Error while trying to post feed with photo link"), "Error: ", 0)
			}
			errFB = json.Unmarshal(jsonDataFromHttp, &feed)
			fmt.Println("Feed " + feed.Id + " published on facebook")
		}
	}

}
