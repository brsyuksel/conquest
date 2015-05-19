package main

import (
	"conquest/conquest"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	users               uint64
	timeout, configfile string
	sequential          bool
)

func init() {
	flag.Uint64Var(&users, "u", 10, "concurrent users.")
	flag.StringVar(&timeout, "t", "TIMEm",
		"time for requests. Use s, m, h modifiers for m")
	flag.StringVar(&configfile, "c", "conquest.js", "conquest js file path")
	flag.BoolVar(&sequential, "s", false, "do transactions in sequential mode")
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
		case "t":
			var duration time.Duration
			duration, err = time.ParseDuration(timeout)
			if err == nil {
				conq.Duration = &duration
			}
		case "s":
			conq.Sequential = sequential
		}
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(conq)

}
