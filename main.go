package main

import (
	"conquest/conquest"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	users, requests     uint64
	timeout, configfile string
)

func init() {
	flag.Uint64Var(&users, "u", 10, "concurrent users.")
	flag.Uint64Var(&requests, "r", 100, "total requests will be achieved by users.")
	flag.StringVar(&timeout, "t", "TIMEm",
		"time for requests. Use s, m, h modifiers for m")
	flag.StringVar(&configfile, "c", "conquest.js", "conquest js file path")
}

func main() {
	var err error
	flag.Parse()

	if _, err := os.Stat(configfile); os.IsNotExist(err) {
		fmt.Println(configfile, "file not found")
		os.Exit(1)
	}

	conq, err := conquest.RunScript(configfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "u":
			conq.TotalUsers = users
		case "r":
			conq.TotalRequests = requests
		case "t":
			var duration time.Duration
			duration, err = time.ParseDuration(timeout)
			if err == nil {
				conq.Duration = &duration
			}
		}
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(conq)

}
