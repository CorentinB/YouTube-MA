package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func parseThumbnailURL(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract thumbnail url
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:image" {
			thumbnailURL, _ := s.Attr("content")
			video.Thumbnail = strings.Replace(thumbnailURL, "https", "http", -1)
		}
	})
}

func downloadThumbnail(video *Video) {
	if len(video.Thumbnail) < 1 {
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + ".jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer out.Close()
	// get the data
	resp, err := http.Get(video.Thumbnail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
}
