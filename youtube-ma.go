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
)

// Structure containing all metadata for the video
type Video struct {
	Title       string
	Annotations string
	Thumbnail   string
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

func writeFiles(video *Video) {
	video.Title = strings.Replace(video.Title, " ", "_", -1)
	annotationsFile, errAnno := os.Create(video.Title + ".annotations.xml")
	if errAnno != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		os.Exit(1)
	}
	defer annotationsFile.Close()
	descriptionFile, errDescription := os.Create(video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		os.Exit(1)
	}
	defer descriptionFile.Close()
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
	// Parsing thumbnail
	thumbnail := doc.Find("meta", "property", "og:image")
	video.Thumbnail = string(thumbnail.Attrs()["content"])
	// Parsing title
	title := doc.Find("meta", "property", "og:title")
	video.Title = strings.Replace(string(title.Attrs()["content"]), " ", "_", -1)
	// Writing description to the structure
	video.Description = buffer.String()
	return video
}

func main() {
	start := time.Now()
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	color.Green("Archiving ID: " + id)
	color.Green("Parsing description, title and thumbnail..")
	video = parseHTML(id, video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching annotations..")
	video = fetchAnnotations(id, video)
	color.Green("Writing informations locally..")
	writeFiles(video)
	color.Cyan("Done in %s!", time.Since(start))
}
