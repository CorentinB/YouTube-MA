package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

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
	//color.Cyan("[DEBUG] API URL FETCHED: " + url)
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

func fetchTitle(video *Video) *Video {
	title := gjson.Get(video.Raw, "items.0.snippet.title")
	video.Title = title.String()
	return video
}

func fetchThumbnail(video *Video) *Video {
	thumbnail := gjson.Get(video.Raw, "items.0.snippet.thumbnails.maxres.url")
	video.Thumbnail = thumbnail.String()
	return video
}

func writeFiles(video *Video) {
	video.Title = strings.Replace(video.Title, " ", "_", -1)
	annotationsFile, errAnno := os.Create(video.Title + ".annotations.xml")
	if errAnno != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		os.Exit(1)
	}
	defer annotationsFile.Close()
	descriptionFile, errDesc := os.Create(video.Title + ".description")
	if errDesc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDesc)
		os.Exit(1)
	}
	defer descriptionFile.Close()
	infoFile, errInfo := os.Create(video.Title + ".info.json")
	if errInfo != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errInfo)
		os.Exit(1)
	}
	defer infoFile.Close()
	fmt.Fprintf(annotationsFile, video.Annotations)
	fmt.Fprintf(descriptionFile, video.Description)
	fmt.Fprintf(infoFile, video.Raw)
}

func downloadThumbnail(video *Video) {
	video.Title = strings.Replace(video.Title, " ", "_", -1)
	// Create the file
	out, err := os.Create(video.Title + ".jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	// Get the data
	resp, err := http.Get(video.Thumbnail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	key := args[1]
	color.Green("Archiving ID: " + id)
	color.Green("Fetching data from API..")
	video = fetchRawData(id, key, video)
	color.Green("Parsing title..")
	video = fetchTitle(video)
	color.Green("Parsing description..")
	video = fetchDescription(video)
	color.Green("Parsing thumbnail URL..")
	video = fetchThumbnail(video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching annotations..")
	video = fetchAnnotations(id, video)
	color.Green("Writing informations locally..")
	writeFiles(video)
	color.Cyan("Done!")
}
