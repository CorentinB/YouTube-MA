package main

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"os"
	"time"
)

var logFileName string

const inputChanSize = 512
const outputChanSize = 32

func archiveWorker(input, done chan string, inProgress mapset.Set) {
	for id := range input {
		if inProgress.Contains(id) {
			continue
		}
		inProgress.Add(id)

		shouldMarkAsArchived := archiveID(id)

		if shouldMarkAsArchived {
			done <- id
		} else {
			inProgress.Remove(id)
		}
	}
}

func markAsArchivedWorker(done chan string, inProgress mapset.Set) {
	var buffer []string

	for id := range done {
		buffer = append(buffer, id)
		inProgress.Remove(id)

		if len(buffer) > outputChanSize {
			err := markIDsArchived(buffer...)
			buffer = nil

			if err != nil {
				//TODO:  better error message
				fmt.Fprintf(os.Stderr, "Error while marking IDs as archived: %s\n", err.Error())
			}
		}
	}
}

func main() {
	// Parse arguments
	parseArgs(os.Args)

	// Generate log file name based on current time
	t := time.Now()
	logFileName = t.Format("20060102150405") + ".log"

	inputIds := make(chan string, inputChanSize)
	doneIds := make(chan string, outputChanSize)

	// To avoid working on the same video in multiple goroutines
	inProgress := mapset.NewSet()

	// "Mark as archived" worker
	go markAsArchivedWorker(doneIds, inProgress)

	// "Archive" worker
	for i := 0; i < arguments.Concurrency; i++ {
		go archiveWorker(inputIds, doneIds, inProgress)
	}

	// "Fetch IDs" worker
	for {
		for _, id := range getID(arguments.Secret, 0, inputChanSize*2) {
			inputIds <- id
		}
	}
}
