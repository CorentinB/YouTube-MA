package main

import (
	"bufio"
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
	"github.com/labstack/gommon/color"
	"golang.org/x/text/unicode/norm"
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
	resp, err := http.Get("http://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + video.ID)
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
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch annotations!"))
		video.Annotations = "Unable to fetch annotations."
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
	if 1 < len(matches) {
		byteArray := norm.NFC.Bytes(matches[1])
		video.InfoJSON = string(byteArray[:])
	} else {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch json informations!"))
		video.InfoJSON = "Unable to fetch infos."
	}
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
	video.Title = strings.Replace(video.Title, "/", "_", -1)
}

func parseHTML(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	var workers sync.WaitGroup
	workers.Add(4)
	// request video html page
	html, err := http.Get("http://youtube.com/watch?v=" + video.ID + "&bpctr=1532537335")
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

func genPath(video *Video) {
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
	url := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading ") + color.Yellow(lang) + color.Green(" subtitle.."))
	// get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + "." + langCode + ".xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
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
	res, err := http.Get("http://video.google.com/timedtext?hl=en&type=list&v=" + video.ID)
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

func processSingleID(ID string) {
	var wg sync.WaitGroup
	video := new(Video)
	video.ID = ID
	color.Println(color.Yellow("[") + color.Green("-") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Archiving started."))
	wg.Add(2)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching annotations.."))
	go fetchAnnotations(video, &wg)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Parsing infos, description, title and thumbnail.."))
	go parseHTML(video, &wg)
	wg.Wait()
	genPath(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Writing informations locally.."))
	writeFiles(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading thumbnail.."))
	downloadThumbnail(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching subtitles.."))
	fetchSubsList(video)
	color.Println(color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Archiving complete!"))
}

func processSingleIDFromList(ID string, worker *sync.WaitGroup) {
	defer worker.Done()
	var wg sync.WaitGroup
	video := new(Video)
	video.ID = ID
	color.Println(color.Yellow("[") + color.Green("-") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Archiving started."))
	wg.Add(2)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching annotations.."))
	go fetchAnnotations(video, &wg)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Parsing description, title and thumbnail.."))
	go parseHTML(video, &wg)
	wg.Wait()
	genPath(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Writing informations locally.."))
	writeFiles(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading thumbnail.."))
	downloadThumbnail(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching subtitles.."))
	fetchSubsList(video)
	color.Println(color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Archiving complete!"))
}

func processList(path string) {
	var count int
	// start workers group
	var wg sync.WaitGroup
	// open file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// scan the list line by line
	scanner := bufio.NewScanner(file)
	// count number of IDs
	for scanner.Scan() {
		count++
		wg.Add(1)
		go processSingleIDFromList(scanner.Text(), &wg)
		if count == 32 {
			wg.Wait()
			count = 0
		}
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

func argumentParsing(args []string) {
	if len(args) > 1 {
		color.Red("Usage: ./youtube-ma [ID or list of IDs]")
		os.Exit(1)
	}
	if _, err := os.Stat(args[0]); err == nil {
		processList(args[0])
	} else {
		processSingleID(args[0])
	}
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
