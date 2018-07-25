package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/unicode/norm"

	"github.com/fatih/color"
)

// Video structure containing all metadata for the video
type Video struct {
	ID          string
	Title       string
	Annotations string
	Thumbnail   string
	Description string
	Path        string
	InfoJSON    string
}

func fetchAnnotations(video *Video) *Video {
	// Requesting annotations from YouTube
	resp, err := http.Get("https://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + video.ID)
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
	// Write annotations
	annotationsFile, errAnno := os.Create(video.Path + video.ID + "_" + video.Title + ".annotations.xml")
	if errAnno != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		os.Exit(1)
	}
	defer annotationsFile.Close()
	// Write description
	descriptionFile, errDescription := os.Create(video.Path + video.ID + "_" + video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		os.Exit(1)
	}
	defer descriptionFile.Close()
	// Write info json file
	infoFile, errInfo := os.Create(video.Path + video.ID + "_" + video.Title + ".info.json")
	if errInfo != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errInfo)
		os.Exit(1)
	}
	defer infoFile.Close()
	fmt.Fprintf(annotationsFile, "%s", video.Annotations)
	fmt.Fprintf(descriptionFile, "%s", video.Description)
	fmt.Fprintf(infoFile, "%s", video.InfoJSON)
}

func downloadThumbnail(video *Video) {
	// Create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + ".jpg")
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

func parseHTML(video *Video) *Video {
	// request video html page
	html, err := http.Get("https://youtube.com/watch?v=" + video.ID)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	// check status, exit if != 200
	if html.StatusCode != 200 {
		log.Fatalf("Status code error for %s: %d %s", video.ID, html.StatusCode, html.Status)
	}
	body, err := ioutil.ReadAll(html.Body)
	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	// extract description and title
	video.Description = strings.TrimSpace(document.Find("#eow-description").Text())
	// extract various info json
	re := regexp.MustCompile("ytplayer.config = (.*?);ytplayer.load")
	matches := re.FindSubmatch(body)
	var jsonConfig map[string]interface{}
	if len(matches) > 1 {
		err = json.Unmarshal(matches[1], &jsonConfig)
		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}
	}
	// normalize json text and write it into the structure
	byteArray := norm.NFC.Bytes(matches[1])
	video.InfoJSON = string(byteArray[:])
	// extract thumbnail url
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:image" {
			thumbnailURL, _ := s.Attr("content")
			video.Thumbnail = thumbnailURL
		}
	})
	// extract title
	video.Title = strings.TrimSpace(document.Find("#eow-title").Text())
	defer html.Body.Close()
	return video
}

func genPath(video *Video) *Video {
	firstChar := video.ID[:1]
	video.Path = firstChar + "/" + video.ID + "/"
	// create directory if it doesnt exist
	if _, err := os.Stat(video.Path); os.IsNotExist(err) {
		err = os.MkdirAll(video.Path, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	return video
}

func main() {
	start := time.Now()
	video := new(Video)
	args := os.Args[1:]
	video.ID = args[0]
	color.Green("Archiving ID: " + video.ID)
	video = genPath(video)
	color.Green("Parsing description, title and thumbnail..")
	video = parseHTML(video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching annotations..")
	video = fetchAnnotations(video)
	color.Green("Writing informations locally..")
	writeFiles(video)
	color.Cyan("Done in %s!", time.Since(start))
}
