package main

import (
	"log"
	"os"
	"time"
)

// Returns true if we should mark as archived
func archiveID(ID string) (archived bool) {

	// Record start time
	start := time.Now()

	// Create custom logger for this job
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	workerLog := log.New(f, ID+" ", log.LstdFlags)

	// Define structures for the video
	video := new(Video)
	video.ID = ID
	video.InfoJSON.Subtitles = make(map[string][]Subtitle)
	video.playerArgs = make(map[string]interface{})

	// Set thumbnail URL
	video.Thumbnail = "http://i3.ytimg.com/vi/" + ID + "/maxresdefault.jpg"

	// Check if the files already exists
	err = checkFiles(video)
	if err != nil {
		return true
	}

	// Generate path to store files
	err = genPath(video)
	if err != nil {
		workerLog.Fatalln(err)
	}

	// Get HTML of the page and parse it
	err = parseHTML(video)
	if err != nil {
		workerLog.Println(err)
		os.RemoveAll(video.Path)
		return false
	}

	// Fetch subtitles
	err = fetchSubs(video)
	if err != nil {
		workerLog.Println(err)
		os.RemoveAll(video.Path)
		return false
	}

	// Write metadata to files
	err = writeFiles(video)
	if err != nil {
		workerLog.Println(err)
		os.RemoveAll(video.Path)
		return false
	}

	// Download the thumbnail
	err = downloadThumbnail(video)
	if err != nil {
		workerLog.Println(err)
		os.RemoveAll(video.Path)
		return false
	}

	workerLog.Println("archiving completed in " + time.Since(start).String())
	return true
}
