package main

import (
	"os"
	"time"

	"github.com/remeh/sizedwaitgroup"
)

var logFileName string

func main() {
	// Parse arguments
	parseArgs(os.Args)

	// Worker group
	var worker = sizedwaitgroup.New(arguments.Concurrency)

	// Generate log file name based on current time
	t := time.Now()
	logFileName = t.Format("20060102150405") + ".log"

	// Main processing loop
	for i := 0; ; i++ {
		worker.Add()
		ID := getID(arguments.Secret, i, 1)
		go archiveID(ID[0], &worker)
	}
}
