package main

import (
	"conquest/conquest"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	users                       uint64
	timeout, configfile, output string
	sequential                  bool
)

func init() {
	flag.Uint64Var(&users, "u", 10, "concurrent users.")
	flag.StringVar(&timeout, "t", "30s",
		"time for requests stack. Use s, m, h modifiers")
	flag.StringVar(&output, "o", "", "output file for fail transactions logs")
	flag.StringVar(&configfile, "c", "conquest.js", "conquest js file path")
	flag.BoolVar(&sequential, "s", false, "do transactions in sequential mode")
}

func main() {
	var err error
	flag.Parse()

	fmt.Println("conquest v0.1.0\n")

	if _, err := os.Stat(configfile); os.IsNotExist(err) {
		fmt.Println(configfile, "file not found")
		os.Exit(1)
	}

	conq, err := conquest.RunScript(configfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "u":
			conq.TotalUsers = users
		case "t":
			var duration time.Duration
			duration, err = time.ParseDuration(timeout)
			if err == nil {
				conq.Duration = duration
			}
		case "s":
			conq.Sequential = sequential
		}
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("performing transactions...\n")
	
	var fo *os.File
	if output == "" {
		fo = os.Stdout
	} else {
		fo, err = os.Create(output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	reporter := conquest.NewReporter(fo)

	err = conquest.Perform(conq, reporter)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	<-reporter.C.Done
}
