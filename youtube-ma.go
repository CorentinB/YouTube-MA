package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
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

// Tracks structure containing all subtitles tracks for the video
type Tracklist struct {
	Tracks []Track `xml:"track"`
}

type Track struct {
	LangCode string `xml:"lang_code,attr"`
	Lang     string `xml:"lang_translated,attr"`
}

func fetchAnnotations(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	// requesting annotations from YouTube
	resp, err := http.Get("https://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + video.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		annotations := string(bodyBytes)
		video.Annotations = annotations
	} else {
		fmt.Fprintf(os.Stderr, "Error: unable to fetch annotations.\n")
		os.Exit(1)
	}
}

func writeFiles(video *Video) {
	// write annotations
	annotationsFile, errAnno := os.Create(video.Path + video.ID + "_" + video.Title + ".annotations.xml")
	if errAnno != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		os.Exit(1)
	}
	defer annotationsFile.Close()
	// write description
	descriptionFile, errDescription := os.Create(video.Path + video.ID + "_" + video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		os.Exit(1)
	}
	defer descriptionFile.Close()
	// write info json file
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
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + ".jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	// get the data
	resp, err := http.Get(video.Thumbnail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseDescription(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract description
	video.Description = strings.TrimSpace(document.Find("#eow-description").Text())
}

func parseVariousInfo(video *Video, body []byte, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract various info json
	re := regexp.MustCompile("ytplayer.config = (.*?);ytplayer.load")
	matches := re.FindSubmatch(body)
	var jsonConfig map[string]interface{}
	if len(matches) > 1 {
		err := json.Unmarshal(matches[1], &jsonConfig)
		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}
	}
	// normalize json text and write it into the structure
	byteArray := norm.NFC.Bytes(matches[1])
	video.InfoJSON = string(byteArray[:])
}

func parseThumbnailURL(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract thumbnail url
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:image" {
			thumbnailURL, _ := s.Attr("content")
			video.Thumbnail = thumbnailURL
		}
	})
}

func parseTitle(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract title
	title := strings.TrimSpace(document.Find("#eow-title").Text())
	video.Title = strings.Replace(title, " ", "_", -1)
}

func parseHTML(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	var workers sync.WaitGroup
	workers.Add(4)
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
	go parseTitle(video, document, &workers)
	go parseDescription(video, document, &workers)
	go parseVariousInfo(video, body, &workers)
	go parseThumbnailURL(video, document, &workers)
	workers.Wait()
	defer html.Body.Close()
}

func genPath(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
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
}

func downloadSub(video *Video, langCode string, lang string, wg *sync.WaitGroup) {
	defer wg.Done()
	color.Green("Downloading " + lang + " subtitle.." + "[" + langCode + "]")
	// generate subtitle URL
	url := "https://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + "." + langCode + ".xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	// get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func fetchSubsList(video *Video) {
	var wg sync.WaitGroup
	// request subtitles list
	res, err := http.Get("https://video.google.com/timedtext?hl=en&type=list&v=" + video.ID)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	// defer it!
	defer res.Body.Close()
	// check status, exit if != 200
	if res.StatusCode != 200 {
		log.Fatalf("Status code error while fetching subtitles for %s: %d %s", video.ID, res.StatusCode, res.Status)
	}
	// reading tracks list as a byte array
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	var tracks Tracklist
	xml.Unmarshal(data, &tracks)
	wg.Add(len(tracks.Tracks))
	for _, track := range tracks.Tracks {
		go downloadSub(video, track.LangCode, track.Lang, &wg)
	}
	wg.Wait()
}

func main() {
	var wg sync.WaitGroup
	start := time.Now()
	video := new(Video)
	args := os.Args[1:]
	video.ID = args[0]
	color.Green("Archiving ID: " + video.ID)
	wg.Add(3)
	go genPath(video, &wg)
	color.Green("Fetching annotations..")
	go fetchAnnotations(video, &wg)
	color.Green("Parsing description, title and thumbnail..")
	go parseHTML(video, &wg)
	wg.Wait()
	color.Green("Writing informations locally..")
	writeFiles(video)
	color.Green("Downloading thumbnail..")
	downloadThumbnail(video)
	color.Green("Fetching subtitles..")
	fetchSubsList(video)
	color.Cyan("Done in %s!", time.Since(start))
}
