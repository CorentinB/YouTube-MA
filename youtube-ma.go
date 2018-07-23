package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/tidwall/gjson"
)

type Video struct {
	Title       string
	Author      string
	Annotations string
	Thumbnail   string
	Raw         string
	Description string
}

func fetchBasic(id string) *Video {
	video := new(Video)
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
		title := gjson.Get(video.Raw, "title")
		video.Title = title.String()
		author := gjson.Get(video.Raw, "author_name")
		video.Author = author.String()
		thumbnail := gjson.Get(video.Raw, "thumbnail_url")
		video.Thumbnail = thumbnail.String()
	} else {
		color.Red("Error: unable to fetch basic informations from oembed service.")
	}
	return video
}

func fetchAnnotations(id string, video *Video) *Video {
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
	} else {
		color.Red("Error: unable to fetch annotations.")
	}
	return video
}

func fetchRawData(id string, key string, video *Video) *Video {
	// Requesting raw data through YouTube's API
	url := "https://www.googleapis.com/youtube/v3/videos?part=snippet&id=" + id + "&key=" + key
	color.Cyan("[DEBUG] API URL FETCHED: " + url)
	resp, err := http.Get(url)
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
		rawData := string(bodyBytes)
		video.Raw = rawData
	} else {
		color.Red("Error: unable to fetch raw video's data from API.")
	}
	return video
}

func fetchDescription(video *Video) *Video {
	value := gjson.Get(video.Raw, "items.0.snippet.description")
	video.Description = value.String()
	return video
}

func main() {
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	key := args[1]
	color.Cyan("[DEBUG] ID: " + id)
	color.Cyan("[DEBUG] API key: " + key)
	video = fetchBasic(id)
	color.Green("\nTitle: " + video.Title)
	color.Green("Author: " + video.Author)
	color.Green("Thumbnail: " + video.Thumbnail)
	video = fetchAnnotations(id, video)
	color.Green("Annotations: " + video.Annotations)
	video = fetchRawData(id, key, video)
	video = fetchDescription(video)
	color.Green("Description: " + video.Description)
}
