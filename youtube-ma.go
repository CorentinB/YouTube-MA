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
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/wuriyanto48/replacer"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
)

// Video structure containing all metadata for the video
type Video struct {
	ID          string
	Title       string
	Annotations string
	Thumbnail   string
	Description string
	Path        string
	RawHTML     string
	InfoJSON    infoJSON
}

// Tracklist structure containing all subtitles tracks for the video
type Tracklist struct {
	Tracks []Track `xml:"track"`
}

// Track structure for data about single subtitle
type Track struct {
	LangCode string `xml:"lang_code,attr"`
	Lang     string `xml:"lang_translated,attr"`
}

// infoJSON structure containing the generated json data
type infoJSON struct {
	ID                string                `json:"id"`
	Uploader          string                `json:"uploader"`
	UploaderID        string                `json:"uploader_id"`
	UploaderURL       string                `json:"uploader_url"`
	UploadDate        string                `json:"upload_date"`
	License           string                `json:"license,omitempty"`
	Creator           string                `json:"creator"`
	Title             string                `json:"title"`
	AltTitle          string                `json:"alt_title"`
	Thumbnail         string                `json:"thumbnail"`
	Description       string                `json:"description"`
	Category          string                `json:"category"`
	Tags              []string              `json:"tags"`
	Subtitles         map[string][]Subtitle `json:"subtitles"`
	subLock           sync.Mutex
	AutomaticCaptions string   `json:"automatic_captions"`
	Duration          float64  `json:"duration"`
	AgeLimit          float64  `json:"age_limit"`
	Annotations       string   `json:"annotations"`
	Chapters          string   `json:"chapters"`
	WebpageURL        string   `json:"webpage_url"`
	ViewCount         float64  `json:"view_count"`
	LikeCount         float64  `json:"like_count"`
	DislikeCount      float64  `json:"dislike_count"`
	AverageRating     float64  `json:"average_rating"`
	Formats           []Format `json:"formats"`
}

type Subtitle struct {
	URL string `json:"url"`
	Ext string `json:"ext"`
}

// Format structure for all different formats informations
type Format struct {
	FormatID          string          `json:"format_id"`
	URL               string          `json:"url"`
	PlayerURL         string          `json:"player_URL"`
	Ext               string          `json:"ext"`
	Height            float64         `json:"height"`
	FormatNote        string          `json:"format_note"`
	Acodec            string          `json:"acodec"`
	Abr               float64         `json:"abr"`
	FileSize          float64         `json:"filesize"`
	Tbr               float64         `json:"tbr"`
	Width             float64         `json:"width"`
	Fps               float64         `json:"fps"`
	Vcodec            string          `json:"vcodec"`
	DownloaderOptions []DownloaderOpt `json:"downloader_options"`
	Format            string          `json:"format"`
	Protocol          string          `json:"protocol"`
	HTTPHeader        []HTTPHeaders   `json:"http_headers"`
}

// DownloaderOpt structure containing downloader options
type DownloaderOpt struct {
	HttpChunkSize float64 `json:"http_chunk_size"`
}

// HTTPHeaders containing the HTTPHeader for each format of video
type HTTPHeaders struct {
	UserAgent      string `json:"User-Agent"`
	AcceptCharset  string `json:"Accept-Charset"`
	Accept         string `json:"Accept"`
	AcceptEncoding string `json:"Accept-Encoding"`
	AcceptLanguage string `json:"Accept-Language"`
}

func fetchAnnotations(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	// requesting annotations from YouTube
	resp, err := http.Get("http://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + video.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			runtime.Goexit()
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
		runtime.Goexit()
	}
	defer annotationsFile.Close()
	// write description
	descriptionFile, errDescription := os.Create(video.Path + video.ID + "_" + video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		runtime.Goexit()
	}
	defer descriptionFile.Close()
	// write info json file
	infoFile, errInfo := os.Create(video.Path + video.ID + "_" + video.Title + ".info.json")
	if errInfo != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errInfo)
		runtime.Goexit()
	}
	defer infoFile.Close()
	fmt.Fprintf(annotationsFile, "%s", video.Annotations)
	fmt.Fprintf(descriptionFile, "%s", video.Description)
	JSON, _ := JsonMarshalIndentNoEscapeHTML(video.InfoJSON, "", "  ")
	fmt.Fprintf(infoFile, "%s", string(JSON))
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

func parseDescription(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract description
	video.Description = strings.TrimSpace(document.Find("#eow-description").Text())
}

func parseUploaderInfo(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("a").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "yt-uix-sessionlink       spf-link " {
			uploader := s.Text()
			if strings.Contains(uploader, "https://www.youtube.com/watch?v=") == false {
				video.InfoJSON.Uploader = uploader
			}
			uploaderID, _ := s.Attr("href")
			if strings.Contains(uploaderID, "/channel/") == true {
				video.InfoJSON.UploaderID = uploaderID[9:len(uploaderID)]
				video.InfoJSON.UploaderURL = "https://www.youtube.com" + uploaderID
			}
		}
	})
}

func parseLikeDislike(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("button").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "yt-uix-button yt-uix-button-size-default yt-uix-button-opacity yt-uix-button-has-icon no-icon-markup like-button-renderer-like-button like-button-renderer-like-button-clicked yt-uix-button-toggled  hid yt-uix-tooltip" {
			likeCount := s.Text()
			video.InfoJSON.LikeCount = cast.ToFloat64(replacer.Replace(likeCount, ""))
		}
	})
	document.Find("button").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "yt-uix-button yt-uix-button-size-default yt-uix-button-opacity yt-uix-button-has-icon no-icon-markup like-button-renderer-dislike-button like-button-renderer-dislike-button-unclicked yt-uix-clickcard-target   yt-uix-tooltip" {
			dislikeCount := s.Text()
			video.InfoJSON.DislikeCount = cast.ToFloat64(replacer.Replace(dislikeCount, ""))
		}
	})
}

func parseDatePublished(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("itemprop"); name == "datePublished" {
			date, _ := s.Attr("content")
			date = strings.Replace(date, "-", "", -1)
			video.InfoJSON.UploadDate = date
		}
	})
}

func parseViewCount(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("div").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "watch-view-count" {
			viewCount := s.Text()
			reg, err := regexp.Compile("[^0-9]+")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				runtime.Goexit()
			}
			video.InfoJSON.ViewCount = cast.ToFloat64(reg.ReplaceAllString(viewCount, ""))
		}
	})
}

func parseAverageRating(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	/*document.Find("script ").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "avg_rating") == true {
			ytPlayer := s.Text()
			pattern := regexp.MustCompile(`\(([^\)]+)\)`) // anything in parentheses
			match := pattern.FindAllStringSubmatch(ytPlayer, 1)
			fmt.Printf("%#v\n", match) // see the structure of what is being returned
			fmt.Println("result: ", match[0][1])
		}
	})*/
}

func parseTags(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:video:tag" {
			tag, _ := s.Attr("content")
			video.InfoJSON.Tags = append(video.InfoJSON.Tags, tag)
		}
	})
}

func parseCategory(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	pattern, _ := regexp.Compile(`(?s)<h4[^>]*>\s*Category\s*</h4>\s*<ul[^>]*>(.*?)</ul>`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
	if err != nil {
		panic(err)
	}
	video.InfoJSON.Category = doc.Find("a").Text()
}

func parseLicense(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	pattern, _ := regexp.Compile(`<h4[^>]+class="title"[^>]*>\s*License\s*</h4>\s*<ul[^>]*>\s*<li>(.+?)</li`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			panic(err)
		}
		video.InfoJSON.License = doc.Find("a").Text()
	}
}

func parseVariousInfo(video *Video, document *goquery.Document) {
	var wg sync.WaitGroup
	wg.Add(8)
	video.InfoJSON.ID = video.ID
	video.InfoJSON.Description = video.Description
	video.InfoJSON.Annotations = video.Annotations
	video.InfoJSON.Thumbnail = video.Thumbnail
	video.InfoJSON.WebpageURL = "https://www.youtube.com/watch?v=" + video.ID
	go parseUploaderInfo(video, document, &wg)
	go parseLikeDislike(video, document, &wg)
	go parseDatePublished(video, document, &wg)
	go parseLicense(video, document, &wg)
	go parseViewCount(video, document, &wg)
	go parseAverageRating(video, document, &wg)
	go parseTags(video, document, &wg)
	go parseCategory(video, document, &wg)
	wg.Wait()
}

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

func parseTitle(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract title
	title := strings.TrimSpace(document.Find("#eow-title").Text())
	video.InfoJSON.Title = title
	video.Title = strings.Replace(title, " ", "_", -1)
	video.Title = strings.Replace(video.Title, "/", "_", -1)
}

func parseHTML(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	var workers sync.WaitGroup
	workers.Add(3)
	// request video html page
	html, err := http.Get("http://youtube.com/watch?v=" + video.ID + "&gl=US&hl=en&has_verified=1&bpctr=9999999999")
	if err != nil {
		log.Fatalf("Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	// check status, exit if != 200
	if html.StatusCode != 200 {
		log.Fatalf("Status code error for %s: %d %s", video.ID, html.StatusCode, html.Status)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	body, err := ioutil.ReadAll(html.Body)
	// store raw html in video struct
	video.RawHTML = string(body)
	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	go parseTitle(video, document, &workers)
	go parseDescription(video, document, &workers)
	go parseThumbnailURL(video, document, &workers)
	workers.Wait()
	parseVariousInfo(video, document)
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
			runtime.Goexit()
		}
	}
}

func addSubToJSON(video *Video, langCode string) {
	urlXML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	urlTTML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=ttml&name="
	urlVTT := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=vtt&name="
	video.InfoJSON.subLock.Lock()
	video.InfoJSON.Subtitles[langCode] = append(video.InfoJSON.Subtitles[langCode], Subtitle{urlXML, "xml"}, Subtitle{urlTTML, "ttml"}, Subtitle{urlVTT, "vtt"})
	video.InfoJSON.subLock.Unlock()
}

func downloadSub(video *Video, langCode string, lang string, wg *sync.WaitGroup) {
	defer wg.Done()
	addSubToJSON(video, langCode)
	color.Green("Downloading " + lang + " subtitle.." + "[" + langCode + "]")
	// generate subtitle URL
	url := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading ") + color.Yellow(lang) + color.Green(" subtitle.."))
	// get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + "." + langCode + ".xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer out.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
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

func processSingleID(ID string, worker *sync.WaitGroup) {
	defer worker.Done()
	var wg sync.WaitGroup
	video := new(Video)
	video.ID = ID
	video.InfoJSON.Subtitles = make(map[string][]Subtitle)
	color.Println(color.Yellow("[") + color.Green("-") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Archiving started."))
	wg.Add(2)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching annotations.."))
	go fetchAnnotations(video, &wg)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Parsing infos, description, title and thumbnail.."))
	go parseHTML(video, &wg)
	wg.Wait()
	genPath(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Fetching subtitles.."))
	fetchSubsList(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Writing informations locally.."))
	writeFiles(video)
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading thumbnail.."))
	downloadThumbnail(video)
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

func processList(path string, worker *sync.WaitGroup) {
	defer worker.Done()
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
	// start workers group
	var wg sync.WaitGroup
	wg.Add(1)
	if len(args) > 1 {
		color.Red("Usage: ./youtube-ma [ID or list of IDs]")
		os.Exit(1)
	}
	if _, err := os.Stat(args[0]); err == nil {
		go processList(args[0], &wg)
	} else {
		go processSingleID(args[0], &wg)
	}
	wg.Wait()
}

func JsonMarshalIndentNoEscapeHTML(i interface{}, prefix string, indent string) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(prefix, indent)
	err := encoder.Encode(i)
	return buf.Bytes(), err
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
