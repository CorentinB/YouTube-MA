package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/savaki/jq"
)

type Video struct {
	Title       string
	Author      string
	Annotations string
	Thumbnail   string
}

func fetchingBasic(id string) *Video {
	video := new(Video)
	// Declare jq operations
	getTitle, _ := jq.Parse(".title")
	getAuthor, _ := jq.Parse(".author_name")
	getThumb, _ := jq.Parse(".thumbnail_url")
	// Requesting data from oembed (allow getting title, author, thumbnail url)
	resp, err := http.Get("https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=" + id + "&format=json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// Checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		bodyString := string(bodyBytes)
		output := []byte(bodyString)
		// Parsing data
		title, _ := getTitle.Apply(output)
		video.Title = string(title)
		authorName, _ := getAuthor.Apply(output)
		video.Author = string(authorName)
		thumbnailUrl, _ := getThumb.Apply(output)
		video.Thumbnail = string(thumbnailUrl)
	}
	return video
}

func fetchingAnnotations(id string, video *Video) *Video {
	// Requesting annotations from YouTube
	resp, err := http.Get("https://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// Checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		annotations := string(bodyBytes)
		video.Annotations = annotations
	}
	return video
}

func main() {
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	key := args[1]
	fmt.Println("[DEBUG] ID: " + id)
	fmt.Println("[DEBUG] API key: " + key)
	video = fetchingBasic(id)
	fmt.Println("\nTitle: " + video.Title)
	fmt.Println("Author: " + video.Author)
	fmt.Println("Thumbnail: " + video.Thumbnail)
	video = fetchingAnnotations(id, video)
	fmt.Println("Annotations: " + video.Annotations)
}
