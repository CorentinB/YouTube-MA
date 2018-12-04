package main

import (
	"bufio"
	"log"
	"os"
	"sync"
)

func processSingleID(ID string, worker *sync.WaitGroup) {
	defer worker.Done()
	var wg sync.WaitGroup
	video := new(Video)
	video.ID = ID
	video.InfoJSON.Subtitles = make(map[string][]Subtitle)
	video.playerArgs = make(map[string]interface{})
	checkFiles(video)
	logInfo("-", video, "Archiving started.")
	wg.Add(2)
	logInfo("~", video, "Fetching annotations..")
	go fetchAnnotations(video, &wg)
	logInfo("~", video, "Parsing infos, description, title and thumbnail..")
	go parseHTML(video, &wg)
	wg.Wait()
	genPath(video)
	logInfo("~", video, "Fetching subtitles..")
	fetchSubsList(video)
	logInfo("~", video, "Writing informations locally..")
	writeFiles(video)
	logInfo("~", video, "Downloading thumbnail..")
	downloadThumbnail(video)
	logInfo("✓", video, "Archiving complete!")
}

func processList(maxConc int64, path string, worker *sync.WaitGroup) {
	defer worker.Done()
	var count int64
	// start workers group
	var wg sync.WaitGroup
	// open file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// scan the list line by line
	scanner := bufio.NewScanner(file)
	// scan the list line by line
	for scanner.Scan() {
		count++
		wg.Add(1)
		go processSingleID(scanner.Text(), &wg)
		if count == maxConc {
			wg.Wait()
			count = 0
		}
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}
