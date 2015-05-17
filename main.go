package main

import (
	"conquest/conquest"
	"flag"
	"fmt"
	"os"
)

var (
	users, requests     int
	timeout, configfile string
)

func init() {
	flag.IntVar(&users, "u", 10, "concurrent users.")
	flag.IntVar(&requests, "r", 100, "total requests will be achieved by users.")
	flag.StringVar(&timeout, "t", "TIMEm", "time for requests. Use s, m, h modifiers for m")
	flag.StringVar(&configfile, "c", "conquest.js", "conquest js file path")
}

func main() {
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

	fmt.Println(conq)

}
