package main

import (
	"bytes"
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"encoding/json"
    "encoding/base64"
	"strconv"
	"crypto/md5"
	"crypto/hmac"
	"errors"
)

const CONSUMER_KEY = "CONSUMER_KEY"
const CONSUMER_SECRET = "CONSUMER_SECRET"
const ACCESS_TOKEN = "ACCESS_TOKEN"
const ACCESS_TOKEN_SECRET = "ACCESS_TOKEN_SECRET"

var (
	client *twittergo.Client
	auth string = "false"

)


func LoadCredentials() (client *twittergo.Client, err error) {

	consumerKey := os.Getenv(CONSUMER_KEY)
	consumerSecret := os.Getenv(CONSUMER_SECRET)
	accessToken := os.Getenv(ACCESS_TOKEN)
	accessTokenSecret := os.Getenv(ACCESS_TOKEN_SECRET)

	if consumerKey != "" && consumerSecret != "" && accessToken != "" && accessTokenSecret != "" {
		fmt.Printf("Credentials loaded using environment variable\n")
	} else {
		fmt.Printf("No environment variable found\n")
		fmt.Printf("Tryng with CREDENTIAL file...\n")

		credentials, err := ioutil.ReadFile("CREDENTIALS")
		errorHandling(err, "Error while loading CREDENTIALS file: ", 1)

		fmt.Printf("Credentials loaded using CREDENTIAL file\n")

		lines := strings.Split(string(credentials), "\n");

		consumerKey = lines[0]
		consumerSecret = lines[1]
		accessToken = lines[2]
		accessTokenSecret = lines[3]
	}

	config := &oauth1a.ClientConfig {
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	user := oauth1a.NewAuthorizedConfig(accessToken, accessTokenSecret);
	client = twittergo.NewClient(config, user);

	return
}

func GetBody(message string, media []byte, address string, createdAt string, placeId string) (body io.ReadWriter, header string, err error) {
	var (
		mp *multipart.Writer
		//media  []byte
		writer io.Writer
		msg string
	)

	body = bytes.NewBufferString("")
	mp = multipart.NewWriter(body)

	t, _ := time.Parse(time.RFC3339, createdAt)
    t1 := t.Format(time.RFC822)
	t1 = t1[0:len(t1)-3]

	if message != "" {
	    msg = fmt.Sprintf("%v", t1) + " '" +  message + "'" + " In " + address
	} else {
		msg = fmt.Sprintf("%v", t1) + " In " + address
	}

	mp.WriteField("status", fmt.Sprintf(msg))
//	mp.WriteField("place_id", placeId)

	writer, err = mp.CreateFormField("media[]")
	errorHandling(err, "Error while creating writer: ", 1)

	writer.Write(media)
	header = fmt.Sprintf("multipart/form-data;boundary=%v", mp.Boundary())
	mp.Close()
	return
}

type location struct {
      Type string
      Coordinates []float64
}

type Fine struct {
    Id string `json:"_id"`
    Address string
    Approved int
    Category string
    CreatedAt string
    ImageData string
    Loc location
    Text string
}

func getFines() (data []Fine) {

    timultoUrl := "http://beta.timulto.org/api/fines/twitter"
    resp, err := http.Get(timultoUrl)
	errorHandling(err, "Error while requesting data: ", 1)

    jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	errorHandling(err, "Error while parsing body: ", 1)

    //data []Fine
    err = json.Unmarshal(jsonDataFromHttp, &data)
	errorHandling(err, "Error while unmarshalling fines: ", 1)

    return
}

func decode(str string) (data []byte){

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	return
}

type PlaceResult struct {
	Places []Location `json:"places"`
}

type Location struct {
	Id string `json:"id"`
}

type Place struct {
	Result PlaceResult `json:"result"`
}

func getPlaceId(latitude float64, longitude float64) (p string) {

	//var client *twittergo.Client
	var err error
	var data Place

	fmt.Printf("Looking for coordinates %v and %v\n\n", latitude, longitude)

	if client == nil {
		client, err = LoadCredentials();
	}

	errorHandling(err, "Error while loading credential: ", 1)

//	twitterUrl := "https://api.twitter.com/1.1/geo/search.json?lat=" + strconv.FormatFloat(latitude, 'f', 1, 32) + "&long=" + strconv.FormatFloat(longitude, 'f', 1, 32)
	twitterUrl := "https://api.twitter.com/1.1/geo/reverse_geocode.json?lat=" + strconv.FormatFloat(latitude, 'f', 7, 32) + "&long=" + strconv.FormatFloat(longitude, 'f', 7, 32)

	req, err1 := http.NewRequest("GET", twitterUrl, nil)
	errorHandling(err1, "Error while creating the request object: ", 1)

	resp, err2 := client.SendRequest(req)
	errorHandling(err2, "Error while sending the request: ", 1)

	jsonDataFromHttp, err3 := ioutil.ReadAll(resp.Body)
	errorHandling(err3, "Error while reading response body: ", 1)

//	fmt.Println(string(jsonDataFromHttp[:]))

	err4 := json.Unmarshal(jsonDataFromHttp, &data)
	errorHandling(err4, "Error while parsing json response: ", 1)

	if len(data.Result.Places) >0 {
		return data.Result.Places[0].Id
	}
	return ""

}

func main() {

	if len(os.Args) < 2 {
		errorHandling(errors.New("Invalid number of arguments"), "Error: ", 1)
	}

//	if os.Args[2] == "true" {
//		auth = "tue"
//	}

	if os.Args[1] == "batch" {
		fmt.Println("Running in batch mode")
		publish(nil, nil)
	} else {
		fmt.Println("Running in server mode")
		http.HandleFunc("/publish", publish)
		http.ListenAndServe("localhost:8000", nil)
	}
}

func publish(w http.ResponseWriter, r *http.Request) {

    var (
		req    *http.Request
		resp   *twittergo.APIResponse
		tweet  *twittergo.Tweet
	);

	if client == nil {
		client, _ = LoadCredentials();
	}

    var toTwet = getFines()
    var image []byte
	baseendpoint := "/1.1/statuses/update_with_media.json"
//	baseendpoint := "/1.1/statuses/update.json"

	if len(toTwet) == 0 {
	    fmt.Println("No mew tweet to post")
		if w != nil {
			io.WriteString(w, "No mew tweet to post\n")
		}
	}

    for _, element := range toTwet {


		placeId := getPlaceId(element.Loc.Coordinates[1], element.Loc.Coordinates[0])

		latitude := strconv.FormatFloat(element.Loc.Coordinates[1], 'f', 7, 32)
		longitude := strconv.FormatFloat(element.Loc.Coordinates[0], 'f', 7, 32)

//		endpoint := baseendpoint + "?place_id=" + placeId + "&display_coordinates=true"
		endpoint := baseendpoint + "?lat=" + latitude + "&long=" + longitude + "&display_coordinates=true"

        image = decode(element.ImageData[22:])
        body, header, err := GetBody(element.Text, image, element.Address, element.CreatedAt, placeId)

		errorHandling(err, "Problem loading body: ", 1)

        req, err = http.NewRequest("POST", endpoint, body)
		errorHandling(err, "Could not parse request: ", 1)

        req.Header.Set("Content-Type", header)

        resp, err = client.SendRequest(req)
		errorHandling(err, "Could not send request: ", 1)

        tweet = &twittergo.Tweet{}
        err = resp.Parse(tweet)
		errorHandling(err, "Problem parsing response: ", 1)

		// Mark fine posted on twitter
        url1 := "http://beta.timulto.org/api/fine/" + element.Id +  "/twitter"
		tId := strconv.FormatUint(tweet.Id(), 10)

        parameters := url.Values{}
		parameters.Add("postId", tId)
		resp1, err1 :=http.PostForm(url1, parameters)

        if ! errorHandling(err1, "Problem while marking tweet " + tId + " published", 1)  {
			fmt.Println("Tweet " + tId + " marked as published.")
		}

		_, err2 := ioutil.ReadAll(resp1.Body)
		errorHandling(err2, "Problem while reading response: ", 1)
		fmt.Println("------------------------------------------------------------------------------------")
		fmt.Printf("Endpoint ...........%v\n", endpoint)
		fmt.Printf("Place ID ...........%v\n", placeId)
		fmt.Printf("ID .................%v\n", tweet.Id())
		fmt.Printf("Tweet ..............%v\n", tweet.Text())
		fmt.Printf("User ...............%v\n", tweet.User().Name())
		fmt.Printf("latitude ...........%v\n", latitude)
		fmt.Printf("longitude ..........%v\n", longitude)
		fmt.Println("------------------------------------------------------------------------------------\n\n")

		if w != nil && r != nil {
			io.WriteString(w, "------------------------------------------------------------------------------------\n")
			io.WriteString(w, "Endpoint ..........." + endpoint + "\n")
			io.WriteString(w, "Place ID............" + placeId + "\n")
			io.WriteString(w, "ID ................." + fmt.Sprintf("%v", tweet.Id()) + "\n")
			io.WriteString(w, "Tweet .............." + tweet.Text() + "\n")
			io.WriteString(w, "User ..............." + tweet.User().Name() + "\n")
			io.WriteString(w, "latitude ..........." + latitude + "\n")
			io.WriteString(w, "longitude .........." + longitude + "\n")
			io.WriteString(w, "------------------------------------------------------------------------------------\n\n")
		}
    }
}

func errorHandling(err error, msg string, exitCode int)(isError bool) {

	if err != nil {
		fmt.Println(msg, err)
		if exitCode == -1 {
			os.Exit(exitCode)
		}
		isError = true
	} else {
		isError = false
	}
	return
}

func encHmacMD5 (token string, key string) string{

	t := []byte(token)
	k := []byte(key)

	h := hmac.New(md5.New, k)
	h.Write(t)
	sum := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Printf("SUM: %v\n", sum)

	return sum
}
