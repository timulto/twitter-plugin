package main

import (
	"net/http"
	"io"
	"time"
	"fmt"

)

var (

	tick *time.Ticker
	doneChan chan(bool)
)

func GetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	toRender := "<!DOCTYPE html>" +
			"<html>" +
			"<head lang='en'>" +
			"<meta charset='UTF-8'>" +
			"<title>Tweety Console</title>" +
			"</head>" +
			"<script>" +
			"function setInterval() {" +
			"document.getElementById('startln').href = 'http://localhost:8000/startTicker?interval=' + document.getElementById('interval').value;" +
			"}" +
			"</script>" +
			"<body>" +
			"<div id='commands'>" +
			"<ul>" +
			"<li><a href='http://localhost:8000/startTicker?interval=10s' id='startln'>Start</a> <input type='text' id='interval' onkeyup='setInterval()' value='10s'></li>" +
			"<li><a href='http://localhost:8000/stopTicker'>Stop</a></li>" +
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
		fmt.Println("go func")
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
		fmt.Println("go func")
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