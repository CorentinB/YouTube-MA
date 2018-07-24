package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/fatih/color"
	"github.com/tidwall/gjson"
)

// Structure containing all metadata for the video
type Video struct {
	Title       string
	Annotations string
	Thumbnail   string
	Oembed      string
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

func fetchOembed(id string, video *Video) *Video {
	// Requesting raw data through YouTube's API
	url := "https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=" + id + "&format=json"
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
		oembed := string(bodyBytes)
		video.Oembed = oembed
	} else {
		color.Red("Error: unable to fetch oembed video's data.")
	}
	return video
}

func fetchTitle(video *Video) *Video {
	title := gjson.Get(video.Oembed, "title")
	video.Title = strings.Replace(title.String(), " ", "_", -1)
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
	oembedFile, errOembed := os.Create(video.Title + ".oembed.json")
	if errOembed != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errOembed)
		os.Exit(1)
	}
	defer oembedFile.Close()
	descriptionFile, errDescription := os.Create(video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		os.Exit(1)
	}
	defer descriptionFile.Close()
	fmt.Fprintf(oembedFile, video.Oembed)
	fmt.Fprintf(annotationsFile, video.Annotations)
	fmt.Fprintf(descriptionFile, video.Description)
}

func downloadThumbnail(video *Video) {
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

func parseHTML(id string, video *Video) *Video {
	var buffer bytes.Buffer
	resp, err := soup.Get("https://youtube.com/watch?v=" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	doc := soup.HTMLParse(resp)
	description := doc.Find("div", "id", "watch-description-text").FindAll("p")
	for _, description := range description {
		buffer.WriteString(description.Text())
	}
	thumbnail := doc.Find("meta", "property", "og:image")
	video.Description = buffer.String()
	video.Thumbnail = string(thumbnail.Attrs()["content"])
	return video
}

func main() {
	start := time.Now()
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	color.Green("Archiving ID: " + id)
	color.Green("Fetching data from Oembed..")
	video = fetchOembed(id, video)
	color.Green("Parsing title..")
	video = fetchTitle(video)
	color.Green("Parsing description and thumbnail..")
	video = parseHTML(id, video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching annotations..")
	video = fetchAnnotations(id, video)
	color.Green("Writing informations locally..")
	writeFiles(video)
	color.Cyan("Done in %s!", time.Since(start))
}
