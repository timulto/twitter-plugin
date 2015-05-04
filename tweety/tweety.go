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
    "log"
	"strconv"
)

func LoadCredentials() (client *twittergo.Client, err error) {

	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if err != nil {
		return
	}

	lines := strings.Split(string(credentials), "\n");
	config := &oauth1a.ClientConfig {
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}

	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3]);
	client = twittergo.NewClient(config, user);

	return
}

func GetBody(message string, media []byte, address string, createdAt string) (body io.ReadWriter, header string, err error) {
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

	//placeId := getPlaceId(latitude, longitude)
    //mp.WriteField("place_id", placeId)
	writer, err = mp.CreateFormField("media[]")
	if err != nil {
		return
	}
	writer.Write(media)
	header = fmt.Sprintf("multipart/form-data;boundary=%v", mp.Boundary())
	mp.Close()
	return
}

type location struct {
      Type string
      Coordinates []float32
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


	if err != nil {
		fmt.Printf("Error while requesting data %v\n", err)
	}

    jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
    //s := string(jsonDataFromHttp[:])
    // fmt.Printf(s)

    if err != nil {
        log.Fatal(err)
    }
    //data []Fine
    err = json.Unmarshal(jsonDataFromHttp, &data)

    if err != nil {
        log.Fatal(err)
    }
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
	Id string
}

type Place struct {
	Result []PlaceResult
}

func getPlaceId(latitude float32, longitude float32) (p string) {

	var client *twittergo.Client

	var data Place

	client, _ = LoadCredentials();

	twitterUrl := "https://api.twitter.com/1.1/geo/search.json"
	req, _ := http.NewRequest("GET", twitterUrl, nil)

	resp, err := client.SendRequest(req)

	if err != nil {
		fmt.Println("error:", err)
		return
	}

	jsonDataFromHttp, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		fmt.Println("error:", err1)
		return
	}

	err2 := json.Unmarshal(jsonDataFromHttp, &data)
	if err2 != nil {
		fmt.Println("error:", err2)
		return
	}

	return data.Result[0].Id

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
	if err != nil {
		fmt.Printf("Could not parse CREDENTIALS file: %v\n", err);
		os.Exit(1);
	}

    var toTwet = getFines()
    var image []byte
	endpoint := "/1.1/statuses/update_with_media.json"

	if len(toTwet) == 0 {
	    fmt.Println("No mew tweet to post")
	}

    for _, element := range toTwet {

		fmt.Println(element.Id)

        image = decode(element.ImageData[22:])
        body, header, err := GetBody(element.Text, image, element.Address, element.CreatedAt)

        if err != nil {
            fmt.Printf("Problem loading body: %v\n", err)
            os.Exit(1)
        }

        req, err = http.NewRequest("POST", endpoint, body)
        if err != nil {
            fmt.Printf("Could not parse request: %v\n", err)
            os.Exit(1)
        }
        req.Header.Set("Content-Type", header)

        resp, err = client.SendRequest(req)
        if err != nil {
            fmt.Printf("Could not send request: %v\n", err)
            os.Exit(1)
        }
        tweet = &twittergo.Tweet{}
        err = resp.Parse(tweet)
        if err != nil {
            fmt.Printf("Problem parsing response: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("ID:                         %v\n", tweet.Id())
        fmt.Printf("Tweet:                      %v\n", tweet.Text())
        fmt.Printf("User:                       %v\n", tweet.User().Name())

		// Mark fine posted on twitter
        url1 := "http://beta.timulto.org/api/fine/" + element.Id +  "/twitter"
		tId := strconv.FormatUint(tweet.Id(), 10)

        parameters := url.Values{}
		parameters.Add("postId", tId)
		resp1, err1 :=http.PostForm(url1, parameters)

        if err1 != nil {
            fmt.Println("Problem while marking tweet " + tId + " published.")
        } else {
			fmt.Println("Tweet " + tId + " marked as published.")
		}
        _, err1 = ioutil.ReadAll(resp1.Body)

//        if resp.HasRateLimit() {
//            fmt.Printf("Rate limit:                 %v\n", resp.RateLimit())
//            fmt.Printf("Rate limit remaining:       %v\n", resp.RateLimitRemaining())
//            fmt.Printf("Rate limit reset:           %v\n", resp.RateLimitReset())
//        } else {
//            fmt.Printf("Could not parse rate limit from response.\n")
//        }
//        if resp.HasMediaRateLimit() {
//            fmt.Printf("Media Rate limit:           %v\n", resp.MediaRateLimit())
//            fmt.Printf("Media Rate limit remaining: %v\n", resp.MediaRateLimitRemaining())
//            fmt.Printf("Media Rate limit reset:     %v\n", resp.MediaRateLimitReset())
//        } else {
//            fmt.Printf("Could not parse media rate limit from response.\n")
//        }
    }
}
