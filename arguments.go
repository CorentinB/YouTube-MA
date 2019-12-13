package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/akamensky/argparse"
)

var arguments = struct {
	Concurrency int
	Output      string
	Secret      string
	Proxy       *url.URL
	Verbose     bool
}{}

func parseArgs(args []string) {
	// Create new parser object
	parser := argparse.NewParser("YouTube-MA", "YouTube metadata archiver")

	concurrency := parser.Int("j", "concurrency", &argparse.Options{
		Required: false,
		Help:     "Concurrency",
		Default:  4})

	output := parser.String("o", "output", &argparse.Options{
		Required: false,
		Help:     "Output directory",
		Default:  "videos"})

	secret := parser.String("s", "secret", &argparse.Options{
		Required: true,
		Help:     "Secret youtube.the-eye.eu API key",
		Default:  false})

	verbose := parser.Flag("v", "verbose", &argparse.Options{
		Required: false,
		Help:     "Verbose output",
		Default:  false})

	proxy := parser.String("p", "proxy", &argparse.Options{
		Required: false,
		Help:     "Proxy url",
		Default:  ""})

	// Parse input
	err := parser.Parse(args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
		os.Exit(0)
	}

	if proxy != nil {
		arguments.Proxy, _ = url.Parse(*proxy)
	}

	// Remove trailing slash in output path
	*output = strings.Replace(*output, "/", "", -1)

	// Fill arguments structure
	arguments.Concurrency = *concurrency
	arguments.Output = *output
	arguments.Secret = *secret
	arguments.Verbose = *verbose
}
