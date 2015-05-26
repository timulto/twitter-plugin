package main

import (
	"net/http"
	"io"
	"time"
	"fmt"
	"net"
	"os"

)

var (

	tick *time.Ticker
	doneChan chan(bool)
)

func GetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	port := os.Getenv("PORT")
	toRender := "<!DOCTYPE html>" +
			"<html>" +
			"<head lang='en'>" +
			"<meta charset='UTF-8'>" +
			"<title>Tweety Console</title>" +
			"</head>" +
			"<script>" +
			"function setInterval() {" +
			"document.getElementById('startln').href = ':" + port + "/startTicker?interval=' + document.getElementById('interval').value;" +
			"}" +
			"</script>" +
			"<body>" +
			"<div id='commands'>" +
			"<ul>" +
			"<li><a href=':" + port + "/startTicker?interval=10s' id='startln'>Start</a> <input type='text' id='interval' onkeyup='setInterval()' value='10s'></li>" +
			"<li><a href=':" + port + "/stopTicker'>Stop</a></li>" +
			"</ul>" +
			"</div>" +
			"</body>" +
			"</html>"

	io.WriteString(w, toRender)
}


func StartTriggering(w http.ResponseWriter, r *http.Request) {

	if w != nil {
		interval := r.URL.Query().Get("interval")

		if interval == "" {
			io.WriteString(w, "Interval parameter not specified!\n")
			return
		}

		timerRange = interval;
	}

	d, err := time.ParseDuration(timerRange)
	ErrorHandling(err, "Error while parsing duration: ", 1)
	ticker := time.NewTicker(d)
	tick = ticker

	//doneChan = make(chan bool)

	go func() {
		publish(nil, nil)
		fmt.Println("waiting for next publishing....")
		for t := range ticker.C {
			publish(nil, nil)
			if w != nil {
				io.WriteString(w, "Published at " + time.Now().Local().Format("2006-01-02 15:04:05 +0800") + "\n")
			} else {
				fmt.Printf("Published at: %v\n", t)
			}
		}
	}()

	//<- doneChan
}

func StartTriggeringBatch(timerRange string) {

	fmt.Println("TimerRange: " + timerRange)

	d, err := time.ParseDuration(timerRange)
	ErrorHandling(err, "Error while parsing duration: ", 1)
	ticker := time.NewTicker(d)
	tick = ticker

	doneChan = make(chan bool)

	go func() {
		publish(nil, nil)
		fmt.Println("waiting for next publishing....")
		for t := range ticker.C {
			publish(nil, nil)
			fmt.Printf("Published at: %v\n", t)
		}
	}()
	<- doneChan
}

func StopTicker(w http.ResponseWriter, r *http.Request) {
	StopTriggering()
	io.WriteString(w, "Publisher stopped")
}

func StopTriggering() {
 	tick.Stop()
}

func getLocalIP() string {

	var ip string

	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println(ipnet.IP.String())
				ip = ipnet.String()
			}
		}
	}
	return ip
}
