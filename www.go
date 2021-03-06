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
	toRender := "<!DOCTYPE html>" +
			"<html>" +
			"<head lang='en'>" +
			"<meta charset='UTF-8'>" +
			"<title>Tweety Console</title>" +
			"</head>" +
			"<script>" +
			"function setInterval() {" +
			"document.getElementById('startln').href = '/startTicker?interval=' + document.getElementById('interval').value;" +
			"}" +
			"</script>" +
			"<body>" +
			"<div id='commands'>" +
			"<ul>" +
			"<li><a href='/startTicker?interval=300s' id='startln'>Start</a> <input type='text' id='interval' onkeyup='setInterval()' value='300s'></li>" +
			"<li><a href='/stopTicker'>Stop</a></li>" +
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

	doneChan = make(chan bool)

	go func() {
		publish(nil, nil)
		fmt.Println("waiting for next publishing....")

		for {
			select {
			case t:= <- ticker.C:
				publish(nil, nil)
				fmt.Printf("Published at: %v\n", t)
			case <- doneChan:
				ticker.Stop()
				fmt.Println("Publishing paused")
			}
		}
	}()

	if w != nil {
		io.WriteString(w, "Started at "+time.Now().Local().Format("2006-01-02 15:04:05 +0800")+"\n")
	}
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
 	doneChan <- true
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
