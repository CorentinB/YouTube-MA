package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/tidwall/gjson"
)

type Video struct {
	Title       string
	Annotations string
	Thumbnail   string
	Oembed      string
	Description string
	Html        string
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

func fetchThumbnail(video *Video) *Video {
	thumbnail := gjson.Get(video.Oembed, "thumbnail_url")
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
	htmlFile, errFile := os.Create(video.Title + ".html")
	if errFile != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		os.Exit(1)
	}
	defer htmlFile.Close()
	fmt.Fprintf(annotationsFile, video.Annotations)
	fmt.Fprintf(htmlFile, video.Html)
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

func fetchYoutubeHtml(id string, video *Video) *Video {
	// Request the HTML page.
	res, err := http.Get("https://youtube.com/watch?v=" + id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("Status code error: %d %s", res.StatusCode, res.Status)
	}
	bytes, _ := ioutil.ReadAll(res.Body)
	video.Html = string(bytes)
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	doc.Find("title").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		title := s.Find("title").Text()
		color.Red("Debug title")
		fmt.Printf("Title: %s\n", title)
	})
	return video
}

func main() {
	start := time.Now()
	video := new(Video)
	args := os.Args[1:]
	id := args[0]
	//key := args[1]
	color.Green("Archiving ID: " + id)
	color.Green("Fetching HTML raw page..")
	video = fetchYoutubeHtml(id, video)
	color.Green("Fetching data from Oembed..")
	video = fetchOembed(id, video)
	color.Green("Parsing title..")
	video = fetchTitle(video)
	//color.Green("Parsing description..")
	//video = fetchDescription(video)
	color.Green("Parsing thumbnail URL..")
	video = fetchThumbnail(video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching annotations..")
	video = fetchAnnotations(id, video)
	color.Green("Writing informations locally..")
	writeFiles(video)

	color.Cyan("Done in %s!", time.Since(start))
}
