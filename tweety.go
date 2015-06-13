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
	"os"
	"strings"
	"time"
	"encoding/json"
	"strconv"
	"errors"
	"os/signal"
	"syscall"
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
	Category map[string]string
)

func main() {

	// SIGINT Managment
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("SIGTERM Detected")
	}()

	if len(os.Args) < 2 {
		ErrorHandling(errors.New("Invalid number of arguments"), "Error: ", 1)
	}

	loadCategory()

	if len(os.Args) == 3 {
		timerRange = os.Args[2]
	} else {
		timerRange = "300s"
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
		port := os.Getenv("PORT")
		fmt.Println("Listening on port " + port)
		StartTriggering(nil, nil)
		http.ListenAndServe(":"+port, nil)
	} else {
		ErrorHandling(errors.New("Invalid argument, valid optins are 'batch' or 'server'"), "Error: ", 1)
	}
}

func loadCategory() {

	Category = map[string]string {
		"PRC": "Sosta in divieto",
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

func GetBody(message string, media []byte, address string, createdAt string, placeId string, city string, cat string, county string) (body io.ReadWriter, header string, err error) {
	var (
		mp *multipart.Writer
		//media  []byte
		writer io.Writer
		msgStart string
		msgEnd string
	)

	body = bytes.NewBufferString("")
	mp = multipart.NewWriter(body)

	t, _ := time.Parse(time.RFC3339, createdAt)
	t1 := t.Format(time.RFC822)
	t1 = t1[0:len(t1)-3]

	msgStart = Category[cat]

	msgEnd = " In "+address

	msgHash := ""

	//	// category + rome
	//	if cat == "PRC" || cat == "DST" {
	//		if city == "rome" || city == "roma" {
	//			msgHash += " @plromacapitale @fajelamulta @romamigliore @incivileabordo"
	//		}
	//	}
	//	if cat == "RTF" {
	//		if city == "rome" || city == "roma" {
	//			msgHash += " #AMARoma @RomaPulita"
	//		}
	//	}
	//	if cat == "ABS" || cat == "ILL" || cat == "MNT" || cat == "VND" || cat == "SGN" || cat == "DST" || cat == "RFT" {
	//		if city == "rome" || city == "roma" {
	//			msgHash += " @Retake_Roma @romafaschifo"
	//		}
	//	}
	//	if city == "rome" || city == "roma" {
	//		msgHash += " @Antincivili"
	//	}

	// category + milan
	if cat == "RTF" {
		if city == "milano" || city == "milan" {
			msgHash += " @milanopulita"
		}
	}

	// benevento
	if county == "bn" {
		msgHash += " @sannioreport"
	}


	if len(message) > 0 {
		message = " "+message
		msgStart = ""
	}

	if len(msgStart + message + msgEnd + msgHash) > 120 {
		fmt.Println("[GetBody] message is over limit")
		availableLen := (120 - len(msgStart + msgEnd + msgHash))

		if availableLen > 0 {
			if availableLen < len(message) {
				fmt.Println("[GetBody] truncating message")
				if availableLen > 3 {
					message = message[0:(availableLen-3)]+"..."
				} else {
					message = message[0:(availableLen)]
				}
			} else {
				fmt.Println("[GetBody] blanking message")
				message = ""
			}
		} else {
			fmt.Println("[GetBody] It's not possible to gracefully truncate message")
		}
	}

	tText := fmt.Sprintf("%v%v%v%v", msgStart, message, msgEnd, msgHash)
	if len(tText) > 120 {
		fmt.Println("[GetBody] still to long message, truncating to 140")
		tText = tText[0:120]
	}

	fmt.Println("Text: " + tText)

	mp.WriteField("status", tText)

	//	mp.WriteField("place_id", placeId)

	writer, err = mp.CreateFormField("media[]")
	ErrorHandling(err, "Error while creating writer: ", 1)

	writer.Write(media)
	header = fmt.Sprintf("multipart/form-data;boundary=%v", mp.Boundary())
	mp.Close()
	return
}

type location struct {
	Type        string
	Coordinates []float64
}

type Fine struct {
	Id        string `json:"_id"`
	Address   string
	City      string
	County    string `json:"county"`
	Approved  bool
	Category  string
	CreatedAt string
	ImageData string
	Loc       location
	Text      string
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

	if len(data.Result.Places) > 0 {
		return data.Result.Places[0].Id
	}
	return ""

}


func publish(w http.ResponseWriter, r *http.Request) {

	var (
		req    *http.Request
		resp   *twittergo.APIResponse
		tweet  *twittergo.Tweet
		//mp *multipart.Writer
		photoId string
		photoLink string
		errPhoto error
		errLink error
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

	// ***** Get Page token ***************************
	pageAccessToken, _ := GetFBPageToken()

	for _, element := range toTwet {

		placeId := getPlaceId(element.Loc.Coordinates[1], element.Loc.Coordinates[0])

		latitude := strconv.FormatFloat(element.Loc.Coordinates[1], 'f', 7, 32)
		longitude := strconv.FormatFloat(element.Loc.Coordinates[0], 'f', 7, 32)

		endpoint := baseendpoint + "?lat=" + latitude + "&long=" + longitude + "&display_coordinates=true"

		image = Decode(element.ImageData[22:])
		body, header, err := GetBody(element.Text, image, element.Address, element.CreatedAt, placeId, strings.ToLower(element.City), element.Category, strings.ToLower(element.County))

		ErrorHandling(err, "Problem loading body: ", 1)

		req, err = http.NewRequest("POST", endpoint, body)
		if !ErrorHandling(err, "[Tweet Post] Could not parse request: ", 0) {

			req.Header.Set("Content-Type", header)

			resp, err = client.SendRequest(req)
			ErrorHandling(err, "[Tweet Post] Could not send request: ", 0)

			tweet = &twittergo.Tweet{}
			err = resp.Parse(tweet)
			if !ErrorHandling(err, "[Tweet Post] Problem parsing response: ", 0) {
				// Mark fine posted on twitter
				MarkAsTwitted(tweet, element)
			}
		}

		// Post fine on Facebook
		if pageAccessToken != "" {
			photoId, errPhoto = FBUploadPhoto(pageAccessToken, image) // **** Photo Upload
			if errPhoto == nil {
				photoLink, errLink = GetFBPhotoDetails(photoId, pageAccessToken) // **** Photo Details
				if errLink == nil {
					FBPostFeed(element.Category, element.Text, element.Address, pageAccessToken, photoLink) // **** Feed Post Begin
				}
			}
		}

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
			io.WriteString(w, "Endpoint ..........."+endpoint+"\n")
			io.WriteString(w, "Place ID............"+placeId+"\n")
			io.WriteString(w, "ID ................."+fmt.Sprintf("%v", tweet.Id())+"\n")
			io.WriteString(w, "Tweet .............."+tweet.Text()+"\n")
			io.WriteString(w, "User ..............."+tweet.User().Name()+"\n")
			io.WriteString(w, "latitude ..........."+latitude+"\n")
			io.WriteString(w, "longitude .........."+longitude+"\n")
			io.WriteString(w, "------------------------------------------------------------------------------------\n\n")
		}
	}
}

type Photo struct {
	Id string `json:"id"`
}

type Feed struct {
	Id string `json:"id"`
}

type PhotoDetails struct {
	Link string `json:"link"`
}

type Me struct {
	Id string `json:"id"`
	Name string `json:"name"`
	Access_token string `json:"access_token"`
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
