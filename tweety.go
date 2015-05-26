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
//	"crypto/md5"
//	"crypto/hmac"
	"errors"
	//"net"
)

const CONSUMER_KEY = "CONSUMER_KEY"
const CONSUMER_SECRET = "CONSUMER_SECRET"
const ACCESS_TOKEN = "ACCESS_TOKEN"
const ACCESS_TOKEN_SECRET = "ACCESS_TOKEN_SECRET"
const HMAC_KEY = "HMAC_KEY"


const APP_NAME = "twitter"

var (
	client *twittergo.Client
	auth string = "false"
	hmacKey string
	timerRange string
	//doneChan chan(bool)
	category map[string]string
)

func loadCategory() {

	category = map[string]string {
		"PRC": "Parcheggio incivile",
		"RFT": "Rifiuti o cassonetti sporchi",
		"ACC": "Accessibilità scarsa o mancante",
		"ABS": "Abusivismo",
		"DST": "Disturbo della quiete pubblica",
		"ILL": "Illuminazione",
		"MNT": "Manto stradale",
		"VND": "Atti vandalici",
		"SGN": "Segnaletica mancante",
		"MLT": "Maltrattamento animali",
	}
}

func LoadCredentials() (client *twittergo.Client, err error) {

	consumerKey := os.Getenv(CONSUMER_KEY)
	consumerSecret := os.Getenv(CONSUMER_SECRET)
	accessToken := os.Getenv(ACCESS_TOKEN)
	accessTokenSecret := os.Getenv(ACCESS_TOKEN_SECRET)
	hmacKey = os.Getenv(HMAC_KEY)

	if consumerKey != "" && consumerSecret != "" && accessToken != "" && accessTokenSecret != "" {
		fmt.Printf("Credentials loaded using environment variable\n")
	} else {
		fmt.Printf("No environment variable found\n")
		fmt.Printf("Tryng with CREDENTIAL file...\n")

		credentials, err := ioutil.ReadFile("CREDENTIALS")
		ErrorHandling(err, "Error while loading CREDENTIALS file: ", 1)

		fmt.Printf("Credentials loaded using CREDENTIAL file\n")

		lines := strings.Split(string(credentials), "\n");

		consumerKey = lines[0]
		consumerSecret = lines[1]
		accessToken = lines[2]
		accessTokenSecret = lines[3]
		hmacKey = lines[4]
	}

	config := &oauth1a.ClientConfig {
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}

	user := oauth1a.NewAuthorizedConfig(accessToken, accessTokenSecret);
	client = twittergo.NewClient(config, user);

	return
}

func GetBody(message string, media []byte, address string, createdAt string, placeId string, city string, cat string) (body io.ReadWriter, header string, err error) {
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

	msg = fmt.Sprintf("%v", t1) + " - " + category[cat] + " -"
	if message != "" {
	    msg += " '" +  message + "'"
	}
	msg += " In " + address

//	if cat == "PRC" || cat == "DST" {
//		if city == "rome" || city == "roma" {
//			msg += " - @plromacapitale @fajelamulta @romamigliore"
//		}
//	}
//	if cat == "RTF" {
//		if city == "rome" || city == "roma" {
//			msg += " - #AMARoma"
//		}
//	}
//	if cat == "ABS" || cat == "ILL" || cat == "MNT" || cat == "VND" || cat == "SGN" || cat == "DST" || cat == "RFT" {
//		if city == "rome" || city == "roma" {
//			msg += " - @Retake_Roma @romafaschifo"
//		}
//	}


	mp.WriteField("status", fmt.Sprintf(msg))
//	mp.WriteField("place_id", placeId)

	writer, err = mp.CreateFormField("media[]")
	ErrorHandling(err, "Error while creating writer: ", 1)

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
	City string
    Approved bool
    Category string
    CreatedAt string
    ImageData string
    Loc location
    Text string
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

	var err error
	var data Place

	fmt.Printf("Looking for coordinates %v and %v\n\n", latitude, longitude)

	if client == nil {
		client, err = LoadCredentials();
	}

	ErrorHandling(err, "Error while loading credential: ", 1)

	twitterUrl := "https://api.twitter.com/1.1/geo/reverse_geocode.json?lat=" + strconv.FormatFloat(latitude, 'f', 7, 32) + "&long=" + strconv.FormatFloat(longitude, 'f', 7, 32)

	req, err1 := http.NewRequest("GET", twitterUrl, nil)
	ErrorHandling(err1, "Error while creating the request object: ", 1)

	resp, err2 := client.SendRequest(req)
	ErrorHandling(err2, "Error while sending the request: ", 1)

	jsonDataFromHttp, err3 := ioutil.ReadAll(resp.Body)
	ErrorHandling(err3, "Error while reading response body: ", 1)

	err4 := json.Unmarshal(jsonDataFromHttp, &data)
	ErrorHandling(err4, "Error while parsing json response: ", 1)

	if len(data.Result.Places) >0 {
		return data.Result.Places[0].Id
	}
	return ""

}

func main() {

	if len(os.Args) < 2 {
		ErrorHandling(errors.New("Invalid number of arguments"), "Error: ", 1)
	}

	loadCategory()

	if len(os.Args) == 3 {
		timerRange = os.Args[2]
	}

	if os.Args[1] == "batch" {
		fmt.Println("Running in batch mode")
		if timerRange == "" {
			publish(nil, nil)
		} else {
			StartTriggeringBatch(timerRange)
		}

	} else if os.Args[1] == "server" {
		fmt.Println("Running in server mode")
		http.HandleFunc("/publish", publish)
		http.HandleFunc("/startTicker", StartTriggering)
		http.HandleFunc("/stopTicker", StopTicker)
		http.HandleFunc("/", GetInfo)
		fmt.Println("Listening on port 8000")
		http.ListenAndServe("localhost:8000", nil)
	} else {
		ErrorHandling(errors.New("Invalid argument, valid optins are 'batch' or 'server'"), "Error: ", 1)
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

    var toTwet = GetFines()
    var image []byte
	baseendpoint := "/1.1/statuses/update_with_media.json"

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

		endpoint := baseendpoint + "?lat=" + latitude + "&long=" + longitude + "&display_coordinates=true"

        image = decode(element.ImageData[22:])
        body, header, err := GetBody(element.Text, image, element.Address, element.CreatedAt, placeId, strings.ToLower(element.City), element.Category)

		ErrorHandling(err, "Problem loading body: ", 1)

        req, err = http.NewRequest("POST", endpoint, body)
		ErrorHandling(err, "Could not parse request: ", 1)

        req.Header.Set("Content-Type", header)

        resp, err = client.SendRequest(req)
		ErrorHandling(err, "Could not send request: ", 1)

		tweet = &twittergo.Tweet{}
		err = resp.Parse(tweet)
		ErrorHandling(err, "Problem parsing response: ", 1)

		// Mark fine posted on twitter
		url1 := "http://beta.timulto.org"
		resource := "/api/fine/" + element.Id +  "/twitter"

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

		ErrorHandling(err1, "Problem while marking tweet " + tId + " published", 1)

		r, err2 := ioutil.ReadAll(resp1.Body)
		ErrorHandling(err2, "Problem while reading response: ", 1)

		respCode := resp1.StatusCode
		fmt.Printf("Resp Code: %v\n", respCode)
		if respCode != 200 {
			fmt.Println("Error: " + fmt.Sprintf("%s", r))
			ErrorHandling(errors.New("Error while trying to mark tweet as red"), "Error: ", 1)
		}
//		fmt.Println("Response Body: \n" + fmt.Sprintf("%s", r))

		fmt.Println("Tweet " + tId + " marked as published.")

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


//func getIpAddress () string {
//
//	var toRet string
//
//	host, _ := os.Hostname()
//	addrs, _ := net.LookupIP(host)
//
//	for _, addr := range addrs {
//		if ipv4 := addr.To4(); ipv4 != nil {
//			if toRet == "" {
//				toRet = string(ipv4[:])
//			}
//			//fmt.Println("IPv4: ", ipv4)
//		}
//	}
//	fmt.Println("IPv4: " + toRet)
//	return toRet
//}