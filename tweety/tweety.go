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
)

func LoadCredentials() (client *twittergo.Client, err error) {

	credentials, err := ioutil.ReadFile("CREDENTIALS")
	errorHandling(err, "Error while loading CREDENTIALS file: ", 1)

	lines := strings.Split(string(credentials), "\n");
	config := &oauth1a.ClientConfig {
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}

	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3]);
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

	var client *twittergo.Client

	var data Place

	fmt.Printf("Looking for coordinates %v and %v\n", latitude, longitude)

	client, err := LoadCredentials();
	errorHandling(err, "Error while loading credential: ", 1)

//	twitterUrl := "https://api.twitter.com/1.1/geo/search.json?lat=" + strconv.FormatFloat(latitude, 'f', 1, 32) + "&long=" + strconv.FormatFloat(longitude, 'f', 1, 32)
	twitterUrl := "https://api.twitter.com/1.1/geo/reverse_geocode.json?lat=" + strconv.FormatFloat(latitude, 'f', 1, 32) + "&long=" + strconv.FormatFloat(longitude, 'f', 1, 32)

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

    var (
		err    error
		client *twittergo.Client
		req    *http.Request
		resp   *twittergo.APIResponse
		tweet  *twittergo.Tweet
	);

	client, err = LoadCredentials();
	errorHandling(err, "Could not parse CREDENTIALS file: ", 1)

    var toTwet = getFines()
    var image []byte
	baseendpoint := "/1.1/statuses/update_with_media.json"
//	baseendpoint := "/1.1/statuses/update.json"

	if len(toTwet) == 0 {
	    fmt.Println("No mew tweet to post")
	}

    for _, element := range toTwet {


		placeId := getPlaceId(element.Loc.Coordinates[0], element.Loc.Coordinates[1])

		latitude := strconv.FormatFloat(element.Loc.Coordinates[0], 'f', 1, 32)
		longitude := strconv.FormatFloat(element.Loc.Coordinates[1], 'f', 1, 32)

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

		fmt.Printf("Endpoint ...........%v\n", endpoint)
		fmt.Printf("Place ID ...........%v\n", placeId)
		fmt.Printf("ID .................%v\n", tweet.Id())
		fmt.Printf("Tweet ..............%v\n", tweet.Text())
		fmt.Printf("User ...............%v\n", tweet.User().Name())
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
