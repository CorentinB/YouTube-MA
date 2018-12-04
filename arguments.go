package main

import (
	"os"
	"strconv"
	"sync"

	"github.com/labstack/gommon/color"
)

func argumentParsing(args []string) {
	// start workers group
	var wg sync.WaitGroup
	var maxConc int64
	maxConc = 16
	wg.Add(1)
	if len(args) > 2 {
		color.Red("Usage: ./youtube-ma [ID or list of IDs] [CONCURRENCY]")
		os.Exit(1)
	} else if len(args) == 2 {
		if _, err := strconv.ParseInt(args[1], 10, 64); err == nil {
			maxConc, _ = strconv.ParseInt(args[1], 10, 64)
		} else {
			color.Red("Usage: ./youtube-ma [ID or list of IDs] [CONCURRENCY]")
			os.Exit(1)
		}
	}
	if _, err := os.Stat(args[0]); err == nil {
		go processList(maxConc, args[0], &wg)
	} else {
		go processSingleID(args[0], &wg)
	}
	wg.Wait()
}
